package admin

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"lylme_spage/internal/config"
	"lylme_spage/internal/logger"
	"lylme_spage/internal/models"
	"lylme_spage/internal/site"
	"lylme_spage/internal/webutil"
)

// newTestEngine 构建内存 sqlite + 真实 web/admin 模板的后台测试引擎，
// 并挂载测试专用的 /__test_login 登录入口（返回该会话的 CSRF token）。
func newTestEngine(t *testing.T) (string, *Handler) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(models.AllModels()...); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}

	// 测试运行目录为 internal/admin，回退两级得到项目根（模板/主题均在其下）
	root := filepath.Join("..", "..")
	store, err := site.NewStore(db, root)
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}
	log, err := logger.New(t.TempDir(), "error")
	if err != nil {
		t.Fatalf("创建日志失败: %v", err)
	}
	h, err := New(&config.Config{RootDir: root, AdminPath: "admin"}, store, log)
	if err != nil {
		t.Fatalf("创建后台 handler 失败: %v", err)
	}

	r := gin.New()
	r.Use(sessions.Sessions("lylme_session", cookie.NewStore([]byte("test-session-secret"))))
	h.Register(r)
	r.GET("/__test_login", func(c *gin.Context) {
		if !webutil.SetAdmin(c, adminSign(store)) {
			c.String(http.StatusInternalServerError, "login failed")
			return
		}
		c.String(http.StatusOK, webutil.CSRFToken(c))
	})

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv.URL, h
}

// loginClient 返回带登录会话 Cookie 的客户端及该会话 CSRF token
func loginClient(t *testing.T, baseURL string) (*http.Client, string) {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("创建 cookie jar 失败: %v", err)
	}
	cl := &http.Client{Jar: jar}
	resp, err := cl.Get(baseURL + "/__test_login")
	if err != nil {
		t.Fatalf("测试登录失败: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("测试登录状态码 %d", resp.StatusCode)
	}
	return cl, strings.TrimSpace(string(body))
}

// postForm 以表单方式 POST 并返回状态码与响应体
func postForm(t *testing.T, cl *http.Client, rawurl string, vals url.Values) (int, string) {
	t.Helper()
	resp, err := cl.PostForm(rawurl, vals)
	if err != nil {
		t.Fatalf("POST %s 失败: %v", rawurl, err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return resp.StatusCode, string(body)
}

// ajaxResp ajax_mcp.php 响应结构
type ajaxResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Key         string `json:"key"`
		CreatedAt   string `json:"created_at"`
		APIBase     string `json:"api_base"`
		MCPEndpoint string `json:"mcp_endpoint"`
		MCPConfig   string `json:"mcp_config"`
	} `json:"data"`
}

// TestMCPConfigJSON 配置 JSON 纯函数：结构、url 与内嵌 Authorization 断言
func TestMCPConfigJSON(t *testing.T) {
	key := "sk-abc123def456"
	cases := []struct {
		root string
		want string
	}{
		{"https://example.com/", "https://example.com/mcp"},
		{"http://localhost:8080", "http://localhost:8080/mcp"},
	}
	for _, tc := range cases {
		out := mcpConfigJSON(tc.root, key)
		var cfg mcpClientCfg
		if err := json.Unmarshal([]byte(out), &cfg); err != nil {
			t.Fatalf("配置不是合法 JSON: %v", err)
		}
		srv, ok := cfg.MCPServers[mcpServerName]
		if !ok {
			t.Fatalf("缺少 MCP 服务键 %q", mcpServerName)
		}
		if srv.URL != tc.want {
			t.Fatalf("url = %q, 期望 %q", srv.URL, tc.want)
		}
		if got := srv.Headers["Authorization"]; got != "Bearer "+key {
			t.Fatalf("Authorization = %q, 期望 Bearer %s", got, key)
		}
	}
}

// TestMCPPageLoginProtect 未登录访问页面 302 跳转登录
func TestMCPPageLoginProtect(t *testing.T) {
	baseURL, _ := newTestEngine(t)

	noRedirect := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := noRedirect.Get(baseURL + "/admin/mcp.php")
	if err != nil {
		t.Fatalf("GET mcp.php 失败: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("未登录状态码 = %d, 期望 302", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); !strings.HasSuffix(loc, "/admin/login.php") {
		t.Fatalf("跳转地址 = %q, 期望登录页", loc)
	}
}

// TestMCPPageAndAjax 登录后页面渲染 + ensure/regenerate 鉴权、CSRF 与新旧密钥行为
func TestMCPPageAndAjax(t *testing.T) {
	baseURL, h := newTestEngine(t)

	// 未登录 POST：frag 中间件输出跳转脚本，不执行密钥操作
	anon := &http.Client{}
	_, body := postForm(t, anon, baseURL+"/admin/ajax_mcp.php?set=ensure", url.Values{})
	if !strings.Contains(body, "login.php") {
		t.Fatalf("未登录 POST 应输出登录跳转脚本，实际 %q", body)
	}

	cl, token := loginClient(t, baseURL)
	endpoint := baseURL + "/admin/ajax_mcp.php"

	// 缺少 CSRF token
	_, body = postForm(t, cl, endpoint+"?set=ensure", url.Values{})
	var res ajaxResp
	if err := json.Unmarshal([]byte(body), &res); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	if res.Code != -1 {
		t.Fatalf("缺少 CSRF 时 code = %d, 期望 -1", res.Code)
	}

	// 错误 CSRF token
	_, body = postForm(t, cl, endpoint+"?set=ensure", url.Values{"csrf_token": {"wrong-token"}})
	_ = json.Unmarshal([]byte(body), &res)
	if res.Code != -1 {
		t.Fatalf("错误 CSRF 时 code = %d, 期望 -1", res.Code)
	}

	// GET 方式请求 ajax：非法请求
	resp, err := cl.Get(endpoint + "?set=ensure")
	if err != nil {
		t.Fatalf("GET ajax_mcp 失败: %v", err)
	}
	gb, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	_ = json.Unmarshal(gb, &res)
	if res.Code != -1 {
		t.Fatalf("GET ajax 时 code = %d, 期望 -1", res.Code)
	}

	// 未知 set 分支
	_, body = postForm(t, cl, endpoint+"?set=unknown", url.Values{"csrf_token": {token}})
	_ = json.Unmarshal([]byte(body), &res)
	if res.Code != -1 {
		t.Fatalf("未知 set 时 code = %d, 期望 -1", res.Code)
	}

	// ensure：首次生成
	_, body = postForm(t, cl, endpoint+"?set=ensure", url.Values{"csrf_token": {token}})
	if err := json.Unmarshal([]byte(body), &res); err != nil {
		t.Fatalf("ensure 响应不是合法 JSON: %v", err)
	}
	if res.Code != 200 {
		t.Fatalf("ensure code = %d, msg = %s", res.Code, res.Msg)
	}
	key1 := res.Data.Key
	if !strings.HasPrefix(key1, "sk-") {
		t.Fatalf("密钥应以 sk- 开头，实际 %q", key1)
	}
	if !strings.Contains(res.Data.MCPConfig, key1) {
		t.Fatal("ensure 返回的配置中应内嵌当前 key")
	}
	if !strings.HasSuffix(res.Data.APIBase, "/api/v1") || !strings.HasSuffix(res.Data.MCPEndpoint, "/mcp") {
		t.Fatalf("地址字段异常: api_base=%q mcp_endpoint=%q", res.Data.APIBase, res.Data.MCPEndpoint)
	}

	// ensure 幂等：返回同一把 key
	_, body = postForm(t, cl, endpoint+"?set=ensure", url.Values{"csrf_token": {token}})
	_ = json.Unmarshal([]byte(body), &res)
	if res.Data.Key != key1 {
		t.Fatal("ensure 不幂等：第二次返回了不同 key")
	}

	// 页面渲染：包含 key、Bearer 配置与接入地址
	pageResp, err := cl.Get(baseURL + "/admin/mcp.php")
	if err != nil {
		t.Fatalf("GET mcp.php 失败: %v", err)
	}
	pb, _ := io.ReadAll(pageResp.Body)
	_ = pageResp.Body.Close()
	if pageResp.StatusCode != http.StatusOK {
		t.Fatalf("登录后页面状态码 = %d", pageResp.StatusCode)
	}
	page := string(pb)
	if !strings.Contains(page, key1) {
		t.Fatal("页面应展示当前密钥明文")
	}
	if !strings.Contains(page, "Bearer "+key1) {
		t.Fatal("页面配置区应内嵌 Bearer 密钥")
	}
	if !strings.Contains(page, "/api/v1") || !strings.Contains(page, "接口与 MCP") {
		t.Fatal("页面应包含 API 地址与标题")
	}

	// regenerate：返回新 key，旧 key 失效
	_, body = postForm(t, cl, endpoint+"?set=regenerate", url.Values{"csrf_token": {token}})
	_ = json.Unmarshal([]byte(body), &res)
	if res.Code != 200 {
		t.Fatalf("regenerate code = %d, msg = %s", res.Code, res.Msg)
	}
	key2 := res.Data.Key
	if key2 == key1 || !strings.HasPrefix(key2, "sk-") {
		t.Fatalf("重新生成的 key 异常: %q", key2)
	}
	if !strings.Contains(res.Data.MCPConfig, key2) {
		t.Fatal("regenerate 返回的配置中应内嵌新 key")
	}

	// service 层：旧 key 失效、新 key 有效
	if h.KeySvc.Validate(key1) {
		t.Fatal("旧 key 应已失效")
	}
	if !h.KeySvc.Validate(key2) {
		t.Fatal("新 key 应校验通过")
	}

	// 页面与新 key 一致，且不再包含旧 key
	pageResp2, err := cl.Get(baseURL + "/admin/mcp.php")
	if err != nil {
		t.Fatalf("再次 GET mcp.php 失败: %v", err)
	}
	pb2, _ := io.ReadAll(pageResp2.Body)
	_ = pageResp2.Body.Close()
	page2 := string(pb2)
	if !strings.Contains(page2, key2) {
		t.Fatal("页面应展示新 key")
	}
	if strings.Contains(page2, key1) {
		t.Fatal("页面不应再包含旧 key")
	}
}

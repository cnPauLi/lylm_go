package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"lylme_spage/internal/models"
	"lylme_spage/internal/service"
	"lylme_spage/internal/site"
)

// newThemeAPIEnv 构建主题域集成测试环境：
// 内存 SQLite（全模型迁移）+ 指向真实项目根（filepath.Join("..","..")）
// 的 site.Store，确保主题管理器能发现真实主题；简化 Deps 仅装配
// Keys/Theme，中间件关闭限流但保留 401 鉴权。
// 返回引擎、store（用于清理）与可用密钥。
func newThemeAPIEnv(t *testing.T) (*gin.Engine, *site.Store, string) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(models.AllModels()...); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	// go test 以包目录 internal/api 为 CWD，../../ 即项目根
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("解析项目根失败: %v", err)
	}
	store, err := site.NewStore(db, root)
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}
	keys := service.NewAPIKeyService(store)
	key, err := keys.Ensure()
	if err != nil {
		t.Fatalf("Ensure key 失败: %v", err)
	}
	d := &Deps{Keys: keys, Theme: service.NewThemeService(store)}
	r := gin.New()
	v1 := r.Group("/api/v1", RequireKeyWithLimit(keys, 0, 0))
	RegisterThemeRoutes(v1, d)
	return r, store, key
}

// themeDo 发起一次 HTTP 请求；key 为空表示不携带 Authorization
func themeDo(t *testing.T, r *gin.Engine, method, path, key, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// themeData 将统一信封的 data 反序列化为 map 便于断言
func themeData(t *testing.T, resp Response) gin.H {
	t.Helper()
	b, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatalf("data 序列化失败: %v", err)
	}
	var m gin.H
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("data 非对象: %v", err)
	}
	return m
}

func TestThemeList(t *testing.T) {
	r, _, key := newThemeAPIEnv(t)

	w := themeDo(t, r, http.MethodGet, "/api/v1/themes", key, "")
	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, want 200；body=%s", w.Code, w.Body.String())
	}
	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应非 JSON: %v", err)
	}
	if resp.Code != CodeOK {
		t.Fatalf("code = %d, want 200", resp.Code)
	}
	data := themeData(t, resp)
	if data["current"] != "default" {
		t.Fatalf("current = %v, want default", data["current"])
	}
	rawThemes, ok := data["themes"].([]any)
	if !ok || len(rawThemes) == 0 {
		t.Fatalf("themes 应为非空数组，实际 %T", data["themes"])
	}
	var foundDefault bool
	for _, rt := range rawThemes {
		item, ok := rt.(map[string]any)
		if !ok {
			t.Fatalf("主题项非对象: %T", rt)
		}
		for _, k := range []string{"dir", "name", "author", "version", "description"} {
			if _, exists := item[k]; !exists {
				t.Fatalf("主题项缺少字段 %s: %+v", k, item)
			}
		}
		if item["dir"] == "default" {
			foundDefault = true
			if item["name"] != "default" {
				t.Fatalf("default 主题 name = %v, want default", item["name"])
			}
		}
	}
	if !foundDefault {
		t.Fatal("主题列表未包含 default")
	}

	// 不带 key → 401
	w = themeDo(t, r, http.MethodGet, "/api/v1/themes", "", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("无凭证状态码 = %d, want 401", w.Code)
	}
}

func TestThemeSwitchCurrent(t *testing.T) {
	r, store, key := newThemeAPIEnv(t)
	themeSvc := service.NewThemeService(store)

	// 无论测试中途如何切换，结束后恢复 default
	t.Cleanup(func() {
		if err := themeSvc.Switch("default"); err != nil {
			t.Errorf("清理：恢复 default 主题失败: %v", err)
		}
	})

	// 切换到 default 成功
	w := themeDo(t, r, http.MethodPut, "/api/v1/themes/current", key, `{"name":"default"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("切回 default 状态码 = %d, want 200；body=%s", w.Code, w.Body.String())
	}
	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应非 JSON: %v", err)
	}
	if data := themeData(t, resp); data["current"] != "default" {
		t.Fatalf("current = %v, want default", data["current"])
	}

	// 切换不存在的主题 → 404
	w = themeDo(t, r, http.MethodPut, "/api/v1/themes/current", key, `{"name":"no-such-theme"}`)
	if w.Code != http.StatusNotFound {
		t.Fatalf("不存在主题状态码 = %d, want 404；body=%s", w.Code, w.Body.String())
	}

	// name 缺失/为空 → 400
	w = themeDo(t, r, http.MethodPut, "/api/v1/themes/current", key, `{"name":""}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("空 name 状态码 = %d, want 400", w.Code)
	}
	w = themeDo(t, r, http.MethodPut, "/api/v1/themes/current", key, `{}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("缺 name 状态码 = %d, want 400", w.Code)
	}

	// 切换到真实存在的 5iux 主题，GET current 应变化
	w = themeDo(t, r, http.MethodPut, "/api/v1/themes/current", key, `{"name":"5iux"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("切换 5iux 状态码 = %d, want 200；body=%s", w.Code, w.Body.String())
	}
	w = themeDo(t, r, http.MethodGet, "/api/v1/themes", key, "")
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应非 JSON: %v", err)
	}
	if data := themeData(t, resp); data["current"] != "5iux" {
		t.Fatalf("切换后 current = %v, want 5iux", data["current"])
	}

	// 不带 key → 401
	w = themeDo(t, r, http.MethodPut, "/api/v1/themes/current", "", `{"name":"default"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("无凭证状态码 = %d, want 401", w.Code)
	}
}

func TestThemeConfigGet(t *testing.T) {
	r, _, key := newThemeAPIEnv(t)

	w := themeDo(t, r, http.MethodGet, "/api/v1/themes/config", key, "")
	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, want 200；body=%s", w.Code, w.Body.String())
	}
	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应非 JSON: %v", err)
	}
	data := themeData(t, resp)
	if data["theme"] != "default" {
		t.Fatalf("theme = %v, want default", data["theme"])
	}
	fields, ok := data["fields"].([]any)
	if !ok {
		t.Fatalf("fields 应为数组，实际 %T", data["fields"])
	}
	if len(fields) == 0 {
		t.Fatal("default 主题 fields 不应为空")
	}
	for _, rf := range fields {
		f, ok := rf.(map[string]any)
		if !ok {
			t.Fatalf("字段定义非对象: %T", rf)
		}
		if f["name"] == "" || f["type"] == "" {
			t.Fatalf("字段 name/type 不能为空: %+v", f)
		}
		if _, exists := f["enum"]; !exists {
			t.Fatalf("字段缺少 enum 键: %+v", f)
		}
	}
	// values 必须为对象（未保存过时为空对象，不允许 null）
	values, ok := data["values"].(map[string]any)
	if !ok {
		t.Fatalf("values 应为对象，实际 %T", data["values"])
	}
	if len(values) != 0 {
		t.Fatalf("未保存过配置时 values 应为空对象，实际 %+v", values)
	}

	// 不带 key → 401
	w = themeDo(t, r, http.MethodGet, "/api/v1/themes/config", "", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("无凭证状态码 = %d, want 401", w.Code)
	}
}

func TestThemeConfigSave(t *testing.T) {
	r, store, key := newThemeAPIEnv(t)

	// 结束后删除测试写入的配置行，恢复「未保存」原状
	t.Cleanup(func() {
		if err := store.DB().Where("k = ?", "theme_config_default").
			Delete(&models.Config{}).Error; err != nil {
			t.Errorf("清理 theme_config_default 失败: %v", err)
		}
		if err := store.Reload(); err != nil {
			t.Errorf("清理后 Reload 失败: %v", err)
		}
	})

	// values 包裹形式保存
	body := `{"theme":"default","values":{"zz_api_test_key":"hello"}}`
	w := themeDo(t, r, http.MethodPut, "/api/v1/themes/config", key, body)
	if w.Code != http.StatusOK {
		t.Fatalf("保存配置状态码 = %d, want 200；body=%s", w.Code, w.Body.String())
	}
	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应非 JSON: %v", err)
	}
	data := themeData(t, resp)
	if data["theme"] != "default" {
		t.Fatalf("响应 theme = %v, want default", data["theme"])
	}
	savedVals, ok := data["values"].(map[string]any)
	if !ok || savedVals["zz_api_test_key"] != "hello" {
		t.Fatalf("响应 values 未包含保存键: %+v", data["values"])
	}

	// GET 读回
	w = themeDo(t, r, http.MethodGet, "/api/v1/themes/config", key, "")
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应非 JSON: %v", err)
	}
	data = themeData(t, resp)
	values, ok := data["values"].(map[string]any)
	if !ok || values["zz_api_test_key"] != "hello" {
		t.Fatalf("GET values 未读回测试键: %+v", data["values"])
	}

	// 平铺形式保存（全量替换语义）；theme 可空，回退当前主题
	w = themeDo(t, r, http.MethodPut, "/api/v1/themes/config", key, `{"theme":"","zz_api_test_key2":"world"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("平铺保存状态码 = %d, want 200；body=%s", w.Code, w.Body.String())
	}
	w = themeDo(t, r, http.MethodGet, "/api/v1/themes/config", key, "")
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应非 JSON: %v", err)
	}
	values = themeData(t, resp)["values"].(map[string]any)
	if values["zz_api_test_key2"] != "world" {
		t.Fatalf("平铺键未读回: %+v", values)
	}
	if _, exists := values["zz_api_test_key"]; exists {
		t.Fatal("保存应为全量替换，旧测试键应消失")
	}

	// 非法 values 类型 → 400
	w = themeDo(t, r, http.MethodPut, "/api/v1/themes/config", key, `{"values":123}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("非法 values 状态码 = %d, want 400", w.Code)
	}
	// 畸形 JSON → 400
	w = themeDo(t, r, http.MethodPut, "/api/v1/themes/config", key, `{not-json`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("畸形 JSON 状态码 = %d, want 400", w.Code)
	}

	// 不带 key → 401
	w = themeDo(t, r, http.MethodPut, "/api/v1/themes/config", "", body)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("无凭证状态码 = %d, want 401", w.Code)
	}
}

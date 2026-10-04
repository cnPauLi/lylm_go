package mcp

import (
	"bytes"
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

	"lylme_spage/internal/api"
	"lylme_spage/internal/config"
	"lylme_spage/internal/models"
	"lylme_spage/internal/service"
	"lylme_spage/internal/site"
)

// expectedToolNames 全部 56 个工具名，按域排列；
// MCP 工具集必须与该清单逐一吻合（即与 /api/v1 端点一一对应）。
var expectedToolNames = []string{
	// group 6
	"group_list", "group_create", "group_update", "group_delete", "group_set_status", "group_sort",
	// link 11
	"link_list", "link_all", "link_get", "link_create", "link_update", "link_delete",
	"link_set_status", "link_move_group", "link_set_pwd", "link_sort", "link_fetch_head",
	// tag 4
	"tag_list", "tag_create", "tag_update", "tag_delete",
	// sou 4
	"sou_list", "sou_create", "sou_update", "sou_delete",
	// pwd 4
	"pwd_list", "pwd_create", "pwd_update", "pwd_delete",
	// settings 8
	"settings_get", "settings_update",
	"settings_about_get", "settings_about_update", "settings_about_reset",
	"settings_account_update", "settings_system_get", "settings_system_update",
	// 微信推送 / 授权 / API 密钥 各 2
	"wxplus_get", "wxplus_update",
	"license_get", "license_update",
	"apikey_get", "apikey_regenerate",
	// apply 6
	"apply_list", "apply_settings_get", "apply_settings_update",
	"apply_update", "apply_review", "apply_delete",
	// theme 4
	"theme_list", "theme_switch", "theme_config_get", "theme_config_save",
	// dashboard / 文件清理 3
	"dashboard_summary", "files_cleanup_scan", "files_cleanup_delete",
}

// expectedPrefixCounts 各统一前缀应有的工具数量
var expectedPrefixCounts = map[string]int{
	"group_":     6,
	"link_":      11,
	"tag_":       4,
	"sou_":       4,
	"pwd_":       4,
	"settings_":  8,
	"apply_":     6,
	"theme_":     4,
	"dashboard_": 1,
	"files_":     2,
}

// mcpEnv 集成测试环境：内存 SQLite + 真实项目根 + httptest 服务器。
type mcpEnv struct {
	t      *testing.T
	d      *api.Deps
	store  *site.Store
	key    string
	srv    *httptest.Server
	nextID int
}

// newMCPEnv 构建测试环境：
// 内存 SQLite（全模型迁移）；root 指向真实项目根（../../，go test CWD 为 internal/mcp），
// 使主题管理器能发现真实主题；MCP 端点关闭限流但保留 Bearer 鉴权。
func newMCPEnv(t *testing.T) *mcpEnv {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(models.AllModels()...); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("解析项目根失败: %v", err)
	}
	store, err := site.NewStore(db, root)
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}
	d := api.NewDeps(store, &config.Config{RootDir: root}, "test")
	key, err := d.Keys.Ensure()
	if err != nil {
		t.Fatalf("Ensure key 失败: %v", err)
	}

	r := gin.New()
	h := NewHandler(d)
	auth := api.RequireKeyWithLimit(d.Keys, 0, 0)
	forward := func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
	for _, p := range []string{"/mcp", "/mcp/"} {
		r.GET(p, auth, forward)
		r.POST(p, auth, forward)
		r.DELETE(p, auth, forward)
	}
	srv := httptest.NewServer(r)

	e := &mcpEnv{t: t, d: d, store: store, key: key, srv: srv}

	t.Cleanup(func() {
		srv.Close()
		// 恢复 default 主题，删除测试可能写入的主题配置行并重载缓存
		if err := d.Theme.Switch("default"); err != nil {
			t.Errorf("清理：恢复 default 主题失败: %v", err)
		}
		if err := store.DB().Where("k = ?", "theme_config_default").
			Delete(&models.Config{}).Error; err != nil {
			t.Errorf("清理 theme_config_default 失败: %v", err)
		}
		if err := store.Reload(); err != nil {
			t.Errorf("清理后 Reload 失败: %v", err)
		}
	})

	return e
}

// post 向 /mcp 发起一次 POST。body 为 string 时原样发送，否则按 JSON 序列化。
// key 为空表示不携带 Authorization。
func (e *mcpEnv) post(body any, key string) (int, http.Header, []byte) {
	e.t.Helper()

	var rdr io.Reader
	switch v := body.(type) {
	case string:
		rdr = strings.NewReader(v)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			e.t.Fatalf("请求序列化失败: %v", err)
		}
		rdr = bytes.NewReader(b)
	}

	req, err := http.NewRequest(http.MethodPost, e.srv.URL+"/mcp", rdr)
	if err != nil {
		e.t.Fatalf("构造请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	resp, err := e.srv.Client().Do(req)
	if err != nil {
		e.t.Fatalf("发起请求失败: %v", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		e.t.Fatalf("读取响应失败: %v", err)
	}
	return resp.StatusCode, resp.Header, raw
}

// extractPayloads 从 Streamable HTTP 响应中提取 JSON-RPC 负载：
// text/event-stream 时按 SSE 格式逐行解析（mcp-go 写出 "event: message\ndata: <json>\n\n"），
// 取每个 "data: " 前缀后的 JSON；其余 Content-Type 将整体作为单个 JSON 负载。
func extractPayloads(header http.Header, raw []byte) [][]byte {
	ct := header.Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		trimmed := bytes.TrimSpace(raw)
		if len(trimmed) == 0 {
			return nil
		}
		return [][]byte{trimmed}
	}
	var out [][]byte
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, "data:") {
			p := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if p != "" {
				out = append(out, []byte(p))
			}
		}
	}
	return out
}

// rpcRaw 发送一个带 id 的 JSON-RPC 请求，返回状态码、响应头与解析后的信封。
func (e *mcpEnv) rpcRaw(method string, params any) (int, http.Header, map[string]any) {
	e.t.Helper()

	e.nextID++
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      e.nextID,
		"method":  method,
	}
	if params != nil {
		req["params"] = params
	}

	status, header, raw := e.post(req, e.key)
	payloads := extractPayloads(header, raw)
	if len(payloads) == 0 {
		return status, header, nil
	}
	var m map[string]any
	if err := json.Unmarshal(payloads[len(payloads)-1], &m); err != nil {
		e.t.Fatalf("%s 响应不是合法 JSON: %v；body=%s", method, err, string(raw))
	}
	return status, header, m
}

// rpc 发送一个 JSON-RPC 请求，要求 HTTP 200 且信封无 error，返回 result。
func (e *mcpEnv) rpc(method string, params any) map[string]any {
	e.t.Helper()

	status, _, m := e.rpcRaw(method, params)
	if status != http.StatusOK {
		e.t.Fatalf("%s 返回 HTTP %d（want 200）", method, status)
	}
	if m == nil {
		e.t.Fatalf("%s 无响应负载", method)
	}
	if rpcErr, ok := m["error"]; ok {
		e.t.Fatalf("%s 返回 JSON-RPC 错误: %v", method, rpcErr)
	}
	result, ok := m["result"].(map[string]any)
	if !ok {
		e.t.Fatalf("%s 的 result 非对象: %T", method, m["result"])
	}
	return result
}

// notify 发送一个无 id 的 JSON-RPC 通知，要求 2xx（initialized 通常为 202）。
func (e *mcpEnv) notify(method string, params any) {
	e.t.Helper()

	req := map[string]any{"jsonrpc": "2.0", "method": method}
	if params != nil {
		req["params"] = params
	}
	status, _, raw := e.post(req, e.key)
	if status < 200 || status >= 300 {
		e.t.Fatalf("通知 %s 返回 HTTP %d；body=%s", method, status, string(raw))
	}
}

// handshake 完成 legacy 握手：initialize → notifications/initialized。
// 断言协商版本、tools 能力，并验证无状态模式不签发会话 ID。
func (e *mcpEnv) handshake() {
	e.t.Helper()

	status, header, m := e.rpcRaw("initialize", map[string]any{
		"protocolVersion": "2025-11-25",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "go-test", "version": "1.0"},
	})
	if status != http.StatusOK {
		e.t.Fatalf("initialize 返回 HTTP %d", status)
	}
	if sid := header.Get("Mcp-Session-Id"); sid != "" {
		e.t.Fatalf("无状态模式不应返回 Mcp-Session-Id，实际 %q", sid)
	}
	result, ok := m["result"].(map[string]any)
	if !ok {
		e.t.Fatalf("initialize result 非对象: %v", m)
	}
	if result["protocolVersion"] != "2025-11-25" {
		e.t.Fatalf("协商版本 = %v, want 2025-11-25", result["protocolVersion"])
	}
	caps, ok := result["capabilities"].(map[string]any)
	if !ok {
		e.t.Fatalf("initialize capabilities 非对象: %v", result["capabilities"])
	}
	if _, ok := caps["tools"]; !ok {
		e.t.Fatal("服务端未声明 tools 能力")
	}

	e.notify("notifications/initialized", map[string]any{})
}

// toolOutcome 为 tools/call 的提取结果。
type toolOutcome struct {
	Text    string
	IsError bool
}

// callTool 调用一个 MCP 工具并提取首个 TextContent 文本与 isError 标记。
func (e *mcpEnv) callTool(name string, args map[string]any) toolOutcome {
	e.t.Helper()

	params := map[string]any{"name": name}
	if args != nil {
		params["arguments"] = args
	}
	result := e.rpc("tools/call", params)

	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		e.t.Fatalf("%s 返回 content 非非空数组: %v", name, result["content"])
	}
	first, ok := content[0].(map[string]any)
	if !ok {
		e.t.Fatalf("%s content[0] 非对象: %T", name, content[0])
	}
	text, _ := first["text"].(string)
	isErr, _ := result["isError"].(bool)
	return toolOutcome{Text: text, IsError: isErr}
}

// mustParseJSON 断言工具文本为合法 JSON，返回解析结果。
func mustParseJSON(t *testing.T, name, text string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(text), &v); err != nil {
		t.Fatalf("%s 返回文本不是合法 JSON: %v；text=%s", name, err, text)
	}
	return v
}

// ----------------------- 测试用例 -----------------------

// TestMCPRequiresKey：无 key 与错误 key 的请求必须 401。
func TestMCPRequiresKey(t *testing.T) {
	e := newMCPEnv(t)
	body := `{"jsonrpc":"2.0","id":1,"method":"initialize",` +
		`"params":{"protocolVersion":"2025-11-25","capabilities":{},` +
		`"clientInfo":{"name":"x","version":"1"}}}`

	for _, key := range []string{"", "sk-wrong-key"} {
		status, _, raw := e.post(body, key)
		if status != http.StatusUnauthorized {
			t.Fatalf("key=%q 状态码 = %d, want 401；body=%s", key, status, string(raw))
		}
	}
}

// TestMCPToolsList：握手后工具清单必须与 56 个期望工具名逐一吻合，
// 每个工具带 inputSchema，且各前缀计数正确。
func TestMCPToolsList(t *testing.T) {
	e := newMCPEnv(t)
	e.handshake()

	result := e.rpc("tools/list", map[string]any{})
	rawTools, ok := result["tools"].([]any)
	if !ok {
		t.Fatalf("tools/list tools 非数组: %T", result["tools"])
	}
	if len(rawTools) != len(expectedToolNames) {
		t.Fatalf("工具数量 = %d, want %d", len(rawTools), len(expectedToolNames))
	}

	got := make(map[string]bool, len(rawTools))
	for _, rt := range rawTools {
		tool, ok := rt.(map[string]any)
		if !ok {
			t.Fatalf("工具定义非对象: %T", rt)
		}
		name, _ := tool["name"].(string)
		if name == "" {
			t.Fatalf("工具缺少 name: %v", tool)
		}
		if got[name] {
			t.Fatalf("工具名重复: %s", name)
		}
		if _, ok := tool["inputSchema"]; !ok {
			t.Fatalf("工具 %s 缺少 inputSchema", name)
		}
		got[name] = true
	}

	for _, want := range expectedToolNames {
		if !got[want] {
			t.Fatalf("缺少工具 %q", want)
		}
	}

	for prefix, want := range expectedPrefixCounts {
		count := 0
		for name := range got {
			if strings.HasPrefix(name, prefix) {
				count++
			}
		}
		if count != want {
			t.Fatalf("前缀 %s 工具数 = %d, want %d", prefix, count, want)
		}
	}
}

// TestMCPCallEveryDomain：9 大注册域各至少 1 个成功调用（共 11 个），
// 返回文本均为可解析 JSON。
func TestMCPCallEveryDomain(t *testing.T) {
	e := newMCPEnv(t)
	e.handshake()

	// group 域：create 后 list 读回
	created := e.callTool("group_create", map[string]any{"name": "MCP分组"})
	if created.IsError {
		t.Fatalf("group_create 失败: %s", created.Text)
	}
	createdMap := mustParseJSON(t, "group_create", created.Text).(map[string]any)
	gid, _ := createdMap["id"].(float64)
	if gid <= 0 {
		t.Fatalf("group_create 未返回有效 id: %s", created.Text)
	}

	list := e.callTool("group_list", nil)
	if list.IsError {
		t.Fatalf("group_list 失败: %s", list.Text)
	}
	listMap := mustParseJSON(t, "group_list", list.Text).(map[string]any)
	groups, _ := listMap["groups"].([]any)
	if len(groups) != 1 {
		t.Fatalf("group_list groups 长度 = %d, want 1", len(groups))
	}

	// link 域
	link := e.callTool("link_create", map[string]any{
		"name": "MCP链接", "url": "https://example.com/", "group_id": gid,
	})
	if link.IsError {
		t.Fatalf("link_create 失败: %s", link.Text)
	}
	linkMap := mustParseJSON(t, "link_create", link.Text).(map[string]any)
	if linkID, _ := linkMap["id"].(float64); linkID <= 0 {
		t.Fatalf("link_create 未返回有效 id: %s", link.Text)
	}

	// tag / sou / pwd 域：列表可解析
	for _, tc := range []struct {
		tool string
	}{
		{"tag_list"}, {"sou_list"}, {"pwd_list"},
	} {
		out := e.callTool(tc.tool, nil)
		if out.IsError {
			t.Fatalf("%s 失败: %s", tc.tool, out.Text)
		}
		mustParseJSON(t, tc.tool, out.Text)
	}

	// settings 域：敏感键不得出现
	settingsOut := e.callTool("settings_get", nil)
	if settingsOut.IsError {
		t.Fatalf("settings_get 失败: %s", settingsOut.Text)
	}
	settingsMap := mustParseJSON(t, "settings_get", settingsOut.Text).(map[string]any)
	for _, k := range []string{"admin_pwd", "api_key"} {
		if _, exists := settingsMap[k]; exists {
			t.Fatalf("settings_get 泄露敏感键 %s", k)
		}
	}

	// apply 域：前台提交（service.Create）后经 MCP 审核通过
	rec, err := e.d.Apply.Create(service.ApplyCreateInput{
		Name: "MCP申请", URL: "https://apply.example.com/", GroupID: int(gid),
	})
	if err != nil {
		t.Fatalf("创建测试申请失败: %v", err)
	}
	review := e.callTool("apply_review", map[string]any{
		"id": rec.ApplyID, "status": 1,
	})
	if review.IsError {
		t.Fatalf("apply_review 失败: %s", review.Text)
	}
	reviewMap := mustParseJSON(t, "apply_review", review.Text).(map[string]any)
	if reviewMap["status"] != float64(1) {
		t.Fatalf("apply_review status = %v, want 1", reviewMap["status"])
	}

	// theme 域：current=default，themes 非空
	themesOut := e.callTool("theme_list", nil)
	if themesOut.IsError {
		t.Fatalf("theme_list 失败: %s", themesOut.Text)
	}
	themesMap := mustParseJSON(t, "theme_list", themesOut.Text).(map[string]any)
	if themesMap["current"] != "default" {
		t.Fatalf("theme_list current = %v, want default", themesMap["current"])
	}
	if ts, _ := themesMap["themes"].([]any); len(ts) == 0 {
		t.Fatal("theme_list themes 为空")
	}

	// dashboard 域
	dash := e.callTool("dashboard_summary", nil)
	if dash.IsError {
		t.Fatalf("dashboard_summary 失败: %s", dash.Text)
	}
	dashMap := mustParseJSON(t, "dashboard_summary", dash.Text).(map[string]any)
	if dashMap["app_version"] != "test" {
		t.Fatalf("app_version = %v, want test", dashMap["app_version"])
	}
	if links, _ := dashMap["links"].(float64); links < 2 {
		t.Fatalf("dashboard links = %v, want >= 2（新建链接 + 审核通过链接）", dashMap["links"])
	}

	// files 域：扫描报告可解析且含约定键
	scan := e.callTool("files_cleanup_scan", nil)
	if scan.IsError {
		t.Fatalf("files_cleanup_scan 失败: %s", scan.Text)
	}
	scanMap := mustParseJSON(t, "files_cleanup_scan", scan.Text).(map[string]any)
	for _, k := range []string{"total", "used", "unused", "total_size", "total_size_str", "dirs"} {
		if _, exists := scanMap[k]; !exists {
			t.Fatalf("files_cleanup_scan 缺少字段 %s", k)
		}
	}
}

// TestMCPParameterErrors：必填缺失、非法枚举、类型错误均返回 IsError 工具结果。
func TestMCPParameterErrors(t *testing.T) {
	e := newMCPEnv(t)
	e.handshake()

	// 必填缺失
	out := e.callTool("group_create", map[string]any{})
	if !out.IsError || out.Text == "" {
		t.Fatalf("缺 name 应返回参数错误，实际 IsError=%v text=%q", out.IsError, out.Text)
	}

	// 非法枚举
	out = e.callTool("apply_review", map[string]any{"id": 1, "status": 5})
	if !out.IsError {
		t.Fatalf("status=5 应返回参数错误，实际 text=%q", out.Text)
	}

	// 类型错误（ids 应为数组）
	out = e.callTool("link_set_status", map[string]any{"ids": "not-an-array", "status": 1})
	if !out.IsError {
		t.Fatalf("ids 为字符串应返回参数错误，实际 text=%q", out.Text)
	}
}

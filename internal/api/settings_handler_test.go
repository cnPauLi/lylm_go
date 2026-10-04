package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"lylme_spage/internal/config"
	"lylme_spage/internal/models"
	"lylme_spage/internal/service"
	"lylme_spage/internal/site"
)

// settingsDo 发起一次 JSON API 请求并返回录制结果；body 为 nil 时不带请求体。
// key 非空时携带 Bearer 鉴权头。
func settingsDo(t *testing.T, r *gin.Engine, method, path, key string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw := []byte(nil)
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("编码请求体失败: %v", err)
		}
		raw = b
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// settingsEnvelope 校验统一响应信封并返回业务码与 data 原始 JSON
func settingsEnvelope(t *testing.T, w *httptest.ResponseRecorder) (int, json.RawMessage) {
	t.Helper()
	var env struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("响应不是统一信封 JSON: %v, body=%s", err, w.Body.String())
	}
	return env.Code, env.Data
}

// settingsDataMap 将 data 解析为 map（data 为 null 时返回 nil）
func settingsDataMap(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("data 不是 JSON 对象: %v, raw=%s", err, raw)
	}
	return m
}

// TestSettingsEndpointsFlow 在同一引擎上顺序验证全部 settings 域端点。
func TestSettingsEndpointsFlow(t *testing.T) {
	r, _, key := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) {
		RegisterSettingsRoutes(v1, d)
	})

	// 未携带密钥 → 401
	if w := settingsDo(t, r, http.MethodGet, "/api/v1/settings", "", nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("无密钥访问应 401，实际 %d", w.Code)
	}

	// 1. GET /settings：敏感键缺席，且响应体不含密钥明文
	w := settingsDo(t, r, http.MethodGet, "/api/v1/settings", key, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /settings 状态码 = %d, body=%s", w.Code, w.Body.String())
	}
	code, raw := settingsEnvelope(t, w)
	if code != CodeOK {
		t.Fatalf("GET /settings code = %d", code)
	}
	conf := settingsDataMap(t, raw)
	if _, ok := conf["admin_pwd"]; ok {
		t.Fatal("GET /settings 不得返回 admin_pwd")
	}
	if _, ok := conf["api_key"]; ok {
		t.Fatal("GET /settings 不得返回 api_key")
	}
	if strings.Contains(w.Body.String(), key) {
		t.Fatal("GET /settings 响应体中泄露了当前密钥明文")
	}

	// 2. PUT /settings：全字段批量更新
	settings := map[string]any{
		"title":          "测试导航站",
		"logo":           "./logo.png",
		"background":     "./bg.jpg",
		"wap_background": "./wapbg.jpg",
		"keywords":       "导航,搜索",
		"description":    "这是一个测试站",
		"mode":           "2",
		"snapshot":       "https://snap.example.com/?url=",
		"copyright":      "© 2026 测试",
		"icp":            "京ICP备00000000号",
		"wztj":           "<script>tj</script>",
		"cdnpublic":      "https://cdn.example.com/",
		"yan":            "true",
	}
	if w := settingsDo(t, r, http.MethodPut, "/api/v1/settings", key, settings); w.Code != http.StatusOK {
		t.Fatalf("PUT /settings 应 200，实际 %d, body=%s", w.Code, w.Body.String())
	}
	w = settingsDo(t, r, http.MethodGet, "/api/v1/settings", key, nil)
	conf = settingsDataMap(t, settingsEnvelopeData(t, w))
	want := map[string]string{
		"title":          "测试导航站",
		"logo":           "./logo.png",
		"background":     "./bg.jpg",
		"wap_background": "./wapbg.jpg",
		"keywords":       "导航,搜索",
		"description":    "这是一个测试站",
		"mode":           "2",
		"snapshot":       "https://snap.example.com/?url=",
		"copyright":      "© 2026 测试",
		"icp":            "京ICP备00000000号",
		"wztj":           "<script>tj</script>",
		"cdnpublic":      "https://cdn.example.com/",
		"yan":            "true",
	}
	for k, v := range want {
		if got, _ := conf[k].(string); got != v {
			t.Fatalf("读回配置 %s = %v，期望 %q", k, conf[k], v)
		}
	}
	// yan 非 "true" 取值经 service 归一为 "false"
	settings["yan"] = "yes"
	if w := settingsDo(t, r, http.MethodPut, "/api/v1/settings", key, settings); w.Code != http.StatusOK {
		t.Fatalf("PUT /settings(yan=yes) 应 200，实际 %d", w.Code)
	}
	w = settingsDo(t, r, http.MethodGet, "/api/v1/settings", key, nil)
	conf = settingsDataMap(t, settingsEnvelopeData(t, w))
	if got, _ := conf["yan"].(string); got != "false" {
		t.Fatalf("yan=yes 应归一为 false，实际 %q", got)
	}

	// 3. 关于页：默认 → 更新 → 缺字段 400 → 重置
	w = settingsDo(t, r, http.MethodGet, "/api/v1/settings/about", key, nil)
	if got := settingsDataMap(t, settingsEnvelopeData(t, w))["content"]; got != service.DefaultAboutContent {
		t.Fatalf("未配置时关于页应为默认内容，实际 %v", got)
	}
	customAbout := "<p>这是自定义关于页</p>"
	if w := settingsDo(t, r, http.MethodPut, "/api/v1/settings/about", key, map[string]any{"content": customAbout}); w.Code != http.StatusOK {
		t.Fatalf("PUT about 应 200，实际 %d", w.Code)
	}
	w = settingsDo(t, r, http.MethodGet, "/api/v1/settings/about", key, nil)
	if got := settingsDataMap(t, settingsEnvelopeData(t, w))["content"]; got != customAbout {
		t.Fatalf("关于页应为自定义内容，实际 %v", got)
	}
	// content 为必填：空对象 → 400
	if w := settingsDo(t, r, http.MethodPut, "/api/v1/settings/about", key, map[string]any{}); w.Code != http.StatusBadRequest {
		t.Fatalf("PUT about 缺 content 应 400，实际 %d", w.Code)
	}
	if w := settingsDo(t, r, http.MethodPost, "/api/v1/settings/about/reset", key, nil); w.Code != http.StatusOK {
		t.Fatalf("POST about/reset 应 200，实际 %d, body=%s", w.Code, w.Body.String())
	}
	w = settingsDo(t, r, http.MethodGet, "/api/v1/settings/about", key, nil)
	if got := settingsDataMap(t, settingsEnvelopeData(t, w))["content"]; got != service.DefaultAboutContent {
		t.Fatalf("重置后应恢复默认内容，实际 %v", got)
	}

	// 4. 账号：空库下旧密码必然错误 → 400；缺少 old_password → 400
	if w := settingsDo(t, r, http.MethodPut, "/api/v1/settings/account", key, map[string]any{
		"username": "admin", "old_password": "whatever", "new_password": "n1", "confirm_password": "n1",
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("旧密码错误应 400，实际 %d", w.Code)
	}
	if w := settingsDo(t, r, http.MethodPut, "/api/v1/settings/account", key, map[string]any{
		"new_password": "n1", "confirm_password": "n1",
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("缺少 old_password 应 400，实际 %d", w.Code)
	}

	// 5. 系统设置：非法 admin_path / 非法 debug → 400；合法（含 trim）→ 200 并可读回
	if w := settingsDo(t, r, http.MethodPut, "/api/v1/settings/system", key, map[string]any{
		"admin_path": "a/b", "debug": "1",
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("非法 admin_path 应 400，实际 %d", w.Code)
	}
	if w := settingsDo(t, r, http.MethodPut, "/api/v1/settings/system", key, map[string]any{
		"admin_path": "panel", "debug": "3",
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("非法 debug 应 400，实际 %d", w.Code)
	}
	if w := settingsDo(t, r, http.MethodPut, "/api/v1/settings/system", key, map[string]any{
		"admin_path": "  panel-1  ", "debug": "1",
	}); w.Code != http.StatusOK {
		t.Fatalf("合法系统设置应 200，实际 %d, body=%s", w.Code, w.Body.String())
	}
	w = settingsDo(t, r, http.MethodGet, "/api/v1/settings/system", key, nil)
	sys := settingsDataMap(t, settingsEnvelopeData(t, w))
	if got, _ := sys["admin_path"].(string); got != "panel-1" {
		t.Fatalf("admin_path 读回 = %v，期望 panel-1", sys["admin_path"])
	}
	if got, _ := sys["debug"].(string); got != "1" {
		t.Fatalf("debug 读回 = %v，期望 1", sys["debug"])
	}

	// 6. 微信推送：读取空配置 → 更新 → 读回
	w = settingsDo(t, r, http.MethodGet, "/api/v1/wxplus", key, nil)
	wp := settingsDataMap(t, settingsEnvelopeData(t, w))
	if got, _ := wp["key"].(string); got != "" {
		t.Fatalf("初始 wxplus.key 应为空，实际 %q", got)
	}
	if w := settingsDo(t, r, http.MethodPut, "/api/v1/wxplus", key, map[string]any{
		"key": "WXSECRET", "time": "08:30",
	}); w.Code != http.StatusOK {
		t.Fatalf("PUT wxplus 应 200，实际 %d", w.Code)
	}
	w = settingsDo(t, r, http.MethodGet, "/api/v1/wxplus", key, nil)
	wp = settingsDataMap(t, settingsEnvelopeData(t, w))
	if got, _ := wp["key"].(string); got != "WXSECRET" {
		t.Fatalf("wxplus.key 读回 = %v，期望 WXSECRET", wp["key"])
	}
	if got, _ := wp["time"].(string); got != "08:30" {
		t.Fatalf("wxplus.time 读回 = %v，期望 08:30", wp["time"])
	}

	// 7. 授权码：默认 hostmode=2 → 更新 → 非法 hostmode 400 → 读回
	w = settingsDo(t, r, http.MethodGet, "/api/v1/license", key, nil)
	lic := settingsDataMap(t, settingsEnvelopeData(t, w))
	if got, _ := lic["hostmode"].(string); got != "2" {
		t.Fatalf("初始 hostmode = %v，期望 2", lic["hostmode"])
	}
	if w := settingsDo(t, r, http.MethodPut, "/api/v1/license", key, map[string]any{
		"code": "LIC-123", "hostmode": "1",
	}); w.Code != http.StatusOK {
		t.Fatalf("PUT license 应 200，实际 %d", w.Code)
	}
	if w := settingsDo(t, r, http.MethodPut, "/api/v1/license", key, map[string]any{
		"code": "LIC-123", "hostmode": "9",
	}); w.Code != http.StatusBadRequest {
		t.Fatalf("非法 hostmode 应 400，实际 %d", w.Code)
	}
	w = settingsDo(t, r, http.MethodGet, "/api/v1/license", key, nil)
	lic = settingsDataMap(t, settingsEnvelopeData(t, w))
	if got, _ := lic["code"].(string); got != "LIC-123" {
		t.Fatalf("license.code 读回 = %v，期望 LIC-123", lic["code"])
	}
	if got, _ := lic["hostmode"].(string); got != "1" {
		t.Fatalf("license.hostmode 读回 = %v，期望 1", lic["hostmode"])
	}

	// 8. GET /apikey：仅掩码与创建时间，无明文
	w = settingsDo(t, r, http.MethodGet, "/api/v1/apikey", key, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /apikey 应 200，实际 %d", w.Code)
	}
	info := settingsDataMap(t, settingsEnvelopeData(t, w))
	if exists, _ := info["exists"].(bool); !exists {
		t.Fatal("exists 应为 true")
	}
	createdAt, _ := info["created_at"].(string)
	if _, err := time.Parse(time.RFC3339, createdAt); err != nil {
		t.Fatalf("created_at 应为 RFC3339，实际 %q: %v", createdAt, err)
	}
	masked, _ := info["masked"].(string)
	if !strings.HasPrefix(masked, "sk-") || !strings.HasSuffix(masked, "…") {
		t.Fatalf("masked 形如 sk-xxxx…，实际 %q", masked)
	}
	if masked == key || strings.Contains(w.Body.String(), key) {
		t.Fatal("GET /apikey 不得回读完整明文")
	}

	// 9. POST /apikey/regenerate：新密钥可用、旧密钥立即失效
	oldKey := key
	w = settingsDo(t, r, http.MethodPost, "/api/v1/apikey/regenerate", oldKey, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("regenerate 应 200，实际 %d, body=%s", w.Code, w.Body.String())
	}
	regen := settingsDataMap(t, settingsEnvelopeData(t, w))
	newKey, _ := regen["key"].(string)
	if !strings.HasPrefix(newKey, "sk-") || newKey == oldKey {
		t.Fatalf("新密钥异常: %q", newKey)
	}
	newCreatedAt, _ := regen["created_at"].(string)
	if _, err := time.Parse(time.RFC3339, newCreatedAt); err != nil {
		t.Fatalf("regenerate created_at 应为 RFC3339: %v", err)
	}
	// 新密钥请求 200
	if w := settingsDo(t, r, http.MethodGet, "/api/v1/apikey", newKey, nil); w.Code != http.StatusOK {
		t.Fatalf("新密钥请求应 200，实际 %d", w.Code)
	}
	// 旧密钥立即失效 → 401
	if w := settingsDo(t, r, http.MethodGet, "/api/v1/apikey", oldKey, nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("旧密钥应立即失效返回 401，实际 %d", w.Code)
	}
}

// newSettingsEnvWithStore 与 newTestV1 同构，但额外返回 store，
// 以便预置管理员账号等初始数据（newTestV1 不暴露内部 store）。
func newSettingsEnvWithStore(t *testing.T) (*gin.Engine, *Deps, string, *site.Store) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(models.AllModels()...); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	root := t.TempDir()
	store, err := site.NewStore(db, root)
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}
	d := NewDeps(store, &config.Config{RootDir: root}, "test")
	key, err := d.Keys.Ensure()
	if err != nil {
		t.Fatalf("Ensure key 失败: %v", err)
	}
	r := gin.New()
	v1 := r.Group("/api/v1", RequireKeyWithLimit(d.Keys, 0, 0))
	RegisterSettingsRoutes(v1, d)
	return r, d, key, store
}

// TestSettingsAccountFlow 管理员账号：错误密码/不一致/无更改均 400，正确改密后旧密码失效。
func TestSettingsAccountFlow(t *testing.T) {
	r, _, key, store := newSettingsEnvWithStore(t)
	if err := store.Set("admin_user", "admin", ""); err != nil {
		t.Fatal(err)
	}
	if err := store.Set("admin_pwd", service.AdminPasswordHash("oldpass1"), ""); err != nil {
		t.Fatal(err)
	}

	accountPut := func(body map[string]any) *httptest.ResponseRecorder {
		return settingsDo(t, r, http.MethodPut, "/api/v1/settings/account", key, body)
	}

	// 旧密码错误 → 400
	if w := accountPut(map[string]any{"old_password": "wrong", "new_password": "n1", "confirm_password": "n1"}); w.Code != http.StatusBadRequest {
		t.Fatalf("旧密码错误应 400，实际 %d", w.Code)
	}
	// 两次新密码不一致 → 400
	if w := accountPut(map[string]any{"old_password": "oldpass1", "new_password": "a", "confirm_password": "b"}); w.Code != http.StatusBadRequest {
		t.Fatalf("两次不一致应 400，实际 %d", w.Code)
	}
	// 未做出更改 → 400
	if w := accountPut(map[string]any{"old_password": "oldpass1"}); w.Code != http.StatusBadRequest {
		t.Fatalf("无更改应 400，实际 %d", w.Code)
	}
	// 正确改密 → 200，changed_pwd=true
	w := accountPut(map[string]any{"old_password": "oldpass1", "new_password": "newpass2", "confirm_password": "newpass2"})
	if w.Code != http.StatusOK {
		t.Fatalf("正确改密应 200，实际 %d, body=%s", w.Code, w.Body.String())
	}
	res := settingsDataMap(t, settingsEnvelopeData(t, w))
	if changed, _ := res["changed_pwd"].(bool); !changed {
		t.Fatalf("changed_pwd 应为 true，实际 %+v", res)
	}
	if _, leaked := res["password"]; leaked {
		t.Fatal("账号修改响应不得回显新密码明文")
	}
	// 旧密码现在失效
	if w := accountPut(map[string]any{"old_password": "oldpass1"}); w.Code != http.StatusBadRequest {
		t.Fatalf("改密后旧密码应 400，实际 %d", w.Code)
	}
	// 用新密码改名 → 200
	w = accountPut(map[string]any{"username": "  boss  ", "old_password": "newpass2"})
	if w.Code != http.StatusOK {
		t.Fatalf("用新密码改名应 200，实际 %d, body=%s", w.Code, w.Body.String())
	}
	res = settingsDataMap(t, settingsEnvelopeData(t, w))
	if changed, _ := res["changed_user"].(bool); !changed {
		t.Fatalf("changed_user 应为 true，实际 %+v", res)
	}
	if got := store.Get("admin_user", ""); got != "boss" {
		t.Fatalf("admin_user 应为 boss，实际 %q", got)
	}
	// 用新密码同时改名改密 → 200，两者都为 true
	w = accountPut(map[string]any{
		"username": "admin2", "old_password": "newpass2",
		"new_password": "newpass3", "confirm_password": "newpass3",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("同时改名改密应 200，实际 %d, body=%s", w.Code, w.Body.String())
	}
	res = settingsDataMap(t, settingsEnvelopeData(t, w))
	if user, _ := res["changed_user"].(bool); !user {
		t.Fatalf("changed_user 应为 true，实际 %+v", res)
	}
	if pwd, _ := res["changed_pwd"].(bool); !pwd {
		t.Fatalf("changed_pwd 应为 true，实际 %+v", res)
	}
	if got := store.Get("admin_user", ""); got != "admin2" {
		t.Fatalf("admin_user 应为 admin2，实际 %q", got)
	}
}

// settingsEnvelopeData 仅取信封 data（对 data 结构另有要求时配合 settingsDataMap 使用）
func settingsEnvelopeData(t *testing.T, w *httptest.ResponseRecorder) json.RawMessage {
	t.Helper()
	code, raw := settingsEnvelope(t, w)
	if code != CodeOK {
		t.Fatalf("业务码 = %d, want 200, body=%s", code, w.Body.String())
	}
	return raw
}

// TestMaskAPIKey 掩码函数：保留 sk- 与前 4 位 hex，其余以 … 代替
func TestMaskAPIKey(t *testing.T) {
	full, err := service.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey 失败: %v", err)
	}
	want := full[:len("sk-")+maskedKeyHead] + "…"
	if got := maskAPIKey(full); got != want {
		t.Fatalf("maskAPIKey = %q，期望 %q", got, want)
	}
	if strings.Count(maskAPIKey(full), "…") != 1 {
		t.Fatal("掩码应恰好包含一个省略号")
	}
	if got := maskAPIKey("sk-x"); got != "sk-…" {
		t.Fatalf("过短密钥掩码 = %q，期望 sk-…", got)
	}
}

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// doTSPJSON 发起 JSON 请求；key 为空时不带 Authorization（用于 401 抽查）。
// tag/sou/pwd 三域测试共用。
func doTSPJSON(t *testing.T, r *gin.Engine, method, path, key string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("序列化请求体失败: %v", err)
		}
		reader.Reset(raw)
	}
	req := httptest.NewRequest(method, path, &reader)
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

// tspEnvelope 解析统一响应信封
func tspEnvelope(t *testing.T, w *httptest.ResponseRecorder) Response {
	t.Helper()
	var env Response
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("响应非统一信封 JSON: %v, body=%s", err, w.Body.String())
	}
	return env
}

// tspData 取信封 data 并断言为对象
func tspData(t *testing.T, env Response) map[string]any {
	t.Helper()
	m, ok := env.Data.(map[string]any)
	if !ok {
		t.Fatalf("data 不是对象: %#v", env.Data)
	}
	return m
}

// tspInt 取对象中的整数字段
func tspInt(t *testing.T, m map[string]any, field string) int {
	t.Helper()
	v, ok := m[field].(float64)
	if !ok {
		t.Fatalf("字段 %s 不是数字: %#v", field, m[field])
	}
	return int(v)
}

func TestTagAPI_CRUD(t *testing.T) {
	r, _, key := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) {
		RegisterTagRoutes(v1, d)
	})

	// 初始列表为空
	w := doTSPJSON(t, r, http.MethodGet, "/api/v1/tags", key, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /tags 状态码 = %d, want 200", w.Code)
	}
	data := tspData(t, tspEnvelope(t, w))
	if list, _ := data["list"].([]any); len(list) != 0 {
		t.Fatalf("初始 list 应为空，实际 %d 条", len(list))
	}
	if tspInt(t, data, "total") != 0 {
		t.Fatal("初始 total 应为 0")
	}

	// 新增
	w = doTSPJSON(t, r, http.MethodPost, "/api/v1/tags", key, gin.H{
		"name": "首页", "link": "/", "sort": 1, "target": 1,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("POST /tags 状态码 = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	created := tspData(t, tspEnvelope(t, w))
	id := tspInt(t, created, "tag_id")
	if id <= 0 {
		t.Fatal("tag_id 应为正整数")
	}
	if created["tag_name"] != "首页" || created["tag_link"] != "/" {
		t.Fatalf("新增记录字段异常: %#v", created)
	}
	if tspInt(t, created, "sort") != 1 || tspInt(t, created, "tag_target") != 1 {
		t.Fatalf("新增记录 sort/target 异常: %#v", created)
	}

	// 列表含 1 条
	w = doTSPJSON(t, r, http.MethodGet, "/api/v1/tags", key, nil)
	data = tspData(t, tspEnvelope(t, w))
	if list, _ := data["list"].([]any); len(list) != 1 {
		t.Fatalf("list 应有 1 条，实际 %d 条", len(list))
	}
	if tspInt(t, data, "total") != 1 {
		t.Fatal("total 应为 1")
	}

	// 修改
	w = doTSPJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/tags/%d", id), key, gin.H{
		"name": "首页导航", "link": "/home", "sort": 5, "target": 0,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("PUT /tags/:id 状态码 = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	updated := tspData(t, tspEnvelope(t, w))
	if updated["tag_name"] != "首页导航" || updated["tag_link"] != "/home" {
		t.Fatalf("修改后字段异常: %#v", updated)
	}
	if tspInt(t, updated, "sort") != 5 || tspInt(t, updated, "tag_target") != 0 {
		t.Fatalf("修改后 sort/target 异常: %#v", updated)
	}

	// 删除
	w = doTSPJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/tags/%d", id), key, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("DELETE /tags/:id 状态码 = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	deleted := tspData(t, tspEnvelope(t, w))
	if tspInt(t, deleted, "id") != id {
		t.Fatalf("删除响应 id 异常: %#v", deleted)
	}

	// 列表回归为空
	w = doTSPJSON(t, r, http.MethodGet, "/api/v1/tags", key, nil)
	data = tspData(t, tspEnvelope(t, w))
	if list, _ := data["list"].([]any); len(list) != 0 {
		t.Fatalf("删除后 list 应为空，实际 %d 条", len(list))
	}
}

func TestTagAPI_CreateBadRequest(t *testing.T) {
	r, _, key := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) {
		RegisterTagRoutes(v1, d)
	})

	// 缺少必填 link
	w := doTSPJSON(t, r, http.MethodPost, "/api/v1/tags", key, gin.H{"name": "只有名称"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("状态码 = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	env := tspEnvelope(t, w)
	if env.Code != http.StatusBadRequest {
		t.Fatalf("业务码 = %d, want 400", env.Code)
	}

	// name 超长（列长 30）
	w = doTSPJSON(t, r, http.MethodPost, "/api/v1/tags", key, gin.H{
		"name": "0123456789012345678901234567890", "link": "/x",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("name 超长时状态码 = %d, want 400", w.Code)
	}

	// target 非 0/1
	w = doTSPJSON(t, r, http.MethodPost, "/api/v1/tags", key, gin.H{
		"name": "x", "link": "/x", "target": 2,
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("target=2 时状态码 = %d, want 400", w.Code)
	}
}

func TestTagAPI_UpdateNotFound(t *testing.T) {
	r, _, key := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) {
		RegisterTagRoutes(v1, d)
	})

	w := doTSPJSON(t, r, http.MethodPut, "/api/v1/tags/99999", key, gin.H{
		"name": "x", "link": "/x",
	})
	if w.Code != http.StatusNotFound {
		t.Fatalf("状态码 = %d, want 404; body=%s", w.Code, w.Body.String())
	}
	if tspEnvelope(t, w).Code != http.StatusNotFound {
		t.Fatal("业务码应为 404")
	}
}

func TestTagAPI_DeleteNotFound(t *testing.T) {
	r, _, key := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) {
		RegisterTagRoutes(v1, d)
	})

	w := doTSPJSON(t, r, http.MethodDelete, "/api/v1/tags/99999", key, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("状态码 = %d, want 404; body=%s", w.Code, w.Body.String())
	}

	// 非数字 id → 400
	w = doTSPJSON(t, r, http.MethodDelete, "/api/v1/tags/abc", key, nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("非数字 id 状态码 = %d, want 400", w.Code)
	}
}

func TestTagAPI_Unauthorized(t *testing.T) {
	r, _, _ := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) {
		RegisterTagRoutes(v1, d)
	})

	// 不带 key
	w := doTSPJSON(t, r, http.MethodGet, "/api/v1/tags", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("状态码 = %d, want 401; body=%s", w.Code, w.Body.String())
	}
	env := tspEnvelope(t, w)
	if env.Code != http.StatusUnauthorized || env.Data != nil {
		t.Fatalf("401 信封异常: %+v", env)
	}
}

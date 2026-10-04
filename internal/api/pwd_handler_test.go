package api

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPwdAPI_CRUD(t *testing.T) {
	r, _, key := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) {
		RegisterPwdRoutes(v1, d)
	})

	// 初始列表为空
	w := doTSPJSON(t, r, http.MethodGet, "/api/v1/pwd", key, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /pwd 状态码 = %d, want 200", w.Code)
	}
	data := tspData(t, tspEnvelope(t, w))
	if list, _ := data["list"].([]any); len(list) != 0 {
		t.Fatalf("初始 list 应为空，实际 %d 条", len(list))
	}

	// 新增
	w = doTSPJSON(t, r, http.MethodPost, "/api/v1/pwd", key, gin.H{
		"name": "私密分组", "key": "secret", "ps": "访问提示",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("POST /pwd 状态码 = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	created := tspData(t, tspEnvelope(t, w))
	id := tspInt(t, created, "pwd_id")
	if id <= 0 {
		t.Fatal("pwd_id 应为正整数")
	}
	if created["pwd_name"] != "私密分组" || created["pwd_key"] != "secret" || created["pwd_ps"] != "访问提示" {
		t.Fatalf("新增记录字段异常: %#v", created)
	}

	// 列表含 1 条
	w = doTSPJSON(t, r, http.MethodGet, "/api/v1/pwd", key, nil)
	data = tspData(t, tspEnvelope(t, w))
	if list, _ := data["list"].([]any); len(list) != 1 {
		t.Fatalf("list 应有 1 条，实际 %d 条", len(list))
	}
	if tspInt(t, data, "total") != 1 {
		t.Fatal("total 应为 1")
	}

	// 修改
	w = doTSPJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/pwd/%d", id), key, gin.H{
		"name": "私密分组2", "key": "secret2", "ps": "新提示",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("PUT /pwd/:id 状态码 = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	updated := tspData(t, tspEnvelope(t, w))
	if updated["pwd_name"] != "私密分组2" || updated["pwd_key"] != "secret2" || updated["pwd_ps"] != "新提示" {
		t.Fatalf("修改后字段异常: %#v", updated)
	}

	// 删除
	w = doTSPJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/pwd/%d", id), key, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("DELETE /pwd/:id 状态码 = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if tspInt(t, tspData(t, tspEnvelope(t, w)), "id") != id {
		t.Fatal("删除响应 id 异常")
	}

	// 列表回归为空
	w = doTSPJSON(t, r, http.MethodGet, "/api/v1/pwd", key, nil)
	data = tspData(t, tspEnvelope(t, w))
	if list, _ := data["list"].([]any); len(list) != 0 {
		t.Fatalf("删除后 list 应为空，实际 %d 条", len(list))
	}
}

func TestPwdAPI_CreateBadRequest(t *testing.T) {
	r, _, key := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) {
		RegisterPwdRoutes(v1, d)
	})

	// 空请求体（name/key 均缺）
	w := doTSPJSON(t, r, http.MethodPost, "/api/v1/pwd", key, gin.H{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("状态码 = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	if tspEnvelope(t, w).Code != http.StatusBadRequest {
		t.Fatal("业务码应为 400")
	}

	// 有 name 缺 key
	w = doTSPJSON(t, r, http.MethodPost, "/api/v1/pwd", key, gin.H{"name": "无密码"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("缺 key 时状态码 = %d, want 400", w.Code)
	}
}

func TestPwdAPI_UpdateNotFound(t *testing.T) {
	r, _, key := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) {
		RegisterPwdRoutes(v1, d)
	})

	w := doTSPJSON(t, r, http.MethodPut, "/api/v1/pwd/99999", key, gin.H{
		"name": "x", "key": "x",
	})
	if w.Code != http.StatusNotFound {
		t.Fatalf("状态码 = %d, want 404; body=%s", w.Code, w.Body.String())
	}
	if tspEnvelope(t, w).Code != http.StatusNotFound {
		t.Fatal("业务码应为 404")
	}
}

func TestPwdAPI_DeleteNotFound(t *testing.T) {
	r, _, key := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) {
		RegisterPwdRoutes(v1, d)
	})

	w := doTSPJSON(t, r, http.MethodDelete, "/api/v1/pwd/99999", key, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("状态码 = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

func TestPwdAPI_Unauthorized(t *testing.T) {
	r, _, _ := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) {
		RegisterPwdRoutes(v1, d)
	})

	// DELETE 不带 key
	w := doTSPJSON(t, r, http.MethodDelete, "/api/v1/pwd/1", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("状态码 = %d, want 401; body=%s", w.Code, w.Body.String())
	}
	if tspEnvelope(t, w).Code != http.StatusUnauthorized {
		t.Fatal("业务码应为 401")
	}
}

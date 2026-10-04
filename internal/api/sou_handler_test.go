package api

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSouAPI_CRUD(t *testing.T) {
	r, _, key := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) {
		RegisterSouRoutes(v1, d)
	})

	// 初始列表为空
	w := doTSPJSON(t, r, http.MethodGet, "/api/v1/sou", key, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /sou 状态码 = %d, want 200", w.Code)
	}
	data := tspData(t, tspEnvelope(t, w))
	if list, _ := data["list"].([]any); len(list) != 0 {
		t.Fatalf("初始 list 应为空，实际 %d 条", len(list))
	}

	// 新增（st=true；order 不传，由 service 置为 记录数+1=1）
	w = doTSPJSON(t, r, http.MethodPost, "/api/v1/sou", key, gin.H{
		"alias": "baidu", "name": "百度", "hint": "请输入关键词",
		"color": "#2932e1", "link": "https://www.baidu.com/s?wd=", "st": true,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("POST /sou 状态码 = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	created := tspData(t, tspEnvelope(t, w))
	id := tspInt(t, created, "sou_id")
	if id <= 0 {
		t.Fatal("sou_id 应为正整数")
	}
	if created["sou_alias"] != "baidu" || created["sou_name"] != "百度" {
		t.Fatalf("新增记录字段异常: %#v", created)
	}
	if tspInt(t, created, "sou_st") != 1 {
		t.Fatal("st=true 时 sou_st 应为 1")
	}
	if tspInt(t, created, "sou_order") != 1 {
		t.Fatalf("新增 sou_order 应为 1，实际 %#v", created["sou_order"])
	}

	// 列表含 1 条
	w = doTSPJSON(t, r, http.MethodGet, "/api/v1/sou", key, nil)
	data = tspData(t, tspEnvelope(t, w))
	if list, _ := data["list"].([]any); len(list) != 1 {
		t.Fatalf("list 应有 1 条，实际 %d 条", len(list))
	}
	if tspInt(t, data, "total") != 1 {
		t.Fatal("total 应为 1")
	}

	// 修改：改名、停用、指定 order
	w = doTSPJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/sou/%d", id), key, gin.H{
		"alias": "baidu", "name": "百度搜索", "hint": "请输入关键词",
		"color": "#2932e1", "link": "https://www.baidu.com/s?wd=",
		"st": false, "order": 3,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("PUT /sou/:id 状态码 = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	updated := tspData(t, tspEnvelope(t, w))
	if updated["sou_name"] != "百度搜索" {
		t.Fatalf("修改后名称异常: %#v", updated)
	}
	if updated["sou_alias"] != "baidu" {
		t.Fatalf("别名不应丢失: %#v", updated)
	}
	if tspInt(t, updated, "sou_st") != 0 {
		t.Fatal("st=false 时 sou_st 应为 0")
	}
	if tspInt(t, updated, "sou_order") != 3 {
		t.Fatal("sou_order 应改为 3")
	}

	// 删除
	w = doTSPJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/sou/%d", id), key, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("DELETE /sou/:id 状态码 = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if tspInt(t, tspData(t, tspEnvelope(t, w)), "id") != id {
		t.Fatal("删除响应 id 异常")
	}

	// 列表回归为空
	w = doTSPJSON(t, r, http.MethodGet, "/api/v1/sou", key, nil)
	data = tspData(t, tspEnvelope(t, w))
	if list, _ := data["list"].([]any); len(list) != 0 {
		t.Fatalf("删除后 list 应为空，实际 %d 条", len(list))
	}
}

func TestSouAPI_CreateBadRequest(t *testing.T) {
	r, _, key := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) {
		RegisterSouRoutes(v1, d)
	})

	// 缺少必填 alias
	w := doTSPJSON(t, r, http.MethodPost, "/api/v1/sou", key, gin.H{"name": "无别名"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("缺 alias 时状态码 = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	if tspEnvelope(t, w).Code != http.StatusBadRequest {
		t.Fatal("业务码应为 400")
	}

	// 空请求体
	w = doTSPJSON(t, r, http.MethodPost, "/api/v1/sou", key, gin.H{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("空体状态码 = %d, want 400", w.Code)
	}

	// link 超长
	w = doTSPJSON(t, r, http.MethodPost, "/api/v1/sou", key, gin.H{
		"alias": "x", "name": "x",
		"link": "https://example.com/" + string(make([]byte, 260)),
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("link 超长时状态码 = %d, want 400", w.Code)
	}
}

func TestSouAPI_UpdateNotFound(t *testing.T) {
	r, _, key := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) {
		RegisterSouRoutes(v1, d)
	})

	w := doTSPJSON(t, r, http.MethodPut, "/api/v1/sou/99999", key, gin.H{
		"alias": "x", "name": "x",
	})
	if w.Code != http.StatusNotFound {
		t.Fatalf("状态码 = %d, want 404; body=%s", w.Code, w.Body.String())
	}
	if tspEnvelope(t, w).Code != http.StatusNotFound {
		t.Fatal("业务码应为 404")
	}
}

func TestSouAPI_DeleteNotFound(t *testing.T) {
	r, _, key := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) {
		RegisterSouRoutes(v1, d)
	})

	w := doTSPJSON(t, r, http.MethodDelete, "/api/v1/sou/99999", key, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("状态码 = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

func TestSouAPI_Unauthorized(t *testing.T) {
	r, _, _ := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) {
		RegisterSouRoutes(v1, d)
	})

	// POST 不带 key
	w := doTSPJSON(t, r, http.MethodPost, "/api/v1/sou", "", gin.H{
		"alias": "x", "name": "x",
	})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("状态码 = %d, want 401; body=%s", w.Code, w.Body.String())
	}
	if tspEnvelope(t, w).Code != http.StatusUnauthorized {
		t.Fatal("业务码应为 401")
	}
}

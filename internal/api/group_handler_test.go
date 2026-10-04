package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// groupEnv 统一信封（data 保留原始 JSON 便于断言）
type groupEnv struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// groupDo 发起一次 HTTP 调用；body 为空时不发送请求体。
func groupDo(t *testing.T, r groupHTTPHandler, method, path, key, body string) (*httptest.ResponseRecorder, groupEnv) {
	t.Helper()
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
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

	var env groupEnv
	if w.Body.Len() > 0 {
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatalf("响应非统一信封 JSON: %v; body=%s", err, w.Body.String())
		}
	}
	return w, env
}

// groupHTTPHandler 仅用于约束测试 helper 的入参
type groupHTTPHandler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

func TestGroupAPI(t *testing.T) {
	r, _, key := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) {
		RegisterGroupRoutes(v1, d)
	})

	// 未携带 key：401
	t.Run("未鉴权返回401", func(t *testing.T) {
		w, _ := groupDo(t, r, http.MethodGet, "/api/v1/groups", "", "")
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("状态码 = %d, want 401", w.Code)
		}
	})

	// 初始列表为空
	t.Run("列表为空", func(t *testing.T) {
		w, env := groupDo(t, r, http.MethodGet, "/api/v1/groups", key, "")
		if w.Code != http.StatusOK || env.Code != CodeOK {
			t.Fatalf("列表请求失败: code=%d body=%s", w.Code, w.Body.String())
		}
		var data struct {
			Total  int             `json:"total"`
			Groups json.RawMessage `json:"groups"`
		}
		_ = json.Unmarshal(env.Data, &data)
		if data.Total != 0 {
			t.Fatalf("空表 total 应为 0，实际 %d", data.Total)
		}
	})

	// 非法 body：400
	t.Run("创建非法body返回400", func(t *testing.T) {
		w, _ := groupDo(t, r, http.MethodPost, "/api/v1/groups", key, `{"name":""}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("状态码 = %d, want 400; body=%s", w.Code, w.Body.String())
		}
	})

	// JSON 语法错误同样 400
	t.Run("创建畸形JSON返回400", func(t *testing.T) {
		w, _ := groupDo(t, r, http.MethodPost, "/api/v1/groups", key, `{not-json`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("状态码 = %d, want 400", w.Code)
		}
	})

	var groupID int
	t.Run("创建分组成功", func(t *testing.T) {
		w, env := groupDo(t, r, http.MethodPost, "/api/v1/groups", key,
			`{"name":"常用","icon":"https://example.com/i.png","pwd":0}`)
		if w.Code != http.StatusOK || env.Code != CodeOK {
			t.Fatalf("创建失败: %s", w.Body.String())
		}
		var data struct {
			ID int `json:"id"`
		}
		_ = json.Unmarshal(env.Data, &data)
		if data.ID <= 0 {
			t.Fatalf("返回 id 非法: %d", data.ID)
		}
		groupID = data.ID
	})

	t.Run("列表含已创建分组", func(t *testing.T) {
		w, env := groupDo(t, r, http.MethodGet, "/api/v1/groups", key, "")
		if w.Code != http.StatusOK {
			t.Fatalf("列表失败: %s", w.Body.String())
		}
		var data struct {
			Total int `json:"total"`
		}
		_ = json.Unmarshal(env.Data, &data)
		if data.Total != 1 {
			t.Fatalf("total 应为 1，实际 %d", data.Total)
		}
	})

	t.Run("更新分组成功", func(t *testing.T) {
		w, env := groupDo(t, r, http.MethodPut, "/api/v1/groups/"+strconv.Itoa(groupID), key,
			`{"name":"常用-改","icon":"<svg></svg>","pwd":0}`)
		if w.Code != http.StatusOK || env.Code != CodeOK {
			t.Fatalf("更新失败: %s", w.Body.String())
		}
	})

	t.Run("更新不存在分组返回404", func(t *testing.T) {
		w, _ := groupDo(t, r, http.MethodPut, "/api/v1/groups/9999", key,
			`{"name":"x","icon":"","pwd":0}`)
		if w.Code != http.StatusNotFound {
			t.Fatalf("状态码 = %d, want 404", w.Code)
		}
	})

	t.Run("非法id返回400", func(t *testing.T) {
		w, _ := groupDo(t, r, http.MethodPut, "/api/v1/groups/abc", key,
			`{"name":"x","icon":"","pwd":0}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("状态码 = %d, want 400", w.Code)
		}
	})

	t.Run("停用分组成功", func(t *testing.T) {
		w, env := groupDo(t, r, http.MethodPost, "/api/v1/groups/"+strconv.Itoa(groupID)+"/status", key,
			`{"status":0}`)
		if w.Code != http.StatusOK || env.Code != CodeOK {
			t.Fatalf("状态更新失败: %s", w.Body.String())
		}
	})

	t.Run("status非法枚举返回400", func(t *testing.T) {
		w, _ := groupDo(t, r, http.MethodPost, "/api/v1/groups/"+strconv.Itoa(groupID)+"/status", key,
			`{"status":9}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("状态码 = %d, want 400", w.Code)
		}
	})

	t.Run("停用不存在分组返回404", func(t *testing.T) {
		w, _ := groupDo(t, r, http.MethodPost, "/api/v1/groups/9999/status", key, `{"status":0}`)
		if w.Code != http.StatusNotFound {
			t.Fatalf("状态码 = %d, want 404", w.Code)
		}
	})

	t.Run("排序成功", func(t *testing.T) {
		// 再建一个分组，保证 ids 非平凡
		_, env := groupDo(t, r, http.MethodPost, "/api/v1/groups", key, `{"name":"次用","icon":"","pwd":0}`)
		var second struct {
			ID int `json:"id"`
		}
		_ = json.Unmarshal(env.Data, &second)

		w, env := groupDo(t, r, http.MethodPost, "/api/v1/groups/sort", key,
			`{"ids":[`+strconv.Itoa(second.ID)+`,`+strconv.Itoa(groupID)+`]}`)
		if w.Code != http.StatusOK || env.Code != CodeOK {
			t.Fatalf("排序失败: %s", w.Body.String())
		}
	})

	t.Run("排序空ids返回400", func(t *testing.T) {
		w, _ := groupDo(t, r, http.MethodPost, "/api/v1/groups/sort", key, `{"ids":[]}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("状态码 = %d, want 400", w.Code)
		}
	})

	t.Run("删除不存在分组返回404", func(t *testing.T) {
		w, _ := groupDo(t, r, http.MethodDelete, "/api/v1/groups/9999", key, "")
		if w.Code != http.StatusNotFound {
			t.Fatalf("状态码 = %d, want 404", w.Code)
		}
	})

	t.Run("删除分组成功", func(t *testing.T) {
		w, env := groupDo(t, r, http.MethodDelete, "/api/v1/groups/"+strconv.Itoa(groupID), key, "")
		if w.Code != http.StatusOK || env.Code != CodeOK {
			t.Fatalf("删除失败: %s", w.Body.String())
		}
	})
}

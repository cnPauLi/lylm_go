package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLinkAPI(t *testing.T) {
	r, _, key := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) {
		RegisterGroupRoutes(v1, d)
		RegisterLinkRoutes(v1, d)
	})

	// 未携带 key：401
	t.Run("未鉴权返回401", func(t *testing.T) {
		w, _ := groupDo(t, r, http.MethodGet, "/api/v1/links", "", "")
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("状态码 = %d, want 401", w.Code)
		}
	})

	// 准备一个分组
	var groupID int
	t.Run("准备分组", func(t *testing.T) {
		_, env := groupDo(t, r, http.MethodPost, "/api/v1/groups", key, `{"name":"常用","icon":"","pwd":0}`)
		var data struct {
			ID int `json:"id"`
		}
		_ = json.Unmarshal(env.Data, &data)
		if data.ID <= 0 {
			t.Fatalf("分组 id 非法: %d", data.ID)
		}
		groupID = data.ID
	})

	var linkID int
	t.Run("创建链接成功", func(t *testing.T) {
		body := `{"name":"六零导航","url":"https://lylme.com","color":"","icon":"",` +
			`"group_id":0,"link_desc":"导航站","link_keywords":"导航,主页","pwd":0}`
		w, env := groupDo(t, r, http.MethodPost, "/api/v1/links", key, body)
		if w.Code != http.StatusOK || env.Code != CodeOK {
			t.Fatalf("创建链接失败: %s", w.Body.String())
		}
		var data struct {
			ID int `json:"id"`
		}
		_ = json.Unmarshal(env.Data, &data)
		if data.ID <= 0 {
			t.Fatalf("链接 id 非法: %d", data.ID)
		}
		linkID = data.ID
	})

	t.Run("创建缺少必填返回400", func(t *testing.T) {
		w, _ := groupDo(t, r, http.MethodPost, "/api/v1/links", key, `{"name":"","url":""}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("状态码 = %d, want 400; body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("重复URL创建返回409", func(t *testing.T) {
		// 末尾多一个斜杠，同样判重
		w, env := groupDo(t, r, http.MethodPost, "/api/v1/links", key,
			`{"name":"重复站","url":"https://lylme.com/","group_id":0}`)
		if w.Code != http.StatusConflict {
			t.Fatalf("状态码 = %d, want 409; body=%s", w.Code, w.Body.String())
		}
		if env.Code != http.StatusConflict {
			t.Fatalf("业务码 = %d, want 409", env.Code)
		}
		if env.Message != "链接已存在" {
			t.Fatalf("message = %q, want 链接已存在", env.Message)
		}
		var data struct {
			ID int `json:"id"`
		}
		_ = json.Unmarshal(env.Data, &data)
		if data.ID != linkID {
			t.Fatalf("data.id = %d, want %d", data.ID, linkID)
		}
	})

	var linkID2 int
	t.Run("创建第二条链接", func(t *testing.T) {
		_, env := groupDo(t, r, http.MethodPost, "/api/v1/links", key,
			`{"name":"第二个","url":"https://second.example","group_id":`+strconv.Itoa(groupID)+`}`)
		var data struct {
			ID int `json:"id"`
		}
		_ = json.Unmarshal(env.Data, &data)
		if data.ID <= 0 {
			t.Fatalf("第二条链接 id 非法")
		}
		linkID2 = data.ID
	})

	t.Run("分页列表", func(t *testing.T) {
		w, env := groupDo(t, r, http.MethodGet, "/api/v1/links?page=1&page_size=10", key, "")
		if w.Code != http.StatusOK {
			t.Fatalf("分页失败: %s", w.Body.String())
		}
		var data struct {
			List     []json.RawMessage `json:"list"`
			Page     int               `json:"page"`
			PageSize int               `json:"page_size"`
			Total    int64             `json:"total"`
		}
		_ = json.Unmarshal(env.Data, &data)
		if data.Page != 1 || data.PageSize != 10 || data.Total != 2 || len(data.List) != 2 {
			t.Fatalf("分页结果异常: %+v", data)
		}
	})

	t.Run("按分组过滤", func(t *testing.T) {
		w, env := groupDo(t, r, http.MethodGet,
			"/api/v1/links?group_id="+strconv.Itoa(groupID), key, "")
		if w.Code != http.StatusOK {
			t.Fatalf("分组过滤失败: %s", w.Body.String())
		}
		var data struct {
			Total int64 `json:"total"`
		}
		_ = json.Unmarshal(env.Data, &data)
		if data.Total != 1 {
			t.Fatalf("组内应有 1 条，实际 %d", data.Total)
		}
	})

	t.Run("keyword检索命中", func(t *testing.T) {
		w, env := groupDo(t, r, http.MethodGet, "/api/v1/links?keyword=导航", key, "")
		if w.Code != http.StatusOK {
			t.Fatalf("keyword 检索失败: %s", w.Body.String())
		}
		var data struct {
			Total int64 `json:"total"`
		}
		_ = json.Unmarshal(env.Data, &data)
		if data.Total != 1 {
			t.Fatalf("keyword=导航 应命中 1 条，实际 %d", data.Total)
		}
	})

	t.Run("keyword检索无命中", func(t *testing.T) {
		_, env := groupDo(t, r, http.MethodGet, "/api/v1/links?keyword=不存在的词xyz", key, "")
		var data struct {
			Total int64 `json:"total"`
			List  []any `json:"list"`
		}
		_ = json.Unmarshal(env.Data, &data)
		if data.Total != 0 || len(data.List) != 0 {
			t.Fatalf("无命中结果异常: %+v", data)
		}
	})

	t.Run("全量简表", func(t *testing.T) {
		w, env := groupDo(t, r, http.MethodGet, "/api/v1/links/all", key, "")
		if w.Code != http.StatusOK {
			t.Fatalf("all 失败: %s", w.Body.String())
		}
		var data struct {
			Count int `json:"count"`
			Links []struct {
				ID   int    `json:"id"`
				URL  string `json:"url"`
				Name string `json:"name"`
			} `json:"links"`
		}
		_ = json.Unmarshal(env.Data, &data)
		if data.Count != 2 || len(data.Links) != 2 {
			t.Fatalf("简表数量异常: %+v", data)
		}
	})

	t.Run("详情成功", func(t *testing.T) {
		w, env := groupDo(t, r, http.MethodGet, "/api/v1/links/"+strconv.Itoa(linkID), key, "")
		if w.Code != http.StatusOK {
			t.Fatalf("详情失败: %s", w.Body.String())
		}
		var row struct {
			ID int `json:"id"`
		}
		_ = json.Unmarshal(env.Data, &row)
		if row.ID != linkID {
			t.Fatalf("详情 id 异常: %d", row.ID)
		}
	})

	t.Run("详情不存在返回404", func(t *testing.T) {
		w, _ := groupDo(t, r, http.MethodGet, "/api/v1/links/9999", key, "")
		if w.Code != http.StatusNotFound {
			t.Fatalf("状态码 = %d, want 404", w.Code)
		}
	})

	t.Run("更新成功", func(t *testing.T) {
		body := `{"name":"六零导航-改","url":"https://lylme.net","color":"blue",` +
			`"icon":"i","group_id":` + strconv.Itoa(groupID) + `,"link_desc":"新描述",` +
			`"link_keywords":"新关键词","pwd":0}`
		w, env := groupDo(t, r, http.MethodPut, "/api/v1/links/"+strconv.Itoa(linkID), key, body)
		if w.Code != http.StatusOK || env.Code != CodeOK {
			t.Fatalf("更新失败: %s", w.Body.String())
		}
	})

	t.Run("更新不存在返回404", func(t *testing.T) {
		w, _ := groupDo(t, r, http.MethodPut, "/api/v1/links/9999", key,
			`{"name":"x","url":"https://x.com"}`)
		if w.Code != http.StatusNotFound {
			t.Fatalf("状态码 = %d, want 404", w.Code)
		}
	})

	t.Run("批量停用", func(t *testing.T) {
		w, env := groupDo(t, r, http.MethodPost, "/api/v1/links/status", key,
			`{"ids":[`+strconv.Itoa(linkID)+`,`+strconv.Itoa(linkID2)+`],"status":0}`)
		if w.Code != http.StatusOK || env.Code != CodeOK {
			t.Fatalf("批量停用失败: %s", w.Body.String())
		}
	})

	t.Run("status非法枚举返回400", func(t *testing.T) {
		w, _ := groupDo(t, r, http.MethodPost, "/api/v1/links/status", key,
			`{"ids":[`+strconv.Itoa(linkID)+`],"status":5}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("状态码 = %d, want 400", w.Code)
		}
	})

	t.Run("批量移动分组", func(t *testing.T) {
		w, env := groupDo(t, r, http.MethodPost, "/api/v1/links/group", key,
			`{"ids":[`+strconv.Itoa(linkID)+`],"group_id":`+strconv.Itoa(groupID)+`}`)
		if w.Code != http.StatusOK || env.Code != CodeOK {
			t.Fatalf("批量移动失败: %s", w.Body.String())
		}
	})

	t.Run("移动分组缺少group_id返回400", func(t *testing.T) {
		w, _ := groupDo(t, r, http.MethodPost, "/api/v1/links/group", key,
			`{"ids":[`+strconv.Itoa(linkID)+`],"group_id":0}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("状态码 = %d, want 400", w.Code)
		}
	})

	t.Run("批量设置密码", func(t *testing.T) {
		w, env := groupDo(t, r, http.MethodPost, "/api/v1/links/pwd", key,
			`{"ids":[`+strconv.Itoa(linkID)+`,`+strconv.Itoa(linkID2)+`],"pwd":0}`)
		if w.Code != http.StatusOK || env.Code != CodeOK {
			t.Fatalf("批量密码失败: %s", w.Body.String())
		}
	})

	t.Run("排序成功", func(t *testing.T) {
		w, env := groupDo(t, r, http.MethodPost, "/api/v1/links/sort", key,
			`{"ordered":["`+strconv.Itoa(linkID2)+`","`+strconv.Itoa(linkID)+`"]}`)
		if w.Code != http.StatusOK || env.Code != CodeOK {
			t.Fatalf("排序失败: %s", w.Body.String())
		}
	})

	t.Run("排序空ordered返回400", func(t *testing.T) {
		w, _ := groupDo(t, r, http.MethodPost, "/api/v1/links/sort", key, `{"ordered":[]}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("状态码 = %d, want 400", w.Code)
		}
	})

	t.Run("fetch缺少url返回400", func(t *testing.T) {
		w, _ := groupDo(t, r, http.MethodGet, "/api/v1/links/fetch", key, "")
		if w.Code != http.StatusBadRequest {
			t.Fatalf("状态码 = %d, want 400", w.Code)
		}
	})

	t.Run("fetch非法url返回400", func(t *testing.T) {
		for _, q := range []string{"not a url", "ftp://example.com", "http://"} {
			path := "/api/v1/links/fetch?url=" + url.QueryEscape(q)
			w, _ := groupDo(t, r, http.MethodGet, path, key, "")
			if w.Code != http.StatusBadRequest {
				t.Fatalf("url=%q 状态码 = %d, want 400", q, w.Code)
			}
		}
	})

	t.Run("批量删除", func(t *testing.T) {
		w, env := groupDo(t, r, http.MethodDelete, "/api/v1/links", key,
			`{"ids":[`+strconv.Itoa(linkID2)+`]}`)
		if w.Code != http.StatusOK || env.Code != CodeOK {
			t.Fatalf("批量删除失败: %s", w.Body.String())
		}
	})

	t.Run("删除空ids返回400", func(t *testing.T) {
		w, _ := groupDo(t, r, http.MethodDelete, "/api/v1/links", key, `{"ids":[]}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("状态码 = %d, want 400", w.Code)
		}
	})

	t.Run("删除后总数减少", func(t *testing.T) {
		_, env := groupDo(t, r, http.MethodGet, "/api/v1/links", key, "")
		var data struct {
			Total int64 `json:"total"`
		}
		_ = json.Unmarshal(env.Data, &data)
		if data.Total != 1 {
			t.Fatalf("删除后应剩 1 条，实际 %d", data.Total)
		}
	})
}

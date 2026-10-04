package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/models"
	"lylme_spage/internal/service"
)

// setupApplyTest 构建内存环境：注册 apply 域路由，造 1 个分组与 3 条待审申请
func setupApplyTest(t *testing.T) (*gin.Engine, *Deps, string, models.Group, []models.Apply) {
	t.Helper()
	r, d, key := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) { RegisterApplyRoutes(v1, d) })

	grp, err := d.Group.Create(service.GroupInput{Name: "技术博客"})
	if err != nil {
		t.Fatalf("创建分组失败: %v", err)
	}
	specs := []service.ApplyCreateInput{
		{Name: "申请一", URL: "https://a.example.com", GroupID: grp.GroupID},
		{Name: "申请二", URL: "https://b.example.com", GroupID: grp.GroupID},
		{Name: "申请三", URL: "https://c.example.com", GroupID: grp.GroupID},
	}
	rows := make([]models.Apply, 0, len(specs))
	for _, in := range specs {
		row, err := d.Apply.Create(in)
		if err != nil {
			t.Fatalf("创建申请失败: %v", err)
		}
		rows = append(rows, row)
	}
	return r, d, key, grp, rows
}

// applyRequest 发起一次 JSON 请求；key 为空表示不带鉴权头
func applyRequest(t *testing.T, r *gin.Engine, method, url, key string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("编码请求体失败: %v", err)
		}
		reader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, url, reader)
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

func mustStatus(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if w.Code != want {
		t.Fatalf("状态码 = %d, want %d；body=%s", w.Code, want, w.Body.String())
	}
}

func decodeApplyData(t *testing.T, w *httptest.ResponseRecorder, data any) {
	t.Helper()
	var resp struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应非统一信封 JSON: %v；body=%s", err, w.Body.String())
	}
	if resp.Code != CodeOK {
		t.Fatalf("业务码 = %d, want 200", resp.Code)
	}
	if err := json.Unmarshal(resp.Data, data); err != nil {
		t.Fatalf("解析 data 失败: %v；data=%s", err, resp.Data)
	}
}

func TestApplyList(t *testing.T) {
	r, _, key, grp, rows := setupApplyTest(t)

	t.Run("分页第一页", func(t *testing.T) {
		w := applyRequest(t, r, http.MethodGet, "/api/v1/apply?page=1&page_size=2", key, nil)
		mustStatus(t, w, http.StatusOK)
		var page PageResult[applyItem]
		decodeApplyData(t, w, &page)
		if page.Total != int64(len(rows)) || len(page.List) != 2 || page.Page != 1 || page.PageSize != 2 {
			t.Fatalf("分页结构异常: %+v", page)
		}
	})

	t.Run("分页第二页", func(t *testing.T) {
		w := applyRequest(t, r, http.MethodGet, "/api/v1/apply?page=2&page_size=2", key, nil)
		mustStatus(t, w, http.StatusOK)
		var page PageResult[applyItem]
		decodeApplyData(t, w, &page)
		if len(page.List) != 1 {
			t.Fatalf("第二页应有 1 条，得到 %d", len(page.List))
		}
	})

	t.Run("默认分页参数与分组名映射", func(t *testing.T) {
		w := applyRequest(t, r, http.MethodGet, "/api/v1/apply", key, nil)
		mustStatus(t, w, http.StatusOK)
		var page PageResult[applyItem]
		decodeApplyData(t, w, &page)
		if len(page.List) != 3 || page.Page != 1 || page.PageSize != 10 {
			t.Fatalf("默认分页异常: %+v", page)
		}
		for _, item := range page.List {
			if item.GroupID != grp.GroupID || item.GroupName != grp.GroupName {
				t.Fatalf("分组名映射异常: %+v, want %s", item, grp.GroupName)
			}
		}
	})

	t.Run("status=0 过滤", func(t *testing.T) {
		w := applyRequest(t, r, http.MethodGet, "/api/v1/apply?status=0", key, nil)
		mustStatus(t, w, http.StatusOK)
		var page PageResult[applyItem]
		decodeApplyData(t, w, &page)
		if page.Total != 3 || len(page.List) != 3 {
			t.Fatalf("待审过滤应有 3 条，得到 %+v", page)
		}
	})

	t.Run("非法 status 返回 400", func(t *testing.T) {
		w := applyRequest(t, r, http.MethodGet, "/api/v1/apply?status=9", key, nil)
		mustStatus(t, w, http.StatusBadRequest)
	})
}

func TestApplySettings(t *testing.T) {
	r, _, key, _, _ := setupApplyTest(t)

	t.Run("读取默认设置", func(t *testing.T) {
		w := applyRequest(t, r, http.MethodGet, "/api/v1/apply/settings", key, nil)
		mustStatus(t, w, http.StatusOK)
		var got applySettingsData
		decodeApplyData(t, w, &got)
		if got.Open != 0 || got.GG != "" {
			t.Fatalf("默认设置异常: %+v", got)
		}
	})

	t.Run("更新设置并回读", func(t *testing.T) {
		w := applyRequest(t, r, http.MethodPut, "/api/v1/apply/settings", key,
			map[string]any{"open": 1, "gg": "欢迎提交收录"})
		mustStatus(t, w, http.StatusOK)
		var got applySettingsData
		decodeApplyData(t, w, &got)
		if got.Open != 1 || got.GG != "欢迎提交收录" {
			t.Fatalf("更新响应设置异常: %+v", got)
		}
		w = applyRequest(t, r, http.MethodGet, "/api/v1/apply/settings", key, nil)
		decodeApplyData(t, w, &got)
		if got.Open != 1 || got.GG != "欢迎提交收录" {
			t.Fatalf("回读设置异常: %+v", got)
		}
	})

	t.Run("非法 open 返回 400", func(t *testing.T) {
		w := applyRequest(t, r, http.MethodPut, "/api/v1/apply/settings", key,
			map[string]any{"open": 3, "gg": ""})
		mustStatus(t, w, http.StatusBadRequest)
	})
}

func TestApplyUpdate(t *testing.T) {
	r, d, key, grp, rows := setupApplyTest(t)
	target := rows[0].ApplyID

	t.Run("修改成功", func(t *testing.T) {
		w := applyRequest(t, r, http.MethodPut, "/api/v1/apply/"+strconv.Itoa(target), key, map[string]any{
			"name":     "改名后的站点",
			"url":      "https://renamed.example.com",
			"group_id": grp.GroupID,
			"icon":     "https://renamed.example.com/favicon.ico",
		})
		mustStatus(t, w, http.StatusOK)
		row, err := d.Apply.Get(target)
		if err != nil {
			t.Fatalf("读取修改后记录失败: %v", err)
		}
		if row.ApplyName != "改名后的站点" || row.ApplyURL != "https://renamed.example.com" ||
			row.ApplyIcon != "https://renamed.example.com/favicon.ico" {
			t.Fatalf("修改未生效: %+v", row)
		}
	})

	t.Run("不存在记录返回 404", func(t *testing.T) {
		w := applyRequest(t, r, http.MethodPut, "/api/v1/apply/99999", key, map[string]any{
			"name": "不存在", "url": "https://x.example.com",
		})
		mustStatus(t, w, http.StatusNotFound)
	})

	t.Run("name 为空返回 400", func(t *testing.T) {
		w := applyRequest(t, r, http.MethodPut, "/api/v1/apply/"+strconv.Itoa(target), key, map[string]any{
			"name": "", "url": "https://x.example.com",
		})
		mustStatus(t, w, http.StatusBadRequest)
	})

	t.Run("非数字 id 返回 400", func(t *testing.T) {
		w := applyRequest(t, r, http.MethodPut, "/api/v1/apply/abc", key, map[string]any{
			"name": "非法 id",
		})
		mustStatus(t, w, http.StatusBadRequest)
	})
}

func TestApplyReview(t *testing.T) {
	r, d, key, _, rows := setupApplyTest(t)
	passID, rejectID, extraID := rows[0].ApplyID, rows[1].ApplyID, rows[2].ApplyID

	t.Run("审核通过恰好新增一条链接", func(t *testing.T) {
		before, err := d.Link.All()
		if err != nil {
			t.Fatalf("读取链接列表失败: %v", err)
		}
		w := applyRequest(t, r, http.MethodPost, "/api/v1/apply/"+strconv.Itoa(passID)+"/review", key,
			map[string]any{"status": 1})
		mustStatus(t, w, http.StatusOK)
		var got struct {
			ID     int `json:"id"`
			Status int `json:"status"`
		}
		decodeApplyData(t, w, &got)
		if got.ID != passID || got.Status != 1 {
			t.Fatalf("审核响应异常: %+v", got)
		}

		after, err := d.Link.All()
		if err != nil {
			t.Fatalf("再次读取链接列表失败: %v", err)
		}
		// 关键断言：通过新增链接数恰好为 1（len(after) == len(before)+1）
		if len(after) != len(before)+1 {
			t.Fatalf("审核通过应恰好新增 1 条链接，before=%d after=%d", len(before), len(after))
		}
		newLink := after[len(after)-1]
		if newLink.URL != "https://a.example.com" || newLink.Name != "申请一" {
			t.Fatalf("新增链接内容异常: %+v", newLink)
		}
		row, _ := d.Apply.Get(passID)
		if row.ApplyStatus != 1 {
			t.Fatalf("申请状态应为 1，得到 %d", row.ApplyStatus)
		}
	})

	t.Run("重复通过幂等不重复写链接", func(t *testing.T) {
		before, _ := d.Link.All()
		w := applyRequest(t, r, http.MethodPost, "/api/v1/apply/"+strconv.Itoa(passID)+"/review", key,
			map[string]any{"status": 1})
		mustStatus(t, w, http.StatusOK)
		after, _ := d.Link.All()
		if len(after) != len(before) {
			t.Fatalf("重复审核不应新增链接，before=%d after=%d", len(before), len(after))
		}
	})

	t.Run("审核拒绝不新增链接", func(t *testing.T) {
		before, _ := d.Link.All()
		w := applyRequest(t, r, http.MethodPost, "/api/v1/apply/"+strconv.Itoa(rejectID)+"/review", key,
			map[string]any{"status": 2})
		mustStatus(t, w, http.StatusOK)
		after, _ := d.Link.All()
		if len(after) != len(before) {
			t.Fatalf("审核拒绝不应新增链接，before=%d after=%d", len(before), len(after))
		}
		row, _ := d.Apply.Get(rejectID)
		if row.ApplyStatus != 2 {
			t.Fatalf("申请状态应为 2，得到 %d", row.ApplyStatus)
		}
	})

	t.Run("非法枚举返回 400 且不改变记录", func(t *testing.T) {
		w := applyRequest(t, r, http.MethodPost, "/api/v1/apply/"+strconv.Itoa(extraID)+"/review", key,
			map[string]any{"status": 3})
		mustStatus(t, w, http.StatusBadRequest)
		row, _ := d.Apply.Get(extraID)
		if row.ApplyStatus != 0 {
			t.Fatalf("非法审核不应改变状态，得到 %d", row.ApplyStatus)
		}
	})

	t.Run("不存在记录返回 404", func(t *testing.T) {
		w := applyRequest(t, r, http.MethodPost, "/api/v1/apply/99999/review", key,
			map[string]any{"status": 1})
		mustStatus(t, w, http.StatusNotFound)
	})
}

func TestApplyBatchDelete(t *testing.T) {
	r, d, key, _, rows := setupApplyTest(t)

	t.Run("批量删除成功", func(t *testing.T) {
		ids := []int{rows[0].ApplyID, rows[1].ApplyID}
		w := applyRequest(t, r, http.MethodDelete, "/api/v1/apply", key, map[string]any{"ids": ids})
		mustStatus(t, w, http.StatusOK)
		var got struct {
			Deleted int `json:"deleted"`
		}
		decodeApplyData(t, w, &got)
		if got.Deleted != len(ids) {
			t.Fatalf("deleted = %d, want %d", got.Deleted, len(ids))
		}
		res, err := d.Apply.List(-1)
		if err != nil {
			t.Fatalf("查询剩余申请失败: %v", err)
		}
		if len(res.Rows) != 1 {
			t.Fatalf("删除后应剩 1 条，得到 %d", len(res.Rows))
		}
	})

	t.Run("空 ids 返回 400", func(t *testing.T) {
		w := applyRequest(t, r, http.MethodDelete, "/api/v1/apply", key, map[string]any{"ids": []int{}})
		mustStatus(t, w, http.StatusBadRequest)
	})
}

func TestApplyNoKey(t *testing.T) {
	r, _, _, _, _ := setupApplyTest(t)
	w := applyRequest(t, r, http.MethodGet, "/api/v1/apply", "", nil)
	mustStatus(t, w, http.StatusUnauthorized)
}

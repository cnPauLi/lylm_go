package api

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/models"
	"lylme_spage/internal/service"
)

// RegisterApplyRoutes 在 /api/v1 下注册收录申请（apply）域端点：
// 列表分页/过滤、收录设置读写、修改、审核（通过/拒绝）、批量删除。
// 鉴权与限流由 /api/v1 路由组的统一中间件处理。
func RegisterApplyRoutes(g gin.IRouter, d *Deps) {
	g.GET("/apply", d.applyList)
	g.GET("/apply/settings", d.applyGetSettings)
	g.PUT("/apply/settings", d.applyUpdateSettings)
	g.PUT("/apply/:id", d.applyUpdate)
	g.POST("/apply/:id/review", d.applyReview)
	g.DELETE("/apply", d.applyBatchDelete)
}

// ===================== 请求结构 =====================

// applyListQuery GET /apply 查询参数；Status 为指针：缺省=全部，显式传值按状态过滤
type applyListQuery struct {
	Status   *int `form:"status" binding:"omitempty,oneof=0 1 2"`
	Page     int  `form:"page" binding:"omitempty,min=1"`
	PageSize int  `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// applySettingsRequest PUT /apply/settings 请求体；
// Open：0=开启（需审核）/ 1=开启（免审核）/ 2=关闭
type applySettingsRequest struct {
	Open int    `json:"open" binding:"oneof=0 1 2"`
	GG   string `json:"gg" binding:"max=255"`
}

// applyUpdateRequest PUT /apply/:id 请求体（对应 service.ApplyUpdateInput）
type applyUpdateRequest struct {
	Name    string `json:"name" binding:"required,max=60"`
	URL     string `json:"url" binding:"max=255"`
	GroupID int    `json:"group_id"`
	Icon    string `json:"icon"`
}

// applyReviewRequest POST /apply/:id/review 请求体；Status：1=通过 / 2=拒绝
type applyReviewRequest struct {
	Status int `json:"status" binding:"oneof=1 2"`
}

// applyBatchDeleteRequest DELETE /apply 请求体；ids 至少 1 个
type applyBatchDeleteRequest struct {
	IDs []int `json:"ids" binding:"required,min=1,dive,min=1"`
}

// ===================== 出参 DTO =====================

// applyItem 申请列表/详情元素：在 handler 组装 GroupID→Name 映射
type applyItem struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	GroupID   int       `json:"group_id"`
	GroupName string    `json:"group_name"`
	Icon      string    `json:"icon"`
	Desc      string    `json:"desc"`
	ApplyTime time.Time `json:"apply_time"`
	Status    int       `json:"status"`
}

// applySettingsData 收录设置出参
type applySettingsData struct {
	Open int    `json:"open"`
	GG   string `json:"gg"`
}

func toApplyItem(row models.Apply, groupMap map[int]string) applyItem {
	return applyItem{
		ID:        row.ApplyID,
		Name:      row.ApplyName,
		URL:       row.ApplyURL,
		GroupID:   row.ApplyGroup,
		GroupName: groupMap[row.ApplyGroup],
		Icon:      row.ApplyIcon,
		Desc:      row.ApplyDesc,
		ApplyTime: row.ApplyTime,
		Status:    row.ApplyStatus,
	}
}

func toApplySettings(s service.ApplySettings) applySettingsData {
	return applySettingsData{Open: s.Open, GG: s.GG}
}

// ===================== 处理器 =====================

// applyList GET /apply：按状态过滤（缺省全部）、按申请时间倒序，handler 层分页切片
func (d *Deps) applyList(c *gin.Context) {
	var q applyListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		FailBadRequest(c, err.Error())
		return
	}

	status := -1
	if q.Status != nil {
		status = *q.Status
	}
	res, err := d.Apply.List(status)
	if err != nil {
		WriteServiceError(c, err)
		return
	}

	// 分页切片属于展示逻辑，在 handler 完成
	page := q.Page
	if page < 1 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize < 1 {
		pageSize = 10
	}
	total := int64(len(res.Rows))
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(res.Rows) {
		start = len(res.Rows)
	}
	if end > len(res.Rows) {
		end = len(res.Rows)
	}
	items := make([]applyItem, 0, end-start)
	for _, row := range res.Rows[start:end] {
		items = append(items, toApplyItem(row, res.GroupMap))
	}
	OK(c, NewPageResult(items, page, pageSize, total))
}

// applyGetSettings GET /apply/settings
func (d *Deps) applyGetSettings(c *gin.Context) {
	OK(c, toApplySettings(d.Apply.Settings()))
}

// applyUpdateSettings PUT /apply/settings
func (d *Deps) applyUpdateSettings(c *gin.Context) {
	var req applySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		FailBadRequest(c, err.Error())
		return
	}
	if err := d.Apply.UpdateSettings(req.Open, req.GG); err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, toApplySettings(d.Apply.Settings()))
}

// applyUpdate PUT /apply/:id：不存在返回 404
func (d *Deps) applyUpdate(c *gin.Context) {
	id, ok := parseApplyIDParam(c)
	if !ok {
		return
	}
	var req applyUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		FailBadRequest(c, err.Error())
		return
	}
	if _, err := d.Apply.Get(id); err != nil {
		WriteServiceError(c, err)
		return
	}
	if err := d.Apply.Update(id, service.ApplyUpdateInput{
		Name:    req.Name,
		URL:     req.URL,
		GroupID: req.GroupID,
		Icon:    req.Icon,
	}); err != nil {
		WriteServiceError(c, err)
		return
	}
	row, err := d.Apply.Get(id)
	if err != nil {
		WriteServiceError(c, err)
		return
	}
	groupName := ""
	if g, err := d.Group.Get(row.ApplyGroup); err == nil {
		groupName = g.GroupName
	}
	OK(c, gin.H{
		"id":      id,
		"updated": true,
		"row":     toApplyItem(row, map[int]string{row.ApplyGroup: groupName}),
	})
}

// applyReview POST /apply/:id/review：1=通过（写链接表）/2=拒绝；
// 记录不存在 404；重复审核幂等（service 仅对 status=0 记录生效）
func (d *Deps) applyReview(c *gin.Context) {
	id, ok := parseApplyIDParam(c)
	if !ok {
		return
	}
	var req applyReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		FailBadRequest(c, err.Error())
		return
	}
	if _, err := d.Apply.Get(id); err != nil {
		WriteServiceError(c, err)
		return
	}
	if err := d.Apply.SetStatus([]int{id}, req.Status); err != nil {
		WriteServiceError(c, err)
		return
	}
	row, err := d.Apply.Get(id)
	if err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"id": id, "status": row.ApplyStatus})
}

// applyBatchDelete DELETE /apply：按 ids 批量删除
func (d *Deps) applyBatchDelete(c *gin.Context) {
	var req applyBatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		FailBadRequest(c, err.Error())
		return
	}
	if err := d.Apply.BatchDelete(req.IDs); err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"deleted": len(req.IDs)})
}

// parseApplyIDParam 解析并校验路径参数 id（正整数）
func parseApplyIDParam(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		FailBadRequest(c, "路径参数 id 必须为正整数")
		return 0, false
	}
	return id, true
}

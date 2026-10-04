package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/service"
)

// RegisterGroupRoutes 在 /api/v1 下注册分组（group）域端点。
// 鉴权与限流由挂载处的统一中间件处理。
func RegisterGroupRoutes(g gin.IRouter, d *Deps) {
	g.GET("/groups", d.listGroups)
	g.POST("/groups", d.createGroup)
	g.PUT("/groups/:id", d.updateGroup)
	g.DELETE("/groups/:id", d.deleteGroup)
	g.POST("/groups/:id/status", d.setGroupStatus)
	g.POST("/groups/sort", d.sortGroups)
}

// groupSaveRequest 分组新增/修改请求体
type groupSaveRequest struct {
	Name string `json:"name" binding:"required,max=30"` // 分组名称，必填
	Icon string `json:"icon" binding:"max=1000"`        // 图标（URL 或 SVG/HTML 片段）
	Pwd  int    `json:"pwd" binding:"min=0"`            // 关联的加密访问密码组 ID，0 表示不加密
}

// groupStatusRequest 分组启停用请求体
type groupStatusRequest struct {
	Status int `json:"status" binding:"oneof=0 1"` // 1 启用 / 0 停用
}

// groupSortRequest 分组排序请求体：ids 顺序即新的展示顺序
type groupSortRequest struct {
	IDs []int `json:"ids" binding:"required,min=1,max=200,dive,min=1"`
}

// groupPathID 解析路径参数 :id，非法时写入 400
func groupPathID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		FailBadRequest(c, "id 必须为正整数")
		return 0, false
	}
	return id, true
}

// listGroups GET /groups：返回全部分组、总数与各组链接数
func (d *Deps) listGroups(c *gin.Context) {
	list, err := d.Group.List()
	if err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{
		"total":      list.Total,
		"groups":     list.Groups,
		"link_count": list.LinkCount,
	})
}

// createGroup POST /groups：新增分组
func (d *Deps) createGroup(c *gin.Context) {
	var req groupSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		FailBadRequest(c, "请求参数不合法: "+err.Error())
		return
	}
	row, err := d.Group.Create(service.GroupInput{
		Name: req.Name,
		Icon: req.Icon,
		Pwd:  req.Pwd,
	})
	if err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"id": row.GroupID, "group": row})
}

// updateGroup PUT /groups/:id：修改分组
func (d *Deps) updateGroup(c *gin.Context) {
	id, ok := groupPathID(c)
	if !ok {
		return
	}
	var req groupSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		FailBadRequest(c, "请求参数不合法: "+err.Error())
		return
	}
	row, err := d.Group.Update(id, service.GroupInput{
		Name: req.Name,
		Icon: req.Icon,
		Pwd:  req.Pwd,
	})
	if err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"id": row.GroupID, "group": row})
}

// deleteGroup DELETE /groups/:id：删除分组（同时删除组内全部链接）
func (d *Deps) deleteGroup(c *gin.Context) {
	id, ok := groupPathID(c)
	if !ok {
		return
	}
	// service.Delete 按条件删除、不存在时不报错；此处先做存在性校验以返回 404
	if _, err := d.Group.Get(id); err != nil {
		WriteServiceError(c, err)
		return
	}
	if err := d.Group.Delete(id); err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"id": id})
}

// setGroupStatus POST /groups/:id/status：启用/停用分组
func (d *Deps) setGroupStatus(c *gin.Context) {
	id, ok := groupPathID(c)
	if !ok {
		return
	}
	var req groupStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		FailBadRequest(c, "请求参数不合法: "+err.Error())
		return
	}
	// 先校验存在性：对不存在的 id 返回 404
	if _, err := d.Group.Get(id); err != nil {
		WriteServiceError(c, err)
		return
	}
	if err := d.Group.SetStatus(id, req.Status); err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"id": id, "status": req.Status})
}

// sortGroups POST /groups/sort：按 ids 顺序重排分组
func (d *Deps) sortGroups(c *gin.Context) {
	var req groupSortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		FailBadRequest(c, "请求参数不合法: "+err.Error())
		return
	}
	if err := d.Group.Sort(req.IDs); err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"ids": req.IDs})
}

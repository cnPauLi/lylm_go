package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/service"
)

// tagRequest 导航菜单新增/修改请求体。
// 字段长度与 lylme_tags 表列定义对齐（tag_name 30 / tag_link 255）。
type tagRequest struct {
	// Name 导航名称
	Name string `json:"name" binding:"required,max=30"`
	// Link 导航链接
	Link string `json:"link" binding:"required,max=255"`
	// Sort 排序值（ASC；0 时由 service 取默认值 10）
	Sort int `json:"sort" binding:"min=0"`
	// Target 打开方式：1 新标签页，0 当前页
	Target int `json:"target" binding:"oneof=0 1"`
}

// RegisterTagRoutes 在 /api/v1 路由组下注册导航菜单（tag）域 4 个端点。
func RegisterTagRoutes(g gin.IRouter, d *Deps) {
	g.GET("/tags", d.tagList)
	g.POST("/tags", d.tagCreate)
	g.PUT("/tags/:id", d.tagUpdate)
	g.DELETE("/tags/:id", d.tagDelete)
}

// tagList GET /tags：返回全部导航菜单（sort ASC）
func (d *Deps) tagList(c *gin.Context) {
	rows, err := d.Tag.List()
	if err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"list": rows, "total": len(rows)})
}

// tagCreate POST /tags：新增导航菜单
func (d *Deps) tagCreate(c *gin.Context) {
	var req tagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		FailBadRequest(c, "请求参数校验失败: "+err.Error())
		return
	}
	row, err := d.Tag.Create(service.TagInput{
		Name:   req.Name,
		Link:   req.Link,
		Sort:   req.Sort,
		Target: req.Target,
	})
	if err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, row)
}

// tagUpdate PUT /tags/:id：修改导航菜单
func (d *Deps) tagUpdate(c *gin.Context) {
	id, ok := tspPathID(c)
	if !ok {
		return
	}
	var req tagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		FailBadRequest(c, "请求参数校验失败: "+err.Error())
		return
	}
	row, err := d.Tag.Update(id, service.TagInput{
		Name:   req.Name,
		Link:   req.Link,
		Sort:   req.Sort,
		Target: req.Target,
	})
	if err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, row)
}

// tagDelete DELETE /tags/:id：删除导航菜单；不存在返回 404。
func (d *Deps) tagDelete(c *gin.Context) {
	id, ok := tspPathID(c)
	if !ok {
		return
	}
	// service.Delete 按 WHERE 删除、0 行不报错，故先校验存在性以返回 404。
	if _, err := d.Tag.Get(id); err != nil {
		WriteServiceError(c, err)
		return
	}
	if err := d.Tag.Delete(id); err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"id": id})
}

// tspPathID 解析路径参数 :id（正整数），失败写 400；tag/sou/pwd 三域共用。
func tspPathID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		FailBadRequest(c, "路径参数 id 必须为正整数")
		return 0, false
	}
	return id, true
}

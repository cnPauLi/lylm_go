package api

import (
	"github.com/gin-gonic/gin"

	"lylme_spage/internal/service"
)

// souRequest 搜索引擎新增/修改请求体。
// 字段长度与 lylme_sou 表列定义对齐（alias/name 30、hint 60、color 20、
// link/waplink 255；icon 为 text）。REST 层要求 alias/name 必填，
// 比原版「六字段全空才报错」更严格，属于 API 契约的有意收紧。
type souRequest struct {
	// Alias 搜索引擎别名（前台 data-alias 使用）
	Alias string `json:"alias" binding:"required,max=30"`
	// Name 搜索引擎名称
	Name string `json:"name" binding:"required,max=30"`
	// Hint 输入框提示语
	Hint string `json:"hint" binding:"max=60"`
	// Color 标识颜色（如 #2932e1）
	Color string `json:"color" binding:"max=20"`
	// Link PC 端搜索链接模板（关键词占位由前端拼接）
	Link string `json:"link" binding:"max=255"`
	// Waplink 移动端搜索链接模板
	Waplink string `json:"waplink" binding:"max=255"`
	// Icon 图标（URL 或 SVG 名称/内容）
	Icon string `json:"icon" binding:"max=2000"`
	// St 是否启用：true→1 启用，false→0 停用
	St bool `json:"st"`
	// Order 排序值（仅修改时生效；新增由 service 置为 记录数+1）
	Order int `json:"order" binding:"min=0"`
}

// RegisterSouRoutes 在 /api/v1 路由组下注册搜索引擎（sou）域 4 个端点。
func RegisterSouRoutes(g gin.IRouter, d *Deps) {
	g.GET("/sou", d.souList)
	g.POST("/sou", d.souCreate)
	g.PUT("/sou/:id", d.souUpdate)
	g.DELETE("/sou/:id", d.souDelete)
}

// souList GET /sou：返回全部搜索引擎（sou_order ASC）
func (d *Deps) souList(c *gin.Context) {
	rows, err := d.Sou.List()
	if err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"list": rows, "total": len(rows)})
}

// souCreate POST /sou：新增搜索引擎
func (d *Deps) souCreate(c *gin.Context) {
	var req souRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		FailBadRequest(c, "请求参数校验失败: "+err.Error())
		return
	}
	row, err := d.Sou.Create(service.SouInput{
		Alias:   req.Alias,
		Name:    req.Name,
		Hint:    req.Hint,
		Color:   req.Color,
		Link:    req.Link,
		Waplink: req.Waplink,
		Icon:    req.Icon,
		St:      souSt(req.St),
		Order:   req.Order,
	})
	if err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, row)
}

// souUpdate PUT /sou/:id：修改搜索引擎
func (d *Deps) souUpdate(c *gin.Context) {
	id, ok := tspPathID(c)
	if !ok {
		return
	}
	var req souRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		FailBadRequest(c, "请求参数校验失败: "+err.Error())
		return
	}
	row, err := d.Sou.Update(id, service.SouInput{
		Alias:   req.Alias,
		Name:    req.Name,
		Hint:    req.Hint,
		Color:   req.Color,
		Link:    req.Link,
		Waplink: req.Waplink,
		Icon:    req.Icon,
		St:      souSt(req.St),
		Order:   req.Order,
	})
	if err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, row)
}

// souDelete DELETE /sou/:id：删除搜索引擎；不存在返回 404。
func (d *Deps) souDelete(c *gin.Context) {
	id, ok := tspPathID(c)
	if !ok {
		return
	}
	// service.Delete 按 WHERE 删除、0 行不报错，故先校验存在性以返回 404。
	if _, err := d.Sou.Get(id); err != nil {
		WriteServiceError(c, err)
		return
	}
	if err := d.Sou.Delete(id); err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"id": id})
}

// souSt 布尔 → 落库整数（true=1 / false=0）
func souSt(st bool) int {
	if st {
		return 1
	}
	return 0
}

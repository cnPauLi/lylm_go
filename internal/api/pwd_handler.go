package api

import (
	"github.com/gin-gonic/gin"

	"lylme_spage/internal/service"
)

// pwdRequest 加密组新增/修改请求体。
// 字段长度与 lylme_pwd 表列定义对齐（name 30、key/ps 60）。
type pwdRequest struct {
	// Name 加密组名称
	Name string `json:"name" binding:"required,max=30"`
	// Key 访问密码（链接的访问口令）
	Key string `json:"key" binding:"required,max=60"`
	// Ps 密码提示语
	Ps string `json:"ps" binding:"max=60"`
}

// RegisterPwdRoutes 在 /api/v1 路由组下注册加密组（pwd）域 4 个端点。
func RegisterPwdRoutes(g gin.IRouter, d *Deps) {
	g.GET("/pwd", d.pwdList)
	g.POST("/pwd", d.pwdCreate)
	g.PUT("/pwd/:id", d.pwdUpdate)
	g.DELETE("/pwd/:id", d.pwdDelete)
}

// pwdList GET /pwd：返回全部加密组
func (d *Deps) pwdList(c *gin.Context) {
	rows, err := d.Pwd.List()
	if err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"list": rows, "total": len(rows)})
}

// pwdCreate POST /pwd：新增加密组
func (d *Deps) pwdCreate(c *gin.Context) {
	var req pwdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		FailBadRequest(c, "请求参数校验失败: "+err.Error())
		return
	}
	row, err := d.Pwd.Create(service.PwdInput{
		Name: req.Name,
		Key:  req.Key,
		Ps:   req.Ps,
	})
	if err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, row)
}

// pwdUpdate PUT /pwd/:id：修改加密组
func (d *Deps) pwdUpdate(c *gin.Context) {
	id, ok := tspPathID(c)
	if !ok {
		return
	}
	var req pwdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		FailBadRequest(c, "请求参数校验失败: "+err.Error())
		return
	}
	row, err := d.Pwd.Update(id, service.PwdInput{
		Name: req.Name,
		Key:  req.Key,
		Ps:   req.Ps,
	})
	if err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, row)
}

// pwdDelete DELETE /pwd/:id：删除加密组；不存在返回 404。
func (d *Deps) pwdDelete(c *gin.Context) {
	id, ok := tspPathID(c)
	if !ok {
		return
	}
	// service.Delete 按 WHERE 删除、0 行不报错，故先校验存在性以返回 404。
	if _, err := d.Pwd.Get(id); err != nil {
		WriteServiceError(c, err)
		return
	}
	if err := d.Pwd.Delete(id); err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"id": id})
}

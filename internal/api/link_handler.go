package api

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/models"
	"lylme_spage/internal/service"
)

// RegisterLinkRoutes 在 /api/v1 下注册链接（link）域端点。
// 鉴权与限流由挂载处的统一中间件处理。
func RegisterLinkRoutes(g gin.IRouter, d *Deps) {
	g.GET("/links", d.listLinks)
	g.GET("/links/all", d.allLinks)
	g.GET("/links/:id", d.getLink)
	g.POST("/links", d.createLink)
	g.PUT("/links/:id", d.updateLink)
	g.DELETE("/links", d.deleteLinks)
	g.POST("/links/status", d.setLinkStatus)
	g.POST("/links/group", d.moveLinkGroup)
	g.POST("/links/pwd", d.setLinkPwd)
	g.POST("/links/sort", d.sortLinks)
	g.GET("/links/fetch", d.fetchLink)
}

const (
	linkDefaultPageSize = 10
	linkMaxPageSize     = 100
)

// linkSaveRequest 链接新增/修改请求体
type linkSaveRequest struct {
	Name     string `json:"name" binding:"required,max=255"`  // 链接名称，必填
	URL      string `json:"url" binding:"required,max=255"`   // 链接地址，必填
	Color    string `json:"color" binding:"max=32"`           // 名称颜色（十六进制/颜色名）
	Icon     string `json:"icon" binding:"max=2000"`          // 链接图标（URL/类名/SVG）
	GroupID  int    `json:"group_id" binding:"min=0"`          // 所属分组 ID，0 为未分组
	Desc     string `json:"link_desc" binding:"max=255"`      // 链接描述
	Keywords string `json:"link_keywords" binding:"max=512"`  // 检索关键词
	Pwd      int    `json:"pwd" binding:"min=0"`              // 访问密码组 ID，0 为无密码
}

// linkIDsRequest 批量操作通用请求体
type linkIDsRequest struct {
	IDs []int `json:"ids" binding:"required,min=1,max=500,dive,min=1"`
}

// linkStatusRequest 批量启停用请求体
type linkStatusRequest struct {
	IDs    []int `json:"ids" binding:"required,min=1,max=500,dive,min=1"`
	Status int   `json:"status" binding:"oneof=0 1"` // 1 启用 / 0 停用
}

// linkGroupRequest 批量移动分组请求体
type linkGroupRequest struct {
	IDs     []int `json:"ids" binding:"required,min=1,max=500,dive,min=1"`
	GroupID int   `json:"group_id" binding:"min=1"` // 目标分组必填
}

// linkPwdRequest 批量设置访问密码请求体
type linkPwdRequest struct {
	IDs []int `json:"ids" binding:"required,min=1,max=500,dive,min=1"`
	Pwd int   `json:"pwd" binding:"min=0"` // 密码组 ID，0 表示取消密码
}

// linkSortRequest 链接排序请求体：ordered 为按新顺序排列的链接 ID 字符串数组
type linkSortRequest struct {
	Ordered []string `json:"ordered" binding:"required,min=1,max=500,dive,required"`
}

// linkPathID 解析路径参数 :id，非法时写入 400
func linkPathID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		FailBadRequest(c, "id 必须为正整数")
		return 0, false
	}
	return id, true
}

// linkQueryInt 读取整型 query 参数：缺省/非法/负数时回退 def
func linkQueryInt(c *gin.Context, name string, def int) int {
	raw := c.Query(name)
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || v < 0 {
		return def
	}
	return v
}

// linkValidHTTPURL 校验 url 为 http/https 且包含主机名
func linkValidHTTPURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	return (scheme == "http" || scheme == "https") && u.Hostname() != ""
}

// toLinkInput 将请求体转为 service 入参（原始值，规范化由 service 完成）
func toLinkInput(req linkSaveRequest) service.LinkInput {
	return service.LinkInput{
		Name:     req.Name,
		URL:      req.URL,
		Color:    req.Color,
		Icon:     req.Icon,
		GroupID:  req.GroupID,
		Desc:     req.Desc,
		Keywords: req.Keywords,
		Pwd:      req.Pwd,
	}
}

// listLinks GET /links：分页查询链接，支持 group_id 过滤与 keyword 全文检索
func (d *Deps) listLinks(c *gin.Context) {
	page := linkQueryInt(c, "page", 1)
	pageSize := linkQueryInt(c, "page_size", linkDefaultPageSize)
	if pageSize > linkMaxPageSize {
		pageSize = linkMaxPageSize
	}
	groupID := linkQueryInt(c, "group_id", 0)
	keyword := strings.TrimSpace(c.Query("keyword"))

	var (
		rows  []models.Link
		total int64
		err   error
	)
	if keyword != "" {
		// keyword 非空时走全站关键词检索，group_id 不参与过滤
		rows, total, err = d.Link.SearchLinks(keyword, page, pageSize)
	} else {
		rows, total, err = d.Link.ListLinks(groupID, page, pageSize)
	}
	if err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, NewPageResult(rows, page, pageSize, total))
}

// allLinks GET /links/all：返回全站链接简表（id/url/name）
func (d *Deps) allLinks(c *gin.Context) {
	links, err := d.Link.All()
	if err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"count": len(links), "links": links})
}

// getLink GET /links/:id：链接详情
func (d *Deps) getLink(c *gin.Context) {
	id, ok := linkPathID(c)
	if !ok {
		return
	}
	row, err := d.Link.Get(id)
	if err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, row)
}

// createLink POST /links：新增链接；URL 重复返回 409
func (d *Deps) createLink(c *gin.Context) {
	var req linkSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		FailBadRequest(c, "请求参数不合法: "+err.Error())
		return
	}
	res, err := d.Link.Create(toLinkInput(req))
	if err != nil {
		WriteServiceError(c, err)
		return
	}
	if res.Duplicate {
		c.JSON(409, Response{
			Code:    409,
			Message: "链接已存在",
			Data:    gin.H{"id": res.Link.ID},
		})
		return
	}
	OK(c, gin.H{"id": res.Link.ID})
}

// updateLink PUT /links/:id：修改链接
func (d *Deps) updateLink(c *gin.Context) {
	id, ok := linkPathID(c)
	if !ok {
		return
	}
	var req linkSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		FailBadRequest(c, "请求参数不合法: "+err.Error())
		return
	}
	row, err := d.Link.Update(id, toLinkInput(req))
	if err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"id": row.ID, "link": row})
}

// deleteLinks DELETE /links：批量删除链接
func (d *Deps) deleteLinks(c *gin.Context) {
	var req linkIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		FailBadRequest(c, "请求参数不合法: "+err.Error())
		return
	}
	if err := d.Link.BatchDelete(req.IDs); err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"ids": req.IDs})
}

// setLinkStatus POST /links/status：批量启用/停用
func (d *Deps) setLinkStatus(c *gin.Context) {
	var req linkStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		FailBadRequest(c, "请求参数不合法: "+err.Error())
		return
	}
	if err := d.Link.BatchSetStatus(req.IDs, req.Status); err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"ids": req.IDs, "status": req.Status})
}

// moveLinkGroup POST /links/group：批量移动到指定分组
func (d *Deps) moveLinkGroup(c *gin.Context) {
	var req linkGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		FailBadRequest(c, "请求参数不合法: "+err.Error())
		return
	}
	if err := d.Link.BatchMoveGroup(req.IDs, req.GroupID); err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"ids": req.IDs, "group_id": req.GroupID})
}

// setLinkPwd POST /links/pwd：批量设置/取消访问密码
func (d *Deps) setLinkPwd(c *gin.Context) {
	var req linkPwdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		FailBadRequest(c, "请求参数不合法: "+err.Error())
		return
	}
	if err := d.Link.BatchSetPwd(req.IDs, req.Pwd); err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"ids": req.IDs, "pwd": req.Pwd})
}

// sortLinks POST /links/sort：按 ordered 顺序重排链接
func (d *Deps) sortLinks(c *gin.Context) {
	var req linkSortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		FailBadRequest(c, "请求参数不合法: "+err.Error())
		return
	}
	if err := d.Link.Sort(req.Ordered); err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"ordered": req.Ordered})
}

// fetchLink GET /links/fetch：抓取目标站点标题/图标等 head 信息
func (d *Deps) fetchLink(c *gin.Context) {
	rawURL := strings.TrimSpace(c.Query("url"))
	if rawURL == "" {
		FailBadRequest(c, "url 参数必填")
		return
	}
	if !linkValidHTTPURL(rawURL) {
		FailBadRequest(c, "url 格式非法，仅支持 http/https 且必须包含主机名")
		return
	}
	head, err := d.Link.FetchHead(rawURL)
	if err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, head)
}

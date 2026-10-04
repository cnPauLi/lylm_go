package admin

import (
	"encoding/json"
	"errors"
	gtmpl "html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/models"
	"lylme_spage/internal/service"
)

// jsonWrite 输出 JSON（不转义斜杠/Unicode，兼容原版）
func jsonWrite(c *gin.Context, payload gin.H) {
	c.Header("Content-Type", "application/json; charset=utf-8")
	b, err := json.Marshal(payload)
	if err != nil {
		_, _ = c.Writer.WriteString(`{"code":500,"msg":"服务器错误"}`)
		return
	}
	_, _ = c.Writer.Write(b)
}

// linkPage 链接管理模板数据
type linkPage struct {
	Groups  []models.Group
	Pwds    []models.Pwd
	Edit    *models.Link
	Add     bool
	Color   string
	Page    int
	MvGroup gtmpl.JS
	PwdList gtmpl.JS
}

// TableLink 链接列表片段数据
type TableLinkData struct {
	Page       int
	Total      int64
	GroupCount int64
	GPwd       int
	GroupLists []models.Group
	Links      []models.Link
	GroupNames map[int]string
}

// LinkPage 链接管理页 link.php
func (h *Handler) LinkPage(c *gin.Context) {
	groups, pwds := h.allGroupsPwds()
	page := queryInt(c, "page")
	if page <= 0 {
		page = 1
	}
	lp := &linkPage{Groups: groups, Pwds: pwds, Page: page}
	lp.MvGroup = gtmpl.JS(h.mvGroupJS(groups))
	lp.PwdList = gtmpl.JS(h.pwdListJS(pwds))

	switch c.Query("set") {
	case "add":
		lp.Add = true
	case "edit":
		row, err := h.LinkSvc.Get(queryInt(c, "id"))
		if err != nil {
			h.renderHTML(c, "该条记录不存在！")
			return
		}
		lp.Edit = &row
		lp.Color = colorFromName(row.Name)
	}
	data := h.newData(c, "链接管理", "link.php", "link")
	data.Page = lp
	h.render(c, "link.html", data)
}

// TableLink table_link.php?page=<group_id> 片段
func (h *Handler) TableLink(c *gin.Context) {
	gt := h.LinkSvc.GroupTable(queryInt(c, "page"))
	td := &TableLinkData{
		Page:       gt.Page,
		Total:      gt.Total,
		GroupCount: gt.GroupCount,
		GPwd:       gt.GPwd,
		GroupLists: gt.Groups,
		Links:      gt.Links,
		GroupNames: gt.GroupNames,
	}

	data := h.newData(c, "", "table_link.php", "link")
	data.Page = td
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.Engine.Render(c.Writer, "table_link.html", data); err != nil {
		h.Log.Errorf("渲染链接列表失败: %v", err)
		c.String(http.StatusInternalServerError, "渲染失败")
	}
}

// allGroupsPwds 读取全部分组与加密组
func (h *Handler) allGroupsPwds() ([]models.Group, []models.Pwd) {
	var groups []models.Group
	var pwds []models.Pwd
	if err := h.Store.DB().Order("group_order ASC").Find(&groups).Error; err != nil {
		h.Log.Errorf("查询分组失败: %v", err)
	}
	if err := h.Store.DB().Find(&pwds).Error; err != nil {
		h.Log.Errorf("查询加密组失败: %v", err)
	}
	return groups, pwds
}

// mvGroupJS 生成原版 link.php 中的 mv_group 下拉 HTML（供 jquery-confirm 使用）
func (h *Handler) mvGroupJS(groups []models.Group) string {
	var b strings.Builder
	b.WriteString(`var mv_group = '<form action="" class="formName">' + '<select class="form-control group_id" required><option value="">请选择分组...</option>'`)
	for _, g := range groups {
		b.WriteString(` + '<option value="` + strconv.Itoa(g.GroupID) + `">` + strconv.Itoa(g.GroupID) + ` - ` + jsQuote(g.GroupName) + `</option>'`)
	}
	b.WriteString(` + '</select>';`)
	return b.String()
}

// pwdListJS 生成原版 link.php 中的 pwd_list 下拉 HTML
func (h *Handler) pwdListJS(pwds []models.Pwd) string {
	var b strings.Builder
	b.WriteString(`var pwd_list = '<form action="" class="formName">' + '<select class="form-control pwd_id" required>'`)
	for _, p := range pwds {
		b.WriteString(` + '<option value="` + strconv.Itoa(p.PwdID) + `">` + strconv.Itoa(p.PwdID) + ` - ` + jsQuote(p.PwdName) + `</option>'`)
	}
	b.WriteString(` + '<option value="0">0 - 取消加密</option></select><br><a href="./pwd.php" target="_blank">管理加密组</a>';`)
	return b.String()
}

// jsQuote 转义 JS 单引号字符串中的特殊字符
func jsQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	s = strings.ReplaceAll(s, "\n", ``)
	return s
}

// colorFromName 从 <font color="xxx"> 中提取颜色
func colorFromName(name string) string {
	lower := strings.ToLower(name)
	idx := strings.Index(lower, `<font color="`)
	if idx < 0 {
		idx = strings.Index(lower, `<font color='`)
	}
	if idx < 0 {
		return ""
	}
	rest := name[idx+len(`<font color=`):]
	rest = strings.TrimLeft(rest, `"'`)
	end := strings.IndexAny(rest, `"'`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// AjaxLink ajax_link.php 统一入口
func (h *Handler) AjaxLink(c *gin.Context) {
	switch c.Query("submit") {
	case "add_tag":
		h.ajaxAddTag(c)
	case "edit_tag":
		h.ajaxEditTag(c)
	case "add_link":
		h.ajaxAddLink(c)
	case "edit_link":
		h.ajaxEditLink(c)
	case "add_sou":
		h.ajaxAddSou(c)
	case "edit_sou":
		h.ajaxEditSou(c)
	case "set_group":
		h.ajaxSetGroup(c)
	case "pwd_link":
		h.ajaxSetPwd(c)
	case "on":
		h.ajaxSetStatus(c, 1)
	case "off":
		h.ajaxSetStatus(c, 0)
	case "del":
		h.ajaxDelLinks(c)
	case "allorder":
		h.ajaxAllOrder(c)
	case "geturl":
		h.ajaxGetURL(c)
	case "all_links":
		h.ajaxAllLinks(c)
	case "edit_form":
		h.ajaxEditForm(c)
	case "check_update":
		h.ajaxCheckUpdate(c)
	case "update":
		jsonWrite(c, gin.H{"code": 200, "msg": "当前已是最新版本"})
	default:
		jsonWrite(c, gin.H{"code": -2, "msg": "error"})
	}
}

func (h *Handler) ajaxAddTag(c *gin.Context) {
	target := 0
	if c.PostForm("target") == "1" {
		target = 1
	}
	rec, err := h.TagSvc.Create(service.TagInput{
		Name:   c.PostForm("name"),
		Link:   c.PostForm("link"),
		Sort:   postInt(c, "sort"),
		Target: target,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			jsonWrite(c, gin.H{"code": 100, "msg": "保存错误,请确保带星号的都不为空！"})
			return
		}
		h.Log.Errorf("新增导航菜单失败: %v", err)
		jsonWrite(c, gin.H{"code": 500, "msg": "添加导航菜单失败"})
		return
	}
	jsonWrite(c, gin.H{"code": 200, "msg": "添加导航菜单 " + rec.TagName + " 成功！"})
}

func (h *Handler) ajaxEditTag(c *gin.Context) {
	target := 0
	if c.PostForm("target") == "1" {
		target = 1
	}
	row, err := h.TagSvc.Update(queryInt(c, "id"), service.TagInput{
		Name:   c.PostForm("name"),
		Link:   c.PostForm("link"),
		Sort:   postInt(c, "sort"),
		Target: target,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTagNotFound):
			jsonWrite(c, gin.H{"code": 404, "msg": "该条记录不存在！"})
		case errors.Is(err, service.ErrInvalidInput):
			jsonWrite(c, gin.H{"code": 100, "msg": "保存错误,请确保带星号的都不为空！"})
		default:
			h.Log.Errorf("修改导航菜单失败: %v", err)
			jsonWrite(c, gin.H{"code": 500, "msg": "修改导航菜单失败！"})
		}
		return
	}
	jsonWrite(c, gin.H{"code": 200, "msg": "修改导航菜单 " + row.TagName + " 成功！"})
}

// linkInputFromPost 从请求构建链接入参（原始值；规范化由 service 完成）
func linkInputFromPost(c *gin.Context) service.LinkInput {
	return service.LinkInput{
		Color:    c.PostForm("color"),
		Name:     c.PostForm("name"),
		URL:      c.PostForm("url"),
		Icon:     c.PostForm("icon"),
		GroupID:  postInt(c, "group_id"),
		Desc:     c.PostForm("link_desc"),
		Keywords: c.PostForm("link_keywords"),
		Pwd:      postInt(c, "link_pwd"),
	}
}

func (h *Handler) ajaxAddLink(c *gin.Context) {
	res, err := h.LinkSvc.Create(linkInputFromPost(c))
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			jsonWrite(c, gin.H{"code": 100, "msg": "保存错误,请确保带星号的都不为空！"})
			return
		}
		h.Log.Errorf("添加链接失败: %v", err)
		jsonWrite(c, gin.H{"code": 500, "msg": "添加链接失败！"})
		return
	}
	if res.Duplicate {
		sub := res.Submitted
		jsonWrite(c, gin.H{
			"code": 201, "msg": "链接已存在，跳过！ID=" + strconv.Itoa(res.Link.ID),
			"data": gin.H{
				"id": res.Link.ID, "name": sub.Name, "url": sub.URL, "color": sub.Color, "icon": sub.Icon,
				"group_id": sub.GroupID, "link_desc": sub.Desc, "link_keywords": sub.Keywords,
			},
		})
		return
	}
	jsonWrite(c, gin.H{"code": 200, "msg": "添加链接 " + res.Submitted.Name + " 成功！", "id": res.Link.ID})
}

func (h *Handler) ajaxEditLink(c *gin.Context) {
	row, err := h.LinkSvc.Update(queryInt(c, "id"), linkInputFromPost(c))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrLinkNotFound):
			jsonWrite(c, gin.H{"code": 404, "msg": "该条记录不存在！"})
		case errors.Is(err, service.ErrInvalidInput):
			jsonWrite(c, gin.H{"code": 100, "msg": "保存错误,请确保带星号的都不为空！"})
		default:
			h.Log.Errorf("修改链接失败: %v", err)
			jsonWrite(c, gin.H{"code": 500, "msg": "修改链接失败！"})
		}
		return
	}
	jsonWrite(c, gin.H{"code": 200, "msg": "修改链接 " + truncateRunes(row.Name, 20) + " 成功！"})
}

func (h *Handler) ajaxAddSou(c *gin.Context) {
	rec, err := h.SouSvc.Create(service.SouInput{
		Alias:   c.PostForm("alias"),
		Name:    c.PostForm("name"),
		Hint:    c.PostForm("hint"),
		Color:   c.PostForm("color"),
		Link:    c.PostForm("link"),
		Waplink: c.PostForm("waplink"),
		Icon:    c.PostForm("icon"),
		St:      boolVal(c.PostForm("st")),
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			jsonWrite(c, gin.H{"code": 100, "msg": "保存错误,请确保带星号的都不为空！"})
			return
		}
		h.Log.Errorf("新增搜索引擎失败: %v", err)
		jsonWrite(c, gin.H{"code": 500, "msg": "添加搜索引擎失败！"})
		return
	}
	jsonWrite(c, gin.H{"code": 200, "msg": "添加搜索引擎 " + rec.SouName + " 成功！"})
}

func (h *Handler) ajaxEditSou(c *gin.Context) {
	row, err := h.SouSvc.Update(queryInt(c, "id"), service.SouInput{
		Alias:   c.PostForm("alias"),
		Name:    c.PostForm("name"),
		Hint:    c.PostForm("hint"),
		Color:   c.PostForm("color"),
		Link:    c.PostForm("link"),
		Waplink: c.PostForm("waplink"),
		Icon:    c.PostForm("icon"),
		St:      boolVal(c.PostForm("st")),
		Order:   postInt(c, "order"),
	})
	if err != nil {
		if errors.Is(err, service.ErrSouNotFound) {
			jsonWrite(c, gin.H{"code": 404, "msg": "该条记录不存在！"})
			return
		}
		h.Log.Errorf("修改搜索引擎失败: %v", err)
		jsonWrite(c, gin.H{"code": 500, "msg": "修改失败！"})
		return
	}
	jsonWrite(c, gin.H{"code": 200, "msg": "修改搜索引擎 " + row.SouName + " 成功！"})
}

func (h *Handler) ajaxSetGroup(c *gin.Context) {
	ids := postInts(c, "links")
	if err := h.LinkSvc.BatchMoveGroup(ids, postInt(c, "group_id")); err != nil {
		h.reportBatchError(c, "批量移动分组", err, ids)
		return
	}
	jsonWrite(c, gin.H{"code": 200, "msg": "操作成功！"})
}

func (h *Handler) ajaxSetPwd(c *gin.Context) {
	ids := postInts(c, "links")
	if err := h.LinkSvc.BatchSetPwd(ids, postInt(c, "pwd_id")); err != nil {
		h.reportBatchError(c, "批量设置加密", err, ids)
		return
	}
	jsonWrite(c, gin.H{"code": 200, "msg": "操作成功！"})
}

func (h *Handler) ajaxSetStatus(c *gin.Context, status int) {
	ids := postInts(c, "links")
	msg := "启用成功！"
	if status == 0 {
		msg = "禁用成功！"
	}
	if err := h.LinkSvc.BatchSetStatus(ids, status); err != nil {
		h.reportBatchError(c, "批量更新状态", err, ids)
		return
	}
	jsonWrite(c, gin.H{"code": 200, "msg": msg})
}

func (h *Handler) ajaxDelLinks(c *gin.Context) {
	ids := postInts(c, "links")
	if err := h.LinkSvc.BatchDelete(ids); err != nil {
		h.reportBatchError(c, "批量删除链接", err, ids)
		return
	}
	jsonWrite(c, gin.H{"code": 200, "msg": "链接删除成功！"})
}

// reportBatchError 将 service 批量操作错误映射为原版输出：
// 空 ids（ErrInvalidInput）→「错误，失败0条」；数据库错误 →「错误，失败N条」
func (h *Handler) reportBatchError(c *gin.Context, action string, err error, ids []int) {
	if errors.Is(err, service.ErrInvalidInput) {
		jsonWrite(c, gin.H{"code": 100, "msg": "错误，失败0条"})
		return
	}
	h.Log.Errorf("%s失败: %v", action, err)
	jsonWrite(c, gin.H{"code": 100, "msg": "错误，失败" + strconv.Itoa(len(ids)) + "条"})
}

func (h *Handler) ajaxAllOrder(c *gin.Context) {
	raw := postStrings(c, "link_array")
	if err := h.LinkSvc.Sort(raw); err != nil {
		h.Log.Errorf("更新排序失败: %v", err)
	}
	jsonWrite(c, gin.H{"code": 200, "msg": "操作成功！", "order": strings.Join(raw, ",")})
}

func (h *Handler) ajaxGetURL(c *gin.Context) {
	head, err := h.LinkSvc.FetchHead(c.Query("url"))
	if err != nil {
		jsonWrite(c, gin.H{"code": 400, "msg": "Unable to access"})
		return
	}
	c.Header("Content-Type", "application/json; charset=utf-8")
	if b, err := json.Marshal(head); err == nil {
		_, _ = c.Writer.Write(b)
		return
	}
	jsonWrite(c, gin.H{"code": 400, "msg": "Unable to access"})
}

func (h *Handler) ajaxAllLinks(c *gin.Context) {
	links, err := h.LinkSvc.All()
	if err != nil {
		h.Log.Errorf("查询全站链接失败: %v", err)
	}
	out := make([]gin.H, 0, len(links))
	for _, l := range links {
		out = append(out, gin.H{"id": l.ID, "url": l.URL, "name": l.Name})
	}
	jsonWrite(c, gin.H{"code": 200, "msg": "ok", "count": len(out), "links": out})
}

// editFormPage iframe 编辑表单数据
type editFormPage struct {
	ID     int
	Link   models.Link
	Groups []models.Group
	Pwds   []models.Pwd
	Color  string
}

func (h *Handler) ajaxEditForm(c *gin.Context) {
	id := queryInt(c, "id")
	row, err := h.LinkSvc.Get(id)
	if err != nil {
		jsonWrite(c, gin.H{"code": 404, "msg": "该条记录不存在！"})
		return
	}
	groups, pwds := h.allGroupsPwds()
	data := h.newData(c, "编辑链接", "ajax_link.php", "link")
	data.Page = &editFormPage{ID: id, Link: row, Groups: groups, Pwds: pwds, Color: colorFromName(row.Name)}
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.Engine.Render(c.Writer, "edit_link.html", data); err != nil {
		h.Log.Errorf("渲染编辑表单失败: %v", err)
		c.String(http.StatusInternalServerError, "渲染失败")
	}
}

func (h *Handler) ajaxCheckUpdate(c *gin.Context) {
	cur := h.Store.Get("version", "")
	jsonWrite(c, gin.H{
		"code": 200, "version": cur, "current_version": cur,
		"update_log": "", "file": "",
	})
}

// boolVal 等价 PHP 中 `$x == true`：值为 "1"/"true"/"on" 视为真
func boolVal(s string) int {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "on", "yes":
		return 1
	}
	return 0
}

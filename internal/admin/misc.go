package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	gtmpl "html/template"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/models"
	"lylme_spage/internal/service"
	"lylme_spage/internal/theme"
)

// ===================== 搜索引擎 sou.php =====================

// souPage 搜索引擎模板数据
type souPage struct {
	Set   string
	Total int64
	Edit  *models.Sou
	Rows  []models.Sou
}

// SouPage 搜索引擎管理
func (h *Handler) SouPage(c *gin.Context) {
	set := c.Query("set")
	if set == "delete" {
		if err := h.SouSvc.Delete(queryInt(c, "id")); err != nil {
			h.Log.Errorf("删除搜索引擎失败: %v", err)
			h.renderHTML(c, alert("错误", "删除失败！", ""))
			return
		}
		h.renderHTML(c, alert("成功", "删除成功！", "./sou.php"))
		return
	}

	sp := &souPage{Set: set}
	if err := h.Store.DB().Model(&models.Sou{}).Count(&sp.Total).Error; err != nil {
		h.Log.Errorf("统计搜索引擎失败: %v", err)
	}
	rows, err := h.SouSvc.List()
	if err != nil {
		h.Log.Errorf("查询搜索引擎失败: %v", err)
	}
	sp.Rows = rows
	if set == "edit" {
		row, err := h.SouSvc.Get(queryInt(c, "id"))
		if err != nil {
			h.renderHTML(c, `<div class="alert alert-warning">该搜索引擎不存在！</div><a href="./sou.php">&lt;&lt;返回</a>`)
			return
		}
		sp.Edit = &row
	}
	data := h.newData(c, "搜索引擎设置", "sou.php", "sou")
	data.Page = sp
	h.render(c, "sou.html", data)
}

// ===================== 导航菜单 tag.php =====================

// tagPage 导航菜单模板数据
type tagPage struct {
	Set   string
	Total int
	Edit  *models.Tag
	Rows  []models.Tag
}

// TagPage 导航菜单管理
func (h *Handler) TagPage(c *gin.Context) {
	set := c.Query("set")
	if set == "delete" {
		if err := h.TagSvc.Delete(queryInt(c, "id")); err != nil {
			h.Log.Errorf("删除导航菜单失败: %v", err)
			h.renderHTML(c, alert("错误", "删除失败！", ""))
			return
		}
		h.renderHTML(c, alert("成功", "删除成功！", "./tag.php"))
		return
	}

	tp := &tagPage{Set: set}
	rows, err := h.TagSvc.List()
	if err != nil {
		h.Log.Errorf("查询导航菜单失败: %v", err)
	}
	tp.Rows = rows
	tp.Total = len(rows)
	if set == "edit" {
		row, err := h.TagSvc.Get(queryInt(c, "id"))
		if err != nil {
			h.renderHTML(c, alert("提示", "当前记录不存在！", "./tag.php"))
			return
		}
		tp.Edit = &row
	}
	data := h.newData(c, "导航菜单管理", "tag.php", "tag")
	data.Page = tp
	h.render(c, "tag.html", data)
}

// ===================== 加密组 pwd.php =====================

// pwdPage 加密组模板数据
type pwdPage struct {
	Set  string
	Edit *models.Pwd
	Rows []models.Pwd
}

// PwdPage 加密组管理
func (h *Handler) PwdPage(c *gin.Context) {
	set := c.Query("set")
	if c.Request.Method == http.MethodPost {
		switch set {
		case "add_submit":
			h.pwdAdd(c)
			return
		case "edit_submit":
			h.pwdEdit(c)
			return
		}
	}
	if set == "delete" {
		if err := h.PwdSvc.Delete(queryInt(c, "id")); err != nil {
			h.Log.Errorf("删除加密组失败: %v", err)
			h.renderHTML(c, alert("错误", "删除失败！", ""))
			return
		}
		h.renderHTML(c, `<script>window.location.href="./pwd.php";</script>`)
		return
	}

	pp := &pwdPage{Set: set}
	rows, err := h.PwdSvc.List()
	if err != nil {
		h.Log.Errorf("查询加密组失败: %v", err)
	}
	pp.Rows = rows
	if set == "edit" {
		row, err := h.PwdSvc.Get(queryInt(c, "id"))
		if err != nil {
			h.renderHTML(c, `<div class="alert alert-warning">该加密组不存在！</div><a href="./pwd.php">&lt;&lt;返回加密组列表</a>`)
			return
		}
		pp.Edit = &row
	}
	data := h.newData(c, "加密组管理", "pwd.php", "pwd")
	data.Page = pp
	h.render(c, "pwd.html", data)
}

func (h *Handler) pwdAdd(c *gin.Context) {
	_, err := h.PwdSvc.Create(service.PwdInput{
		Name: c.PostForm("pwd_name"),
		Key:  c.PostForm("pwd_key"),
		Ps:   c.PostForm("pwd_ps"),
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			h.renderHTML(c, alert("错误", "失败，请确保带星号的项目都不为空！", ""))
			return
		}
		h.Log.Errorf("新增加密组失败: %v", err)
		h.renderHTML(c, alert("错误", "添加失败", ""))
		return
	}
	h.renderHTML(c, `<script>window.location.href="./pwd.php";</script>`)
}

func (h *Handler) pwdEdit(c *gin.Context) {
	_, err := h.PwdSvc.Update(queryInt(c, "id"), service.PwdInput{
		Name: c.PostForm("pwd_name"),
		Key:  c.PostForm("pwd_key"),
		Ps:   c.PostForm("pwd_ps"),
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPwdNotFound):
			// 原版按 WHERE 更新、不校验存在性：不存在时同样输出跳转脚本
			h.renderHTML(c, `<script>window.location.href="./pwd.php";</script>`)
		case errors.Is(err, service.ErrInvalidInput):
			h.renderHTML(c, alert("错误", "失败，请确保带星号的项目都不为空！", ""))
		default:
			h.Log.Errorf("修改加密组失败: %v", err)
			h.renderHTML(c, alert("错误", "修改失败", ""))
		}
		return
	}
	h.renderHTML(c, `<script>window.location.href="./pwd.php";</script>`)
}

// ===================== 收录管理 apply.php =====================

// applyPage 收录管理模板数据
type applyPage struct {
	Set       string
	ApplyRows int64
	Groups    []models.Group
	GroupMap  map[int]string
	Rows      []models.Apply
	Edit      *models.Apply
	SiteHost  string
}

// ApplyPage 收录管理
func (h *Handler) ApplyPage(c *gin.Context) {
	set := c.Query("set")
	if c.Request.Method == http.MethodPost {
		switch set {
		case "conf_submit":
			if err := h.ApplySvc.UpdateSettings(postInt(c, "apply"), c.PostForm("apply_gg")); err != nil {
				h.Log.Errorf("保存收录设置失败: %v", err)
			}
			h.renderHTML(c, alert("成功", "修改成功！", "./apply.php"))
			return
		case "edit_submit":
			h.applyEdit(c)
			return
		}
	}

	ap := &applyPage{Set: set, SiteHost: siteURL(c, 2)}
	res, err := h.ApplySvc.List(-1)
	if err != nil {
		h.Log.Errorf("查询收录数据失败: %v", err)
	}
	ap.Groups = res.Groups
	ap.GroupMap = res.GroupMap
	ap.ApplyRows = int64(len(res.Rows))
	if set == "edit" {
		row, err := h.ApplySvc.Get(queryInt(c, "id"))
		if err != nil {
			h.renderHTML(c, `<div class="alert alert-warning">该申请记录不存在！</div><a href="./apply.php">&lt;&lt;返回收录管理列表</a>`)
			return
		}
		ap.Edit = &row
	} else if set != "conf" {
		ap.Rows = res.Rows
	}
	data := h.newData(c, "收录管理", "apply.php", "apply")
	data.Page = ap
	h.render(c, "apply.html", data)
}

func (h *Handler) applyEdit(c *gin.Context) {
	err := h.ApplySvc.Update(queryInt(c, "id"), service.ApplyUpdateInput{
		Name:    c.PostForm("apply_name"),
		URL:     c.PostForm("apply_url"),
		GroupID: postInt(c, "apply_group"),
		Icon:    c.PostForm("apply_icon"),
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			h.renderHTML(c, notify("保存错误,请确保带星号的都不为空！", "danger", 3000))
			return
		}
		h.Log.Errorf("修改收录失败: %v", err)
		h.renderHTML(c, notify("修改失败！", "danger", 3000))
		return
	}
	h.renderHTML(c, alert("成功", "修改成功！", "./apply.php"))
}

// AjaxApply ajax_apply.php
func (h *Handler) AjaxApply(c *gin.Context) {
	switch c.Query("set") {
	case "status":
		h.ajaxApplyStatus(c)
	case "delete":
		ids := postInts(c, "id")
		if len(ids) == 0 {
			jsonWrite(c, gin.H{"code": 200, "msg": "操作成功！"})
			return
		}
		if err := h.ApplySvc.BatchDelete(ids); err != nil {
			h.Log.Errorf("删除收录失败: %v", err)
			jsonWrite(c, gin.H{"code": 100, "msg": "错误，失败" + strconv.Itoa(len(ids)) + "条"})
			return
		}
		jsonWrite(c, gin.H{"code": 200, "msg": "操作成功！"})
	default:
		c.Header("Content-Type", "text/plain; charset=utf-8")
		_, _ = c.Writer.WriteString("error")
	}
}

func (h *Handler) ajaxApplyStatus(c *gin.Context) {
	sw := 2
	if c.PostForm("status") == "1" {
		sw = 1
	}
	if err := h.ApplySvc.SetStatus(postInts(c, "id"), sw); err != nil {
		h.Log.Errorf("更新收录状态失败: %v", err)
	}
	jsonWrite(c, gin.H{"code": 200, "msg": "操作成功！"})
}

// ===================== 主题 theme.php =====================

// themePage 主题列表模板数据
type themePage struct {
	Themes  []theme.Info
	Current string
}

// ThemePage 主题设置
func (h *Handler) ThemePage(c *gin.Context) {
	if set := c.Query("set"); set != "" {
		if err := h.Store.Set("template", set, "网站主题"); err != nil {
			h.Log.Errorf("切换主题失败: %v", err)
			h.renderHTML(c, alert("错误", "主题修改失败！", "./theme.php"))
			return
		}
		h.renderHTML(c, alert("成功", "主题修改成功！", "./theme.php"))
		return
	}
	data := h.newData(c, "主题设置", "theme.php", "theme")
	data.Page = &themePage{Themes: h.Store.Theme().List(), Current: h.Store.Template()}
	h.render(c, "theme.html", data)
}

// themeSettingPage 主题自定义设置模板数据
type themeSettingPage struct {
	ThemeName string
	HasConfig bool
	FormHTML  gtmpl.HTML
}

// ThemeSettingPage 主题自定义设置
func (h *Handler) ThemeSettingPage(c *gin.Context) {
	tpl := h.Store.Template()
	fields := h.Store.Theme().ConfigFields(tpl, h.Store.Conf())
	tp := &themeSettingPage{ThemeName: h.Store.Theme().Info(tpl).Name}
	if len(fields) == 0 {
		data := h.newData(c, "主题自定义设置", "theme_setting.php", "theme")
		data.Page = tp
		h.render(c, "theme_setting.html", data)
		return
	}
	tp.HasConfig = true
	values := theme.ConfigMap(h.Store.Conf(), tpl)
	tp.FormHTML = renderThemeForm(fields, values)
	data := h.newData(c, "主题自定义设置", "theme_setting.php", "theme")
	data.Page = tp
	h.render(c, "theme_setting.html", data)
}

// AjaxTheme ajax_theme.php
func (h *Handler) AjaxTheme(c *gin.Context) {
	if c.Query("set") != "save" || c.Request.Method != http.MethodPost {
		jsonWrite(c, gin.H{"code": -1, "msg": "非法请求！"})
		return
	}
	tpl := strings.TrimSpace(c.PostForm("theme_path"))
	if tpl == "" {
		tpl = h.Store.Template()
	}
	if !h.Store.Theme().Exists(tpl) {
		tpl = h.Store.Template()
	}
	_ = c.Request.ParseMultipartForm(32 << 20)
	data := map[string]any{}
	for key, vals := range c.Request.PostForm {
		if key == "file" || key == "theme_path" || key == "csrf_token" {
			continue
		}
		key = strings.TrimSuffix(key, "[]")
		if len(vals) > 1 {
			data[key] = vals
		} else {
			data[key] = vals[0]
		}
	}
	// 补回未提交的 checkbox / switch 默认值
	for _, f := range h.Store.Theme().ConfigFields(tpl, h.Store.Conf()) {
		switch f.Type {
		case "checkbox":
			if v, ok := data[f.Name]; !ok || isEmptyScalar(v) {
				data[f.Name] = []string{}
			}
		case "switch":
			if _, ok := data[f.Name]; !ok {
				data[f.Name] = 0
			}
		}
	}
	raw, err := json.Marshal(data)
	if err != nil {
		jsonWrite(c, gin.H{"code": 0, "msg": "保存失败"})
		return
	}
	name := h.Store.Theme().Info(tpl).Name
	if err := h.Store.Set("theme_config_"+tpl, string(raw), name+"主题自定义设置"); err != nil {
		h.Log.Errorf("保存主题设置失败: %v", err)
		jsonWrite(c, gin.H{"code": 0, "msg": "保存失败"})
		return
	}
	jsonWrite(c, gin.H{"code": 200, "msg": "保存成功"})
}

func isEmptyScalar(v any) bool {
	if s, ok := v.(string); ok {
		return s == ""
	}
	return false
}

// renderThemeForm 依据主题字段定义生成表单 HTML（等价原版 Form/BootstrapForm）
func renderThemeForm(fields []theme.Field, values map[string]any) gtmpl.HTML {
	var b strings.Builder
	b.WriteString(`<form class="form-horizontal" action="ajax_theme.php?set=save" method="post">`)
	for _, f := range fields {
		val := fieldValue(f, values)
		b.WriteString(renderThemeField(f, val))
	}
	b.WriteString(`</form>`)
	return gtmpl.HTML(b.String())
}

func fieldValue(f theme.Field, values map[string]any) any {
	if values != nil {
		if v, ok := values[f.Name]; ok {
			return v
		}
	}
	return f.Value
}

func renderThemeField(f theme.Field, val any) string {
	tip := ""
	if f.Description != "" {
		tip = `<div class="form-text">` + f.Description + `</div>`
	}
	switch f.Type {
	case "hidden":
		return `<input name="` + f.Name + `" value="` + escAttr(anyStr(val)) + `" type="hidden" />`
	case "submit":
		title := f.Title
		if title == "" {
			title = "保存"
		}
		return `<div class="mb-3 row"><div class="col-sm-12"><button type="submit" class="btn btn-primary btn-block">` + title + `</button></div></div>`
	case "textarea", "editor":
		name := f.Name
		return wrapField(f, `<textarea id="`+name+`" name="`+name+`" placeholder="`+escAttr(f.Placeholder)+`" class="form-control">`+escHTML(anyStr(val))+`</textarea>`+tip)
	case "color":
		return wrapField(f, `<input type="text" class="coloris form-control" placeholder="请选择颜色" name="`+f.Name+`" value="`+escAttr(anyStr(val))+`">`+tip)
	case "date":
		return wrapField(f, `<input name="`+f.Name+`" class="form-control" value="`+escAttr(anyStr(val))+`" type="text" input_type="date"/>`+tip)
	case "password":
		return wrapField(f, `<input name="`+f.Name+`" value="`+escAttr(anyStr(val))+`" type="password" class="form-control"/>`+tip)
	case "file":
		return wrapField(f, `<input name="`+f.Name+`" value="`+escAttr(anyStr(val))+`" type="text" class="file form-control" input_type="file"/>`+tip)
	case "select":
		var opts strings.Builder
		opts.WriteString(`<option value=""></option>`)
		for _, o := range f.Enum {
			sel := ""
			if o.Value == anyStr(val) {
				sel = "selected"
			}
			opts.WriteString(`<option value="` + escAttr(o.Value) + `" ` + sel + `>` + escHTML(o.Label) + `</option>`)
		}
		return wrapField(f, `<select class="form-select" name="`+f.Name+`">`+opts.String()+`</select>`+tip)
	case "select_multi":
		sel := map[string]bool{}
		for _, s := range anyList(val) {
			sel[s] = true
		}
		var opts strings.Builder
		opts.WriteString(`<option value=""></option>`)
		for _, o := range f.Enum {
			s := ""
			if sel[o.Value] {
				s = "selected"
			}
			opts.WriteString(`<option value="` + escAttr(o.Value) + `" ` + s + `>` + escHTML(o.Label) + `</option>`)
		}
		return wrapField(f, `<select class="form-select" name="`+f.Name+`" multiple>`+opts.String()+`</select>`+tip)
	case "radio":
		var opts strings.Builder
		for i, o := range f.Enum {
			ck := ""
			if o.Value == anyStr(val) {
				ck = "checked"
			}
			id := f.Name + "_" + strconv.Itoa(i)
			opts.WriteString(`<div class="form-check form-check-inline"><input class="form-check-input" type="radio" name="` + f.Name + `" value="` + escAttr(o.Value) + `" id="` + id + `" ` + ck + `/><label class="form-check-label" for="` + id + `">` + escHTML(o.Label) + `</label></div>`)
		}
		return wrapField(f, opts.String()+tip)
	case "checkbox":
		sel := map[string]bool{}
		for _, s := range anyList(val) {
			sel[s] = true
		}
		var opts strings.Builder
		opts.WriteString(`<input type="hidden" name="` + f.Name + `" value=""/>`)
		for i, o := range f.Enum {
			ck := ""
			if sel[o.Value] {
				ck = "checked"
			}
			id := f.Name + "_" + strconv.Itoa(i)
			v := ""
			if o.Value != "" {
				v = `value="` + escAttr(o.Value) + `"`
			}
			opts.WriteString(`<div class="form-check form-check-inline"><input class="form-check-input" type="checkbox" name="` + f.Name + `[]" ` + v + ` id="` + id + `" ` + ck + `/><label class="form-check-label" for="` + id + `">` + escHTML(o.Label) + `</label></div>`)
		}
		return wrapField(f, opts.String()+tip)
	case "switch":
		ck := ""
		if truthy(val) {
			ck = "checked"
		}
		return wrapField(f, `<div class="form-check form-switch"><input class="form-check-input" type="checkbox" name="`+f.Name+`" `+ck+`/></div>`+tip)
	case "empty_box":
		return `<div id="` + f.Name + `"></div>`
	case "none":
		return ""
	default:
		return wrapField(f, `<input name="`+f.Name+`" value="`+escAttr(anyStr(val))+`" type="text" placeholder="`+escAttr(f.Placeholder)+`" class="form-control"/>`+tip)
	}
}

func wrapField(f theme.Field, inner string) string {
	label := f.Title
	return `<div class="mb-3 row"><label class="col-sm-1 col-form-label">` + escHTML(label) + `</label><div class="col-sm-9">` + inner + `</div></div>`
}

func anyStr(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case int:
		return strconv.Itoa(t)
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return fmt.Sprintf("%v", t)
	case bool:
		if t {
			return "1"
		}
		return "0"
	default:
		return fmt.Sprintf("%v", t)
	}
}

func anyList(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, it := range t {
			out = append(out, anyStr(it))
		}
		return out
	case string:
		if t == "" {
			return nil
		}
		return strings.Split(t, ",")
	default:
		return nil
	}
}

func truthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t != "" && t != "0" && t != "false"
	case float64:
		return t != 0
	case int:
		return t != 0
	}
	return false
}

func escAttr(s string) string { return gtmpl.HTMLEscapeString(s) }
func escHTML(s string) string { return gtmpl.HTMLEscapeString(s) }

// ===================== 文件清理 cleanimg.php =====================

// cleanImgFile 单个文件项
type cleanImgFile struct {
	Rel, Name, Dir, URL, Ext, SizeStr string
	Size                              int64
	IsImg                             bool
}

// cleanImgDir 单目录统计
type cleanImgDir struct {
	Name   string
	Unused int
	Files  []cleanImgFile
}

// cleanImgPage 模板数据
type cleanImgPage struct {
	Total, Used, Unused int
	TotalSize           int64
	TotalSizeStr        string
	Dirs                []cleanImgDir
}

// fmtSize 人类可读文件大小
func fmtSize(b int64) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.2f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.2f KB", float64(b)/float64(1<<10))
	default:
		return strconv.FormatInt(b, 10) + " B"
	}
}

// CleanImg 未使用文件清理
func (h *Handler) CleanImg(c *gin.Context) {
	if c.Request.Method == http.MethodPost && c.Query("action") == "delete" {
		h.cleanImgDelete(c)
		return
	}
	page, err := h.scanUnused()
	if err != nil {
		h.Log.Errorf("扫描未使用文件失败: %v", err)
	}
	data := h.newData(c, "未使用文件清理", "cleanimg.php", "cleanimg")
	data.Page = page
	h.render(c, "cleanimg.html", data)
}

var cleanRelRe = regexp.MustCompile(`^(download|upload)/[^/]+$`)
var cleanDotRe = regexp.MustCompile(`(^|/)\.\.(/|$)`)

func (h *Handler) cleanImgDelete(c *gin.Context) {
	files := postStrings(c, "files")
	base := h.Cfg.Path("files")
	del, fail := 0, 0
	for _, rel := range files {
		rel = strings.ReplaceAll(strings.TrimSpace(rel), `\`, "/")
		if !cleanRelRe.MatchString(rel) || cleanDotRe.MatchString(rel) {
			fail++
			continue
		}
		full := filepath.Join(base, filepath.FromSlash(rel))
		if fi, err := os.Stat(full); err == nil && !fi.IsDir() {
			if err := os.Remove(full); err == nil {
				del++
				continue
			}
		}
		fail++
	}
	msg := "删除成功 " + strconv.Itoa(del) + " 个文件"
	if fail > 0 {
		msg += "，失败 " + strconv.Itoa(fail) + " 个"
	}
	h.renderHTML(c, alert("完成", msg, "cleanimg.php"))
}

func (h *Handler) scanUnused() (*cleanImgPage, error) {
	var icons []string
	if err := h.Store.DB().Model(&models.Group{}).Where("group_icon <> ''").Pluck("group_icon", &icons).Error; err != nil {
		return nil, err
	}
	var linkIcons []string
	if err := h.Store.DB().Model(&models.Link{}).Where("icon <> ''").Pluck("icon", &linkIcons).Error; err != nil {
		return nil, err
	}
	icons = append(icons, linkIcons...)

	page := &cleanImgPage{}
	for _, d := range []string{"download", "upload"} {
		dir := cleanImgDir{Name: d}
		entries, err := os.ReadDir(h.Cfg.Path("files", d))
		if err != nil {
			page.Dirs = append(page.Dirs, dir)
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			page.Total++
			name := e.Name()
			ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
			used := false
			for _, v := range icons {
				if strings.Contains(strings.ToLower(v), strings.ToLower(name)) {
					used = true
					break
				}
			}
			if used {
				page.Used++
				continue
			}
			page.Unused++
			page.TotalSize += info.Size()
			dir.Unused++
			dir.Files = append(dir.Files, cleanImgFile{
				Rel: d + "/" + name, Name: name, Dir: d, URL: "/files/" + d + "/" + name,
				Ext: ext, Size: info.Size(), SizeStr: fmtSize(info.Size()), IsImg: isImageExt(ext),
			})
		}
		page.Dirs = append(page.Dirs, dir)
	}
	page.TotalSizeStr = fmtSize(page.TotalSize)
	return page, nil
}

func isImageExt(ext string) bool {
	switch ext {
	case "png", "jpg", "jpeg", "gif", "ico", "webp":
		return true
	}
	return false
}

// ===================== 批量导入 batch_add.php =====================

// batchAddPage 批量导入模板数据
type batchAddPage struct {
	Mode         string
	Groups       []models.Group
	GroupID      int
	URL          string
	Title        string
	Icon         string
	BookmarkCode string
	GroupsJSON   gtmpl.JS
	SiteTitle    string
}

// BatchAdd 批量导入 / 书签快捷添加
func (h *Handler) BatchAdd(c *gin.Context) {
	url := strings.TrimSpace(c.Query("url"))
	title := strings.TrimSpace(c.Query("title"))
	icon := strings.TrimSpace(c.Query("icon"))
	groupID := queryInt(c, "group_id")
	mode := "batch"
	if c.Query("from") == "bm" || url != "" {
		mode = "quick"
	}

	var groups []models.Group
	if err := h.Store.DB().Order("group_order ASC, group_id ASC").Find(&groups).Error; err != nil {
		h.Log.Errorf("查询分组失败: %v", err)
	}
	if groupID == 0 && len(groups) > 0 {
		groupID = groups[0].GroupID
	}
	groupJSON, _ := json.Marshal(groups)

	page := &batchAddPage{
		Mode: mode, Groups: groups, GroupID: groupID,
		URL: url, Title: title, Icon: icon,
		SiteTitle:  h.Store.Get("title", "六零导航页"),
		GroupsJSON: gtmpl.JS(groupJSON),
	}
	if mode == "batch" {
		page.BookmarkCode = h.bookmarkCode(c)
	}
	data := h.newData(c, "批量导入链接", "batch_add.php", "link")
	data.Page = page
	h.render(c, "batch_add.html", data)
}

func (h *Handler) bookmarkCode(c *gin.Context) string {
	base := siteURL(c, 1) + h.Base() + "/batch_add.php"
	return "javascript:(function(){var u=location.href,t=document.title;void(open('" + base + "?from=bm&url='+encodeURIComponent(u)+'&title='+encodeURIComponent(t),'lylme_quick_add','toolbar=yes,location=yes,directories=no,status=no,menubar=yes,scrollbars=yes,resizable=no,copyhistory=yes,left=200,top=200,width=560,height=780'));})();"
}

// ===================== 检查更新 update.php =====================

// UpdatePage 检查更新页
func (h *Handler) UpdatePage(c *gin.Context) {
	data := h.newData(c, "检查更新", "update.php", "update")
	h.render(c, "update.html", data)
}

// ===================== 微信推送 wxplus.php =====================

// WxPlusPage 微信推送设置
func (h *Handler) WxPlusPage(c *gin.Context) {
	if c.Request.Method == http.MethodPost && c.Query("set") == "save" {
		userkey := c.PostForm("userkey")
		if err := h.Store.Set("wxplus", userkey, ""); err != nil {
			h.Log.Errorf("保存微信推送失败: %v", err)
		}
		if err := h.Store.Set("wxplustime", c.PostForm("wxplustime"), ""); err != nil {
			h.Log.Errorf("保存微信推送时间失败: %v", err)
		}
		if strings.TrimSpace(userkey) == "" {
			h.renderHTML(c, alert("提示", "微信推送已关闭", "./wxplus.php"))
		} else {
			h.renderHTML(c, alert("成功", "微信推送密钥配置成功", "./wxplus.php"))
		}
		return
	}
	data := h.newData(c, "微信推送", "wxplus.php", "wxplus")
	h.render(c, "wxplus.html", data)
}

// ===================== 网站授权 license.php =====================

// licensePage 授权设置模板数据
type licensePage struct {
	HostMode int
	Host1    string
	Host2    string
}

// LicensePage 网站授权设置
func (h *Handler) LicensePage(c *gin.Context) {
	if c.Request.Method == http.MethodPost && c.Query("set") == "save" {
		if err := h.Store.Set("c", c.PostForm("licensecode"), "程序授权码"); err != nil {
			h.Log.Errorf("保存授权码失败: %v", err)
		}
		if err := h.Store.Set("hostmode", c.PostForm("hostmode"), "域名获取方式"); err != nil {
			h.Log.Errorf("保存域名模式失败: %v", err)
		}
		h.renderHTML(c, alert("成功", "修改成功！", "./license.php"))
		return
	}
	hostMode := 2
	if h.Store.Get("hostmode", "2") == "1" {
		hostMode = 1
	}
	data := h.newData(c, "网站授权设置", "license.php", "license")
	data.Page = &licensePage{HostMode: hostMode, Host1: siteURL(c, 1), Host2: siteURL(c, 2)}
	h.render(c, "license.html", data)
}

// ===================== 帮助 help.php =====================

// HelpPage SVG 图标帮助
func (h *Handler) HelpPage(c *gin.Context) {
	if c.Query("doc") != "icon" {
		h.renderHTML(c, alert("错误", "无效的访问！", ""))
		return
	}
	data := h.newData(c, "SVG代码获取教程", "help.php", "")
	h.render(c, "help.html", data)
}

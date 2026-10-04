package admin

import (
	"encoding/base64"
	"encoding/json"
	gtmpl "html/template"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"lylme_spage/internal/stats"
)

// Engine 后台模板引擎（解析 web/admin/*.html）
type Engine struct {
	tmpl *gtmpl.Template
}

// NewEngine 解析 admin 目录下的全部 *.html 模板
func NewEngine(root string) (*Engine, error) {
	t, err := gtmpl.New("").Funcs(adminFuncMap).ParseGlob(filepath.Join(root, "*.html"))
	if err != nil {
		return nil, err
	}
	return &Engine{tmpl: t}, nil
}

// Render 按模板名渲染
func (e *Engine) Render(w io.Writer, name string, data any) error {
	return e.tmpl.ExecuteTemplate(w, name, data)
}

// Data 后台页面渲染上下文
type Data struct {
	Conf        map[string]string
	MenuKey     string
	ApplyRows   int
	CSRFToken   string
	PageTitle   string
	CurrentPage string
	AppVersion  string
	AdminPath   string
	SiteHost    string
	Stats       stats.Summary
	Page        any
}

// Cfg 读取网站配置项
func (d *Data) Cfg(key string) string {
	if d.Conf == nil {
		return ""
	}
	return d.Conf[key]
}

// CSRF 输出隐藏的 CSRF 表单域
func (d *Data) CSRF() gtmpl.HTML {
	return gtmpl.HTML(`<input type="hidden" name="csrf_token" value="` + gtmpl.HTMLEscapeString(d.CSRFToken) + `">`)
}

// Active 菜单高亮类
func (d *Data) Active(key string) string {
	if d.MenuKey == key {
		return " active"
	}
	return ""
}

// ActiveOpen 带子菜单展开的菜单高亮类
func (d *Data) ActiveOpen(key string) string {
	if d.MenuKey == key {
		return " active open"
	}
	return ""
}

// Is 判断当前页面是否为指定文件名
func (d *Data) Is(page string) bool { return d.CurrentPage == page }

// AdminUser 管理员用户名
func (d *Data) AdminUser() string { return d.Cfg("admin_user") }

// SiteTitle 站点标题
func (d *Data) SiteTitle() string {
	if t := d.Cfg("title"); t != "" {
		return t
	}
	return "六零导航页"
}

// TitleFirst 站点标题以 "-" 分割后的首段
func (d *Data) TitleFirst() string {
	parts := strings.Split(d.SiteTitle(), "-")
	return strings.TrimSpace(parts[0])
}

// IsDebug 是否开启调试模式
func (d *Data) IsDebug() bool { return d.Cfg("debug") == "1" }

var adminDigitRe = regexp.MustCompile(`\d+`)

// adminFuncMap 模板辅助函数
var adminFuncMap = gtmpl.FuncMap{
	"dict": func(kv ...string) map[string]string {
		m := map[string]string{}
		for i := 0; i+1 < len(kv); i += 2 {
			m[kv[i]] = kv[i+1]
		}
		return m
	},
	"raw":  func(s string) gtmpl.HTML { return gtmpl.HTML(s) },
	"b64":  func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) },
	"trim": strings.TrimSpace,
	"digits": func(s string) string {
		return adminDigitRe.FindString(s)
	},
	"js":        func(s string) gtmpl.JS { return gtmpl.JS(s) },
	"jsonJS": func(v any) gtmpl.JS {
		b, err := json.Marshal(v)
		if err != nil {
			return gtmpl.JS("null")
		}
		return gtmpl.JS(b)
	},
	"safeURL":   func(s string) gtmpl.URL { return gtmpl.URL(s) },
	"stripTags": stripTags,
	"hasPrefix": strings.HasPrefix,
	"timefmt":   func(t time.Time) string { return t.Format("2006-01-02 15:04:05") },
	"add":       func(a, b int) int { return a + b },
	"sub":       func(a, b int) int { return a - b },
	"year":      func() int { return time.Now().Year() },
}

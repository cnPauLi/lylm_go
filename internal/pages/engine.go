package pages

import (
	"encoding/base64"
	gtmpl "html/template"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"lylme_spage/internal/view"
)

// Engine 固定功能页（web/*.html）模板引擎
type Engine struct {
	base *view.Engine
	tmpl *gtmpl.Template
}

// NewEngine 解析 web 目录下的所有 *.html 固定页面模板
func NewEngine(root string, base *view.Engine) (*Engine, error) {
	tmpl, err := gtmpl.New("").Funcs(pageFuncMap).ParseGlob(filepath.Join(root, "*.html"))
	if err != nil {
		return nil, err
	}
	return &Engine{base: base, tmpl: tmpl}, nil
}

// DataFor 构建页面渲染上下文（复用主题引擎的数据组装逻辑）
func (e *Engine) DataFor(templateName string, isLogin, isMobile bool) *view.ViewData {
	return e.base.DataFor(templateName, isLogin, isMobile)
}

// Render 按文件名渲染固定页面模板（如 "about.html"）
func (e *Engine) Render(w io.Writer, name string, data *view.ViewData) error {
	return e.tmpl.ExecuteTemplate(w, name, data)
}

var pageDigitRe = regexp.MustCompile(`\d+`)

// pageFuncMap 与 internal/view 保持一致的模板函数集
var pageFuncMap = gtmpl.FuncMap{
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
		return pageDigitRe.FindString(s)
	},
}

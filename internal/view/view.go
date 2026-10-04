package view

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	stdhtml "html"
	gtmpl "html/template"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"lylme_spage/internal/models"
	"lylme_spage/internal/site"
)

// SouView 搜索引擎视图
type SouView struct {
	ID    int
	Alias string
	Name  string
	Hint  string
	Color string
	Icon  string
	Link  string
}

// ViewData 模板渲染上下文，方法即模板可调用的辅助函数
type ViewData struct {
	Conf          map[string]string
	Cdn           string
	Template      string
	TemplatePath  string
	AppVersion    string
	IsLogin       bool
	IsMobile      bool
	BackgroundImg string
	Groups        []site.GroupView
	Sou           []SouView
	Tags          []models.Tag
	CSRFToken     string
	Page          any

	store   *site.Store
	yanData []string
}

// Cfg 读取网站配置项
func (v *ViewData) Cfg(key string) string {
	if v.Conf == nil {
		return ""
	}
	return v.Conf[key]
}

// ThemeCfg 读取主题自定义配置
func (v *ViewData) ThemeCfg(name, def string) string {
	if v.store == nil {
		return def
	}
	return v.store.Theme().ConfigValue(v.Template, name, def, v.Conf)
}

// TitleShort 站点标题以 "-" 分割后的首段（等价 explode("-", $conf['title'])[0]）
func (v *ViewData) TitleShort() string {
	parts := strings.Split(v.Cfg("title"), "-")
	return strings.TrimSpace(parts[0])
}

// TitleFirst 站点标题首字符（等价 mb_substr(explode(...)[0], 0, 1)）
func (v *ViewData) TitleFirst() string {
	s := v.TitleShort()
	if s == "" {
		return ""
	}
	return string([]rune(s)[0])
}

// CSRF 输出隐藏的 CSRF 表单域（使用 CSRF() 方法渲染，避免 view 依赖 webutil）
func (v *ViewData) CSRF() gtmpl.HTML {
	return gtmpl.HTML(`<input type="hidden" name="csrf_token" value="` + gtmpl.HTMLEscapeString(v.CSRFToken) + `">`)
}

// LogoRoot 将 logo 统一为站点根路径，便于从 /about/ 等子目录访问
func (v *ViewData) LogoRoot() string {
	logo := strings.TrimSpace(v.Cfg("logo"))
	if logo == "" {
		return ""
	}
	logo = strings.TrimPrefix(logo, "./")
	if strings.HasPrefix(logo, "//") || strings.HasPrefix(logo, "http://") || strings.HasPrefix(logo, "https://") {
		return logo
	}
	if !strings.HasPrefix(logo, "/") {
		logo = "/" + logo
	}
	return logo
}

// Mode 运行模式（1 直接跳转，2 详情页）
func (v *ViewData) Mode() string {
	m := v.Cfg("mode")
	if m == "" {
		return "1"
	}
	return m
}

// Rel 链接 rel 属性（详情页模式不带 nofollow）
func (v *ViewData) Rel() string {
	if v.Mode() == "2" {
		return ""
	}
	return `rel="nofollow"`
}

// Yan 随机一言
func (v *ViewData) Yan() string {
	if len(v.yanData) == 0 {
		return "生活不止眼前的苟且，还有诗和远方的田野。"
	}
	return v.yanData[rand.Intn(len(v.yanData))]
}

// Lists 渲染链接列表（复刻原版 lists() 的片段占位符机制）
func (v *ViewData) Lists(frag map[string]string) gtmpl.HTML {
	var b strings.Builder
	for _, g := range v.Groups {
		gfrag := rearrangeMap(frag, groupReplacements(g))
		b.WriteString(gfrag["g1"])
		b.WriteString(gfrag["g2"])
		if len(g.Links) == 0 {
			b.WriteString(gfrag["g3"])
			b.WriteString("\n\n")
			continue
		}
		for _, l := range g.Links {
			lfrag := rearrangeMap(frag, linkReplacements(l, v.Mode()))
			b.WriteString("\n")
			b.WriteString(lfrag["l1"])
			b.WriteString(lfrag["l2"])
			b.WriteString(lfrag["l3"])
		}
		b.WriteString(gfrag["g3"])
		b.WriteString("\n\n")
	}
	return gtmpl.HTML(b.String())
}

// ListJSON 输出结构化链接数据（供主题 JS 使用）
func (v *ViewData) ListJSON() gtmpl.JS {
	type item struct {
		ID         int    `json:"id"`
		Title      string `json:"title"`
		Alias      string `json:"alias"`
		Keyword    string `json:"keyword"`
		CategoryID int    `json:"category_id"`
		Icon       string `json:"icon"`
		URL        string `json:"url"`
		Out        bool   `json:"out"`
	}
	type group struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
		Icon  string `json:"icon"`
		Items []item `json:"items"`
	}
	var arr []group
	for _, g := range v.Groups {
		gr := group{ID: g.GroupID, Title: g.GroupName, Icon: g.GroupIcon}
		for _, l := range g.Links {
			gr.Items = append(gr.Items, item{
				ID: l.ID, Title: l.Name, Alias: "link" + strconv.Itoa(l.ID),
				Keyword: l.Name, CategoryID: g.GroupID, Icon: l.Icon, URL: l.URL, Out: true,
			})
		}
		arr = append(arr, gr)
	}
	data, _ := json.Marshal(arr)
	return gtmpl.JS(data)
}

// ===== 主题辅助函数（等价各主题 functions.php 中的 theme_* 契约函数） =====

var nonAlnumRe = regexp.MustCompile(`[^a-zA-Z0-9]`)

// ThemeVersion 当前主题版本号（theme.ini 的 theme_version 去除非字母数字字符）
func (v *ViewData) ThemeVersion() string {
	if v.store == nil {
		return "100"
	}
	return sanitizeVersion(v.store.Theme().Info(v.Template).Version)
}

func sanitizeVersion(raw string) string {
	ver := nonAlnumRe.ReplaceAllString(raw, "")
	if ver == "" {
		return "100"
	}
	return ver
}

// CSS 输出主题样式表标签（等价 theme_css，自动附加版本参数）
func (v *ViewData) CSS(path string) gtmpl.HTML {
	href := strings.TrimRight(v.TemplatePath, "/") + "/" + strings.TrimLeft(path, "/") + "?v=" + v.ThemeVersion()
	return gtmpl.HTML(`<link rel="stylesheet" href="` + gtmpl.HTMLEscapeString(href) + `">` + "\n")
}

// JS 输出主题脚本标签（等价 theme_js，defer 加载并附加版本参数）
func (v *ViewData) JS(path string) gtmpl.HTML {
	src := strings.TrimRight(v.TemplatePath, "/") + "/" + strings.TrimLeft(path, "/") + "?v=" + v.ThemeVersion()
	return gtmpl.HTML(`<script defer src="` + gtmpl.HTMLEscapeString(src) + `"></script>` + "\n")
}

// Style 构造完整的内联 style 属性（CSS 文本原样输出，避免 html/template 对样式值再做转义）
func (v *ViewData) Style(css string) gtmpl.HTMLAttr {
	return gtmpl.HTMLAttr(`style="` + strings.ReplaceAll(css, `"`, `&#34;`) + `"`)
}

// ICP 输出 ICP 备案链接（等价 theme_icp，留空不输出）
func (v *ViewData) ICP() gtmpl.HTML {
	icp := strings.TrimSpace(v.Cfg("icp"))
	if icp == "" {
		return ""
	}
	return gtmpl.HTML(`<a class="icp" href="https://beian.miit.gov.cn/" target="_blank" rel="nofollow noopener">` +
		gtmpl.HTMLEscapeString(icp) + `</a>` + "\n")
}

// Gonganbei 输出公安备案链接（等价 theme_security_filing，留空不输出）
func (v *ViewData) Gonganbei() gtmpl.HTML {
	rec := strings.TrimSpace(v.ThemeCfg("gonganbei", ""))
	if rec == "" {
		return ""
	}
	return gtmpl.HTML(`<a class="gonganbei" href="http://www.beian.gov.cn/portal/registerSystemInfo?recordcode=` +
		gtmpl.HTMLEscapeString(digitRe.FindString(rec)) + `" target="_blank" rel="nofollow noopener">` +
		gtmpl.HTMLEscapeString(rec) + `</a>` + "\n")
}

// HasModule 判断主题配置项 modules（多选，逗号分隔）是否包含指定模块
func (v *ViewData) HasModule(name string) bool {
	for _, p := range strings.Split(v.ThemeCfg("modules", ""), ",") {
		if strings.TrimSpace(p) == name {
			return true
		}
	}
	return false
}

// EnumPick 主题枚举兜底：值不在白名单内时返回默认值（等价 theme_lg_enum）
func (v *ViewData) EnumPick(value, allowed, def string) string {
	for _, a := range strings.Split(allowed, ",") {
		if strings.TrimSpace(a) == value {
			return value
		}
	}
	return def
}

// RenderIcon 渲染分组/链接图标（等价 ltab 主题 ltabRenderIcon）
func (v *ViewData) RenderIcon(icon, alt string) gtmpl.HTML {
	alt = htmlspecialchars(stripTags(alt))
	if strings.TrimSpace(icon) == "" {
		return gtmpl.HTML(`<img src="/assets/img/default-icon.png" alt="` + alt + `" loading="lazy">`)
	}
	icon = stdhtml.UnescapeString(icon)
	trimmed := strings.TrimSpace(icon)
	if strings.HasPrefix(strings.ToLower(trimmed), "<svg") {
		return gtmpl.HTML(icon)
	}
	if !strings.Contains(trimmed, "<") {
		return gtmpl.HTML(`<img src="` + htmlspecialchars(trimmed) + `" alt="` + alt + `" loading="lazy">`)
	}
	return gtmpl.HTML(icon)
}

// SouJSON 输出搜索引擎数组 JSON（等价各主题内 array_push($json, ...) 构建结果）
func (v *ViewData) SouJSON() gtmpl.JS {
	arr := make([][3]string, 0, len(v.Sou))
	for _, s := range v.Sou {
		arr = append(arr, [3]string{s.Name, s.Hint, s.Link})
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(arr); err != nil {
		return gtmpl.JS("[]")
	}
	return gtmpl.JS(bytes.TrimRight(buf.Bytes(), "\n"))
}

// LGGlyph 输出 LiquidGlass 主题 SF Symbols 风格内联图标（等价 theme_lg_glyph）
func (v *ViewData) LGGlyph(name string) gtmpl.HTML {
	attr := ` viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"` +
		` stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"`
	body := ""
	switch name {
	case "apple":
		return gtmpl.HTML(`<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true" focusable="false">` +
			`<path d="M16.4 12.7c0-2.2 1.8-3.3 1.9-3.4-1-1.5-2.6-1.7-3.2-1.7-1.3-.1-2.6.8-3.3.8-.7 0-1.7-.8-2.8-.8-1.4 0-2.8.8-3.5 2.1-1.5 2.6-.4 6.5 1.1 8.6.7 1 1.6 2.2 2.7 2.2 1.1 0 1.5-.7 2.8-.7s1.6.7 2.7.7c1.1 0 1.9-1.1 2.6-2.1.8-1.2 1.1-2.3 1.2-2.4-.1 0-2.2-.9-2.2-3.3z` +
			`M14.3 5.8c.6-.7 1-1.7.9-2.7-.9 0-2 .6-2.6 1.3-.6.6-1 1.6-.9 2.6 1 .1 2-.5 2.6-1.2z"/></svg>`)
	case "search":
		body = `<circle cx="11" cy="11" r="6.8"/><path d="M16.2 16.2 20.5 20.5"/>`
	case "sliders":
		body = `<path d="M4 8h9M17.5 8H20M4 16h3.5M12 16h8"/><circle cx="15" cy="8" r="2.1"/><circle cx="9.8" cy="16" r="2.1"/>`
	case "gear":
		body = `<circle cx="12" cy="12" r="3.1"/><path d="M12 3.6v2.1M12 18.3v2.1M5 7.9l1.8 1M17.2 15.1l1.8 1M5 16.1l1.8-1M17.2 8.9l1.8-1"/>`
	case "squares":
		body = `<rect x="4" y="4" width="7" height="7" rx="2"/><rect x="13" y="4" width="7" height="7" rx="2"/><rect x="4" y="13" width="7" height="7" rx="2"/><rect x="13" y="13" width="7" height="7" rx="2"/>`
	case "compass":
		body = `<circle cx="12" cy="12" r="8.4"/><path d="M15 9l-1.7 4.4L9 15l1.7-4.4z"/>`
	case "bookmark":
		body = `<path d="M7 4.5h10a1 1 0 0 1 1 1V20l-6-3.6L6 20V5.5a1 1 0 0 1 1-1z"/>`
	case "wifi":
		body = `<path d="M2.6 9.2a13 13 0 0 1 18.8 0M6.1 12.7a8 8 0 0 1 11.8 0M9.6 16.2a3.6 3.6 0 0 1 4.8 0"/><circle cx="12" cy="19.2" r="1" fill="currentColor" stroke="none"/>`
	case "signal":
		body = `<path d="M3.5 17.5v3M8.5 13.5v7M13.5 9.5v11M18.5 5.5v15"/>`
	case "chevron":
		body = `<path d="M9.5 6l6 6-6 6"/>`
	case "clock":
		body = `<circle cx="12" cy="12" r="8.4"/><path d="M12 7.4V12l3.2 2"/>`
	case "sparkle":
		body = `<path d="M12 4.2l1.4 4.1 4.1 1.4-4.1 1.4L12 15.2l-1.4-4.1L6.5 9.7l4.1-1.4z"/><path d="M18.4 15.6l.7 2 2 .7-2 .7-.7 2-.7-2-2-.7 2-.7z"/>`
	case "eye":
		body = `<path d="M3.8 12S6.4 7.4 12 7.4 20.2 12 20.2 12 17.6 16.6 12 16.6 3.8 12 3.8 12z"/><circle cx="12" cy="12" r="2.6"/>`
	case "link":
		body = `<path d="M10.2 13.8a3.4 3.4 0 0 0 4.9 0l2.9-2.9a3.4 3.4 0 0 0-4.9-4.9l-1 1"/><path d="M13.8 10.2a3.4 3.4 0 0 0-4.9 0l-2.9 2.9a3.4 3.4 0 0 0 4.9 4.9l1-1"/>`
	case "minus":
		body = `<circle cx="12" cy="12" r="8.4"/><path d="M8.6 12h6.8"/>`
	case "back":
		body = `<path d="M14 6l-6 6 6 6"/>`
	case "forward":
		body = `<path d="M10 6l6 6-6 6"/>`
	default:
		body = `<circle cx="12" cy="12" r="8.4"/>`
	}
	return gtmpl.HTML(`<svg` + attr + `>` + body + `</svg>`)
}

// LGBattery 输出 LiquidGlass 主题电池组件（等价 theme_lg_battery）
func (v *ViewData) LGBattery(cls string) gtmpl.HTML {
	return gtmpl.HTML(`<span class="lg-batt ` + gtmpl.HTMLEscapeString(cls) + `" aria-hidden="true">` +
		`<span class="lg-batt-shell"><span class="lg-batt-level" data-lg-batt></span></span>` +
		`<span class="lg-batt-cap"></span></span>`)
}

// ===== 占位符替换（复刻原版 rearr()） =====

var tagRe = regexp.MustCompile(`<[^>]*>`)

func stripTags(s string) string { return tagRe.ReplaceAllString(s, "") }

func htmlspecialchars(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#039;")
	return r.Replace(s)
}

func groupReplacements(g site.GroupView) map[string]string {
	return map[string]string{
		"{group_id}":       strconv.Itoa(g.GroupID),
		"{group_name}":     g.GroupName,
		"{group_icon}":     g.GroupIcon,
		"{link_id}":        "",
		"{link_name}":      "",
		"{link_name_text}": "",
		"{link_desc}":      "",
		"{link_url}":       "",
		"{link_icon}":      "",
	}
}

func linkReplacements(l models.Link, mode string) map[string]string {
	url := l.URL
	if mode == "2" {
		url = "/site-" + strconv.Itoa(l.ID) + ".html"
	}
	alt := stripTags(l.Name)
	icon := l.Icon
	switch {
	case icon == "":
		icon = `<img src="/assets/img/default-icon.png" alt="` + htmlspecialchars(alt) + `" />`
	case !strings.HasPrefix(icon, "<sv"):
		icon = `<img src="` + htmlspecialchars(icon) + `" alt="` + htmlspecialchars(alt) + `" />`
	}
	return map[string]string{
		"{group_id}":       "",
		"{group_name}":     "",
		"{group_icon}":     "",
		"{link_id}":        strconv.Itoa(l.ID),
		"{link_name}":      l.Name,
		"{link_name_text}": htmlspecialchars(alt),
		"{link_desc}":      l.LinkDesc,
		"{link_url}":       url,
		"{link_icon}":      icon,
	}
}

func rearrangeMap(frag map[string]string, repl map[string]string) map[string]string {
	out := make(map[string]string, len(frag))
	for k, v := range frag {
		for pk, pv := range repl {
			v = strings.ReplaceAll(v, pk, pv)
		}
		out[k] = v
	}
	return out
}

// ===== 模板引擎 =====

// Engine 主题模板引擎
type Engine struct {
	store      *site.Store
	appVersion string
	yanData    []string

	mu    sync.RWMutex
	cache map[string]*gtmpl.Template
}

// NewEngine 创建模板引擎
func NewEngine(store *site.Store, appVersion string) *Engine {
	e := &Engine{
		store:      store,
		appVersion: appVersion,
		cache:      map[string]*gtmpl.Template{},
	}
	e.yanData = loadYan(store)
	return e
}

func loadYan(store *site.Store) []string {
	root := filepath.Dir(store.Theme().TemplateRoot())
	data, err := os.ReadFile(filepath.Join(root, "assets", "data", "data.dat"))
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r\n")
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}

var funcMap = gtmpl.FuncMap{
	"dict": func(kv ...string) map[string]string {
		m := map[string]string{}
		for i := 0; i+1 < len(kv); i += 2 {
			m[kv[i]] = kv[i+1]
		}
		return m
	},
	"raw":  func(s string) gtmpl.HTML { return gtmpl.HTML(s) },
	"css":  func(s string) gtmpl.CSS { return gtmpl.CSS(s) },
	"js":   func(s string) gtmpl.JS { return gtmpl.JS(s) },
	"b64":  func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) },
	"trim": strings.TrimSpace,
	"digits": func(s string) string {
		m := digitRe.FindString(s)
		return m
	},
}

var digitRe = regexp.MustCompile(`\d+`)

// DataFor 构建渲染上下文
func (e *Engine) DataFor(templateName string, isLogin bool, isMobile bool) *ViewData {
	conf := e.store.Conf()
	background := conf["background"]
	wap := conf["wap_background"]
	bgImg := background
	if isMobile && wap != "" {
		bgImg = wap
	}
	return &ViewData{
		Conf:          conf,
		Cdn:           e.cdn(conf),
		Template:      templateName,
		TemplatePath:  "./template/" + templateName,
		AppVersion:    e.appVersion,
		IsLogin:       isLogin,
		IsMobile:      isMobile,
		BackgroundImg: bgImg,
		CSRFToken:     "",
		store:         e.store,
		yanData:       e.yanData,
	}
}

func (e *Engine) cdn(conf map[string]string) string {
	c := conf["cdnpublic"]
	if c == "" {
		return "."
	}
	return strings.TrimRight(c, "/") + "/" + strings.TrimLeft(e.appVersion, "/")
}

// Render 渲染主题首页模板
func (e *Engine) Render(w io.Writer, templateName string, data *ViewData) error {
	t, err := e.themeTemplate(templateName)
	if err != nil {
		return err
	}
	return t.ExecuteTemplate(w, "index.html", data)
}

// RenderFile 渲染主题内指定模板文件
func (e *Engine) RenderFile(w io.Writer, templateName, file string, data *ViewData) error {
	t, err := e.themeTemplate(templateName)
	if err != nil {
		return err
	}
	return t.ExecuteTemplate(w, file, data)
}

func (e *Engine) themeTemplate(name string) (*gtmpl.Template, error) {
	e.mu.RLock()
	if t, ok := e.cache[name]; ok {
		e.mu.RUnlock()
		return t, nil
	}
	e.mu.RUnlock()

	dir := e.store.Theme().Dir(name)
	t, err := gtmpl.New("").Funcs(funcMap).ParseGlob(filepath.Join(dir, "*.html"))
	if err != nil {
		return nil, err
	}
	e.mu.Lock()
	e.cache[name] = t
	e.mu.Unlock()
	return t, nil
}

// Invalidate 清除主题模板缓存
func (e *Engine) Invalidate(name string) {
	e.mu.Lock()
	if name == "" {
		e.cache = map[string]*gtmpl.Template{}
	} else {
		delete(e.cache, name)
	}
	e.mu.Unlock()
}

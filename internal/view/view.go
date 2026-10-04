package view

import (
	"encoding/base64"
	"encoding/json"
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
	Alias string
	Name  string
	Hint  string
	Color string
	Icon  string
	Link  string
}

// ViewData 模板渲染上下文，方法即模板可调用的辅助函数
type ViewData struct {
	Conf    map[string]string
	Cdn     string
	Template string
	TemplatePath string
	AppVersion   string
	IsLogin      bool
	IsMobile     bool
	BackgroundImg string
	Groups       []site.GroupView
	Sou          []SouView
	Tags         []models.Tag
	CSRFToken    string
	Page         any

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

// ===== 占位符替换（复刻原版 rearr()） =====

var tagRe = regexp.MustCompile(`<[^>]*>`)

func stripTags(s string) string { return tagRe.ReplaceAllString(s, "") }

func htmlspecialchars(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#039;")
	return r.Replace(s)
}

func groupReplacements(g site.GroupView) map[string]string {
	return map[string]string{
		"{group_id}":   strconv.Itoa(g.GroupID),
		"{group_name}": g.GroupName,
		"{group_icon}": g.GroupIcon,
		"{link_id}":    "",
		"{link_name}":  "",
		"{link_name_text}": "",
		"{link_desc}":  "",
		"{link_url}":   "",
		"{link_icon}":  "",
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
		"{group_id}":   "",
		"{group_name}": "",
		"{group_icon}": "",
		"{link_id}":    strconv.Itoa(l.ID),
		"{link_name}":  l.Name,
		"{link_name_text}": htmlspecialchars(alt),
		"{link_desc}":  l.LinkDesc,
		"{link_url}":   url,
		"{link_icon}":  icon,
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
	"raw": func(s string) gtmpl.HTML { return gtmpl.HTML(s) },
	"b64": func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) },
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

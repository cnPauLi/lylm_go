package admin

import (
	"crypto/md5"
	"encoding/hex"
	"html"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"lylme_spage/internal/config"
	"lylme_spage/internal/logger"
	"lylme_spage/internal/models"
	"lylme_spage/internal/service"
	"lylme_spage/internal/site"
	"lylme_spage/internal/stats"
	"lylme_spage/internal/webutil"
)

// AppVersion 后台展示用的程序版本号（与 main.AppVersion 保持一致）
const AppVersion = "2.7.0"

// 会话闪存键名
const (
	sessLoginError = "login_error"
	sessLoginUser  = "login_error_user"
)

// failState 单 IP 登录失败/锁定状态
type failState struct {
	count     int
	lockUntil int64 // Unix 秒
	level     int
}

// Handler 后台处理器集合
type Handler struct {
	Cfg      *config.Config
	Store    *site.Store
	Log      *logger.Logger
	Engine   *Engine
	Stats    *stats.Store
	GroupSvc *service.GroupService

	mu    sync.Mutex
	fails map[string]*failState
}

// New 创建后台处理器
func New(cfg *config.Config, store *site.Store, log *logger.Logger) (*Handler, error) {
	engine, err := NewEngine(cfg.Path("web", "admin"))
	if err != nil {
		return nil, err
	}
	return &Handler{
		Cfg:      cfg,
		Store:    store,
		Log:      log,
		Engine:   engine,
		GroupSvc: service.NewGroupService(store),
		fails:    map[string]*failState{},
	}, nil
}

// AdminPath 返回实际生效的后台目录名（数据库配置优先，其次 .env）
func (h *Handler) AdminPath() string {
	p := strings.Trim(h.Store.Get("admin_path", h.Cfg.AdminPath), "/")
	if p == "" {
		p = "admin"
	}
	return p
}

// Base 后台根路径，如 /admin
func (h *Handler) Base() string { return "/" + h.AdminPath() }

// Sign 计算后台登录签名 md5(admin_user + admin_pwd)
func adminSign(store *site.Store) string {
	sum := md5.Sum([]byte(store.Get("admin_user", "admin") + store.Get("admin_pwd", "")))
	return hex.EncodeToString(sum[:])
}

// authed 判断当前请求是否通过后台鉴权
func authed(c *gin.Context, store *site.Store) bool {
	return webutil.IsAdmin(c) && webutil.AdminSign(c) == adminSign(store)
}

// RequireAuth 后台页面鉴权中间件：未登录跳转登录页
func RequireAuth(store *site.Store, basePath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !authed(c, store) {
			c.Redirect(http.StatusFound, basePath+"/login.php")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireAuthFragment 供 AJAX/表格片段使用的鉴权中间件：
// 与原版一致，未登录时输出跳转脚本并结束（不返回 302）
func RequireAuthFragment(store *site.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !authed(c, store) {
			c.Header("Content-Type", "text/html; charset=utf-8")
			_, _ = c.Writer.WriteString("<script>window.location.href='./login.php';</script>")
			c.Abort()
			return
		}
		c.Next()
	}
}

// Register 注册全部后台路由
func (h *Handler) Register(r *gin.Engine) {
	base := h.Base()
	auth := RequireAuth(h.Store, base)
	frag := RequireAuthFragment(h.Store)

	// 登录 / 登出
	r.GET(base+"/login.php", h.LoginPage)
	r.POST(base+"/login.php", h.LoginSubmit)

	// 仪表盘
	for _, p := range []string{base, base + "/", base + "/index.php"} {
		r.GET(p, auth, h.Dashboard)
	}

	// 网站配置
	r.GET(base+"/set.php", auth, h.SetPage)
	r.POST(base+"/set.php", auth, h.SetPage)
	r.GET(base+"/about.php", auth, h.AboutPage)
	r.POST(base+"/about.php", auth, h.AboutPage)
	r.GET(base+"/user.php", auth, h.UserPage)
	r.POST(base+"/user.php", auth, h.UserPage)

	// 链接管理
	r.GET(base+"/link.php", auth, h.LinkPage)
	r.GET(base+"/table_link.php", frag, h.TableLink)
	r.GET(base+"/ajax_link.php", frag, h.AjaxLink)
	r.POST(base+"/ajax_link.php", frag, h.AjaxLink)

	// 分组管理
	r.GET(base+"/group.php", auth, h.GroupPage)
	r.POST(base+"/group.php", auth, h.GroupPage)
	r.GET(base+"/table_group.php", frag, h.TableGroup)

	// 搜索引擎 / 导航菜单 / 加密组
	r.GET(base+"/sou.php", auth, h.SouPage)
	r.POST(base+"/sou.php", auth, h.SouPage)
	r.GET(base+"/tag.php", auth, h.TagPage)
	r.POST(base+"/tag.php", auth, h.TagPage)
	r.GET(base+"/pwd.php", auth, h.PwdPage)
	r.POST(base+"/pwd.php", auth, h.PwdPage)

	// 收录管理
	r.GET(base+"/apply.php", auth, h.ApplyPage)
	r.POST(base+"/apply.php", auth, h.ApplyPage)
	r.GET(base+"/ajax_apply.php", frag, h.AjaxApply)
	r.POST(base+"/ajax_apply.php", frag, h.AjaxApply)

	// 主题
	r.GET(base+"/theme.php", auth, h.ThemePage)
	r.GET(base+"/theme_setting.php", auth, h.ThemeSettingPage)
	r.GET(base+"/ajax_theme.php", frag, h.AjaxTheme)
	r.POST(base+"/ajax_theme.php", frag, h.AjaxTheme)

	// 文件清理 / 批量导入 / 更新 / 微信 / 授权 / 帮助
	r.GET(base+"/cleanimg.php", auth, h.CleanImg)
	r.POST(base+"/cleanimg.php", auth, h.CleanImg)
	r.GET(base+"/batch_add.php", auth, h.BatchAdd)
	r.GET(base+"/update.php", auth, h.UpdatePage)
	r.GET(base+"/wxplus.php", auth, h.WxPlusPage)
	r.POST(base+"/wxplus.php", auth, h.WxPlusPage)
	r.GET(base+"/license.php", auth, h.LicensePage)
	r.POST(base+"/license.php", auth, h.LicensePage)
	r.GET(base+"/help.php", auth, h.HelpPage)
}

// newData 构建后台模板上下文
func (h *Handler) newData(c *gin.Context, title, page, menu string) *Data {
	var applyRows int64
	if err := h.Store.DB().Model(&models.Apply{}).Where("apply_status = ?", 0).Count(&applyRows).Error; err != nil {
		h.Log.Errorf("统计待审申请失败: %v", err)
	}
	return &Data{
		Conf:        h.Store.Conf(),
		MenuKey:     menu,
		ApplyRows:   int(applyRows),
		CSRFToken:   webutil.CSRFToken(c),
		PageTitle:   title,
		CurrentPage: page,
		AppVersion:  AppVersion,
		AdminPath:   h.AdminPath(),
		SiteHost:    siteURL(c, 2),
	}
}

// render 渲染后台模板
func (h *Handler) render(c *gin.Context, name string, data *Data) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.Engine.Render(c.Writer, name, data); err != nil {
		h.Log.Errorf("后台页面渲染失败 %s: %v", name, err)
		c.String(http.StatusInternalServerError, "页面渲染失败: %v", err)
	}
}

// renderHTML 直接输出 HTML 字符串（用于兼容原版的脚本提示）
func (h *Handler) renderHTML(c *gin.Context, body string) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	_, _ = c.Writer.WriteString(body)
}

// flashSet 写入一次性提示
func flashSet(c *gin.Context, key, val string) {
	s := sessions.Default(c)
	s.Set(key, val)
	_ = s.Save()
}

// flashGet 读取并清除一次性提示
func flashGet(c *gin.Context, key string) string {
	s := sessions.Default(c)
	v, _ := s.Get(key).(string)
	if v != "" {
		s.Delete(key)
		_ = s.Save()
	}
	return v
}

// alert 生成与原版一致的 jquery-confirm 提示脚本
func alert(title, content, url string) string {
	action := ""
	if url != "" {
		action = `,action:function(){window.location.href="` + url + `";}`
	}
	return `<script>$.alert({title:"` + title + `",content:"` + content + `",buttons:{confirm:{text:"确定",btnClass:"btn-primary"` + action + `}}});</script>`
}

// notify 生成 lightyear.notify 提示脚本
func notify(msg, typ string, delay int) string {
	return `<script>lightyear.notify("` + msg + `", "` + typ + `", ` + strconv.Itoa(delay) + `);</script>`
}

// queryInt 读取 GET 整型参数
func queryInt(c *gin.Context, key string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(c.Query(key)))
	return n
}

// atoi 宽松整型解析
func atoi(s string) (int, error) { return strconv.Atoi(strings.TrimSpace(s)) }

// postInt 读取 POST 整型参数
func postInt(c *gin.Context, key string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(c.PostForm(key)))
	return n
}

// postInts 读取多值整型参数（兼容 `key` 与 PHP 风格 `key[]` 两种提交形式）
func postInts(c *gin.Context, key string) []int {
	raw := c.PostFormArray(key)
	if len(raw) == 0 {
		raw = c.PostFormArray(key + "[]")
	}
	out := make([]int, 0, len(raw))
	for _, s := range raw {
		if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
			out = append(out, n)
		}
	}
	return out
}

// postStrings 读取多值字符串参数（兼容 `key` 与 `key[]`）
func postStrings(c *gin.Context, key string) []string {
	raw := c.PostFormArray(key)
	if len(raw) == 0 {
		raw = c.PostFormArray(key + "[]")
	}
	return raw
}

// truncateRunes 按字符数截断并追加省略号（等价原版 truncate_text）
func truncateRunes(s string, max int) string {
	if max < 1 {
		max = 1
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-3]) + "..."
}

// siteURL 依据请求推导站点根地址；mode=1 使用 Host，mode=2 同 Host（Go 无 SERVER_NAME 区分）
func siteURL(c *gin.Context, mode int) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if p := c.GetHeader("X-Forwarded-Proto"); p != "" {
		if i := strings.Index(p, ","); i >= 0 {
			p = p[:i]
		}
		if v := strings.ToLower(strings.TrimSpace(p)); v != "" {
			scheme = v
		}
	}
	host := strings.TrimRight(c.Request.Host, "/")
	if host == "" {
		host = "localhost"
	}
	return scheme + "://" + host
}

var tagRe = regexp.MustCompile(`<[^>]*>`)

// stripTags 去除 HTML 标签
func stripTags(s string) string { return tagRe.ReplaceAllString(s, "") }

// htmlEscape 等价 htmlspecialchars
func htmlEscape(s string) string { return html.EscapeString(s) }

package router

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	"lylme_spage/internal/admin"
	"lylme_spage/internal/api"
	"lylme_spage/internal/captcha"
	"lylme_spage/internal/config"
	"lylme_spage/internal/install"
	"lylme_spage/internal/logger"
	"lylme_spage/internal/mcp"
	"lylme_spage/internal/pages"
	"lylme_spage/internal/site"
	"lylme_spage/internal/stats"
	"lylme_spage/internal/upload"
	"lylme_spage/internal/view"
	"lylme_spage/internal/webutil"
)

// Handler 路由处理器集合
type Handler struct {
	Cfg        *config.Config
	Store      *site.Store
	Engine     *view.Engine
	Pages      *pages.Engine
	Admin      *admin.Handler
	Log        *logger.Logger
	AppVersion string
}

// Setup 构建 Gin 路由
func Setup(h *Handler) *gin.Engine {
	gin.SetMode(h.Cfg.Mode)
	r := gin.New()
	r.Use(gin.Recovery(), h.requestLogger())

	// 未安装时守卫：非 /install 等白名单请求 302 跳转到 /install/
	r.Use(install.Guard(h.Cfg.RootDir))

	// 会话中间件（Cookie 存储，签名密钥来自配置）
	sessStore := cookie.NewStore([]byte(h.Cfg.SessionSecret))
	sessStore.Options(sessions.Options{
		Path:     "/",
		HttpOnly: true,
		MaxAge:   7 * 24 * 3600,
		SameSite: http.SameSiteLaxMode,
	})
	r.Use(sessions.Sessions("lylme_session", sessStore))

	// 访问统计（非后台页面请求计数）
	adminPath := h.Cfg.AdminPath
	if h.Admin != nil {
		adminPath = h.Admin.AdminPath()
	}
	statsStore := stats.New(h.Cfg.Path("logs"), h.Store.DB())
	statsStore.SetLogger(h.Log)
	r.Use(statsStore.Middleware(adminPath))
	if h.Admin != nil {
		h.Admin.Stats = statsStore
	}

	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	// 主题点击统计占位接口（dashlite / quality6.0 等主题的 JS 会向此上报点击）
	r.POST("/clitool", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"code": 200}) })

	// 静态资源：assets / template / site/static / files（屏蔽 .php 源码）
	// assets 下需放行 bing.php / cron.php 两个动态壁纸接口
	r.GET("/assets/*filepath", h.assetsHandler())
	r.GET("/template/*filepath", h.staticHandler(h.Cfg.Path("template")))
	r.GET("/site/static/*filepath", h.staticHandler(h.Cfg.Path("site", "static")))
	r.GET("/files/*filepath", h.staticHandler(h.Cfg.Path("files")))

	// 验证码 / 上传 / 访问密码 / 二维码
	r.GET("/include/validatecode.php", h.ValidateCode)
	r.POST("/include/file.php", upload.Handler(h.Cfg, h.Store, h.Log))
	r.GET("/include/go.php", pages.PasswordHandler(h.Store, h.Log))
	r.POST("/include/go.php", pages.PasswordHandler(h.Store, h.Log))
	r.GET("/include/qrcode.php", pages.Qrcode(h.Log))

	// 固定功能页
	r.GET("/about/", pages.About(h.Store, h.Pages, h.Log))
	r.GET("/about", pages.About(h.Store, h.Pages, h.Log))

	applyHandler := pages.Apply(h.Cfg, h.Store, h.Pages, h.Log)
	r.GET("/apply/", applyHandler)
	r.GET("/apply", applyHandler)
	r.GET("/apply/index.php", applyHandler)
	r.POST("/apply/index.php", applyHandler)
	r.POST("/apply/wxplus.php", pages.WxPlus(h.Cfg, h.Store, h.Log))
	r.StaticFile("/apply/apply.js", h.Cfg.Path("apply", "apply.js"))

	pwdHandler := pages.Pwd(h.Store, h.Pages, h.Log)
	r.GET("/pwd/", pwdHandler)
	r.GET("/pwd", pwdHandler)

	siteDetail := pages.SiteDetail(h.Cfg, h.Store, h.Pages, h.Log)
	r.GET("/site-:id", siteDetail)
	r.GET("/site", siteDetail)

	r.GET("/sitemap.xml", pages.Sitemap(h.Cfg, h.Store, h.Log))

	// 前台首页
	r.GET("/", h.Home)

	// 后台管理（路径由 ADMIN_PATH / admin_path 配置决定）
	if h.Admin != nil {
		h.Admin.Register(r)
	}

	// REST API（/api/v1）与 MCP 服务（/mcp）共用同一 Deps 实例与同一密钥体系
	d := api.NewDeps(h.Store, h.Cfg, h.AppVersion)
	api.RegisterV1(r, d)
	mcp.Mount(r, d)

	r.NoRoute(func(c *gin.Context) {
		c.String(http.StatusNotFound, "404 page not found")
	})
	return r
}

// ValidateCode 生成并输出验证码图片
func (h *Handler) ValidateCode(c *gin.Context) {
	code := captcha.NewCode()
	webutil.SetCaptchaCode(c, code)
	c.Header("Cache-Control", "no-store")
	c.Header("Content-Type", "image/png")
	if err := captcha.Generate(c.Writer, code); err != nil {
		h.Log.Errorf("验证码生成失败: %v", err)
		c.Status(http.StatusInternalServerError)
	}
}

// Home 前台首页
func (h *Handler) Home(c *gin.Context) {
	tplName := h.Store.Template()
	data := h.baseData(c, tplName)
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.Engine.Render(c.Writer, tplName, data); err != nil {
		h.Log.Errorf("首页渲染失败: %v", err)
		c.String(http.StatusInternalServerError, "主题模板渲染失败: %v", err)
	}
}

// baseData 构建公共渲染上下文
func (h *Handler) baseData(c *gin.Context, tplName string) *view.ViewData {
	isMobile := webutil.IsMobile(c.Request)
	data := h.Engine.DataFor(tplName, false, isMobile)

	groups, err := h.Store.Groups(h.unlocked(c))
	if err != nil {
		h.Log.Errorf("查询分组失败: %v", err)
	}
	data.Groups = groups

	tags, err := h.Store.Tags()
	if err != nil {
		h.Log.Errorf("查询导航菜单失败: %v", err)
	}
	data.Tags = tags

	souList, err := h.Store.SouEnabled()
	if err != nil {
		h.Log.Errorf("查询搜索引擎失败: %v", err)
	}
	for _, s := range souList {
		link := s.SouLink
		if isMobile && s.SouWaplink != "" {
			link = s.SouWaplink
		}
		data.Sou = append(data.Sou, view.SouView{
			ID:    s.SouID,
			Alias: s.SouAlias,
			Name:  s.SouName,
			Hint:  s.SouHint,
			Color: s.SouColor,
			Icon:  s.SouIcon,
			Link:  link,
		})
	}
	return data
}

// unlocked 读取会话中已解锁的加密组
func (h *Handler) unlocked(c *gin.Context) []int { return webutil.UnlockedIDs(c) }

// assetsHandler 服务 /assets/* ，并放行 bing.php / cron.php 动态接口
func (h *Handler) assetsHandler() gin.HandlerFunc {
	static := h.staticHandler(h.Cfg.Path("assets"))
	bing := pages.BingImage(h.Log)
	cron := pages.CronImage(h.Cfg, h.Store, h.Log)
	return func(c *gin.Context) {
		switch c.Param("filepath") {
		case "/img/bing.php":
			bing(c)
			return
		case "/img/cron.php":
			cron(c)
			return
		}
		static(c)
	}
}

// staticHandler 提供静态文件，屏蔽 .php 源码
func (h *Handler) staticHandler(root string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rel := strings.TrimPrefix(c.Param("filepath"), "/")
		if strings.HasSuffix(strings.ToLower(rel), ".php") {
			c.Status(http.StatusNotFound)
			return
		}
		clean := filepath.Clean("/" + rel)
		full := filepath.Join(root, clean)
		if !strings.HasPrefix(full, filepath.Clean(root)) {
			c.Status(http.StatusNotFound)
			return
		}
		fi, err := os.Stat(full)
		if err != nil || fi.IsDir() {
			c.Status(http.StatusNotFound)
			return
		}
		http.ServeFile(c.Writer, c.Request, full)
	}
}

func (h *Handler) requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		h.Log.Infof("%s %s %d %s", c.Request.Method, c.Request.URL.RequestURI(), c.Writer.Status(), time.Since(start))
	}
}

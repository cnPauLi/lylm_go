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

	"lylme_spage/internal/captcha"
	"lylme_spage/internal/config"
	"lylme_spage/internal/logger"
	"lylme_spage/internal/pages"
	"lylme_spage/internal/site"
	"lylme_spage/internal/upload"
	"lylme_spage/internal/view"
	"lylme_spage/internal/webutil"
)

// Handler 路由处理器集合
type Handler struct {
	Cfg    *config.Config
	Store  *site.Store
	Engine *view.Engine
	Log    *logger.Logger
}

// Setup 构建 Gin 路由
func Setup(h *Handler) *gin.Engine {
	gin.SetMode(h.Cfg.Mode)
	r := gin.New()
	r.Use(gin.Recovery(), h.requestLogger())

	// 会话中间件（Cookie 存储，签名密钥来自配置）
	sessStore := cookie.NewStore([]byte(h.Cfg.SessionSecret))
	sessStore.Options(sessions.Options{
		Path:     "/",
		HttpOnly: true,
		MaxAge:   7 * 24 * 3600,
		SameSite: http.SameSiteLaxMode,
	})
	r.Use(sessions.Sessions("lylme_session", sessStore))

	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	// 静态资源：assets / template / site/static / files（屏蔽 .php 源码）
	r.GET("/assets/*filepath", h.staticHandler(h.Cfg.Path("assets")))
	r.GET("/template/*filepath", h.staticHandler(h.Cfg.Path("template")))
	r.GET("/site/static/*filepath", h.staticHandler(h.Cfg.Path("site", "static")))
	r.GET("/files/*filepath", h.staticHandler(h.Cfg.Path("files")))

	// 验证码 / 上传 / 访问密码
	r.GET("/include/validatecode.php", h.ValidateCode)
	r.POST("/include/file.php", upload.Handler(h.Cfg, h.Store, h.Log))
	r.GET("/include/go.php", pages.PasswordHandler(h.Store, h.Log))
	r.POST("/include/go.php", pages.PasswordHandler(h.Store, h.Log))

	// 前台首页
	r.GET("/", h.Home)

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

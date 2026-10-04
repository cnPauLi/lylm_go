package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/admin"
	"lylme_spage/internal/config"
	"lylme_spage/internal/database"
	"lylme_spage/internal/install"
	"lylme_spage/internal/logger"
	"lylme_spage/internal/pages"
	"lylme_spage/internal/router"
	"lylme_spage/internal/site"
	"lylme_spage/internal/view"
)

// AppVersion 程序版本号
const AppVersion = "2.7.0"

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	log, err := logger.New(cfg.LogPath(), cfg.LogLevel)
	if err != nil {
		panic(err)
	}
	defer log.Close()

	installHandler, err := install.New(cfg, log)
	if err != nil {
		log.Errorf("初始化安装向导失败: %v", err)
		os.Exit(1)
	}

	// 数据库不可用时不再退出，改为仅提供安装向导的最小服务，便于全新机器安装
	db, dbErr := database.Open(cfg, log)
	var handler http.Handler
	if dbErr != nil {
		log.Errorf("数据库打开失败，进入安装模式: %v", dbErr)
		handler = installOnlyRouter(cfg, log, installHandler)
	} else {
		store, err := site.NewStore(db, cfg.RootDir)
		if err != nil {
			log.Errorf("初始化数据层失败: %v", err)
			os.Exit(1)
		}

		engine := view.NewEngine(store, AppVersion)

		pageEngine, err := pages.NewEngine(cfg.Path("web"), engine)
		if err != nil {
			log.Errorf("初始化页面模板失败: %v", err)
			os.Exit(1)
		}

		adminHandler, err := admin.New(cfg, store, log)
		if err != nil {
			log.Errorf("初始化后台模板失败: %v", err)
			os.Exit(1)
		}

		h := &router.Handler{Cfg: cfg, Store: store, Engine: engine, Pages: pageEngine, Admin: adminHandler, Log: log}
		r := router.Setup(h)
		// 主路由同样挂载安装向导，使 /install/ 在已连接数据库时也可用
		installHandler.Register(r, "/install")
		handler = r
	}

	srv := &http.Server{Addr: cfg.Addr(), Handler: handler}

	go func() {
		log.Infof("服务已启动：http://%s", cfg.Addr())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Errorf("服务启动失败: %v", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Infof("正在关闭服务...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Errorf("服务关闭异常: %v", err)
	}
}

// installOnlyRouter 数据库不可用时使用的最小路由：仅提供安装向导、安装静态资源与健康检查
func installOnlyRouter(cfg *config.Config, log *logger.Logger, ih *install.Handler) *gin.Engine {
	gin.SetMode(cfg.Mode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Infof("%s %s %d %s", c.Request.Method, c.Request.URL.RequestURI(), c.Writer.Status(), time.Since(start))
	})
	r.Use(install.Guard(cfg.RootDir))

	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	ih.Register(r, "/install")
	return r
}

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

	"lylme_spage/internal/config"
	"lylme_spage/internal/database"
	"lylme_spage/internal/logger"
)

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

	db, err := database.Open(cfg, log)
	if err != nil {
		log.Errorf("%v", err)
		os.Exit(1)
	}

	gin.SetMode(cfg.Mode)
	r := gin.New()
	r.Use(gin.Recovery(), requestLogger(log))

	// 健康检查
	r.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// 静态资源（原版 assets/template 目录原样提供）
	r.Static("/assets", cfg.Path("assets"))
	r.Static("/template", cfg.Path("template"))

	_ = db

	srv := &http.Server{
		Addr:    cfg.Addr(),
		Handler: r,
	}

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

// requestLogger 简单请求日志中间件
func requestLogger(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Infof("%s %s %d %s", c.Request.Method, c.Request.URL.Path, c.Writer.Status(), time.Since(start))
	}
}

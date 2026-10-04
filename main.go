package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lylme_spage/internal/admin"
	"lylme_spage/internal/config"
	"lylme_spage/internal/database"
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

	db, err := database.Open(cfg, log)
	if err != nil {
		log.Errorf("%v", err)
		os.Exit(1)
	}

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

	srv := &http.Server{Addr: cfg.Addr(), Handler: r}

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

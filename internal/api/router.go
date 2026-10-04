package api

import (
	"github.com/gin-gonic/gin"

	"lylme_spage/internal/service"
)

// RegisterV1 在 /api/v1 下注册全部 REST 端点（鉴权 + 限流由中间件统一处理）。
// 后续域的端点在各域 handler 包中追加注册。
func RegisterV1(r gin.IRouter, keys *service.APIKeyService) {
	v1 := r.Group("/api/v1", RequireKey(keys))

	// 自检
	v1.GET("/ping", func(c *gin.Context) { OK(c, gin.H{"pong": true}) })
}

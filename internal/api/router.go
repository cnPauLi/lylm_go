package api

import (
	"github.com/gin-gonic/gin"
)

// RegisterV1 在 /api/v1 下注册全部 REST 端点（鉴权 + 限流由中间件统一处理）。
// 各域路由注册函数分散于域 handler 文件，业务规则只存在于 service 层。
func RegisterV1(r gin.IRouter, d *Deps) {
	v1 := r.Group("/api/v1", RequireKey(d.Keys))

	// 自检
	v1.GET("/ping", func(c *gin.Context) { OK(c, gin.H{"pong": true}) })

	RegisterGroupRoutes(v1, d)
	RegisterLinkRoutes(v1, d)
	RegisterTagRoutes(v1, d)
	RegisterSouRoutes(v1, d)
	RegisterPwdRoutes(v1, d)
	RegisterSettingsRoutes(v1, d)
	RegisterApplyRoutes(v1, d)
	RegisterThemeRoutes(v1, d)
	RegisterDashboardRoutes(v1, d)
}

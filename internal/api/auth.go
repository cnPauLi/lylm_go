package api

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/service"
	"lylme_spage/internal/webutil"
)

// 默认限流参数：每个 IP 每分钟最多 120 次请求
const (
	defaultRateMax = 120
	rateWindow     = time.Minute
)

// RequireKey Bearer 密钥鉴权 + IP 限流（默认阈值）
func RequireKey(keys *service.APIKeyService) gin.HandlerFunc {
	return RequireKeyWithLimit(keys, defaultRateMax, rateWindow)
}

// RequireKeyWithLimit Bearer 鉴权，可注入限流阈值（max<=0 表示不限流），便于测试
func RequireKeyWithLimit(keys *service.APIKeyService, max int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := webutil.RealIP(c.Request)
		if max > 0 && !webutil.RateLimit("apikey:"+ip, max, window) {
			Fail(c, 429, 429, "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}

		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			FailUnauthorized(c, "缺少或格式错误的 Authorization 头，需提供 Bearer sk-...")
			c.Abort()
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if !strings.HasPrefix(token, "sk-") || !keys.Validate(token) {
			FailUnauthorized(c, "无效或已失效的 API 密钥")
			c.Abort()
			return
		}
		c.Next()
	}
}

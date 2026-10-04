package mcp

import (
	"net/http"

	"github.com/gin-gonic/gin"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"lylme_spage/internal/api"
)

// NewHandler 构建 MCP Streamable HTTP（无状态）处理器。
// 每个请求独立处理、不依赖会话粘滞、不返回 Mcp-Session-Id、无服务端主动通知。
func NewHandler(d *api.Deps) http.Handler {
	return mcpserver.NewStreamableHTTPServer(
		NewServer(d).server,
		mcpserver.WithStateLess(true),
	)
}

// Mount 在 Gin 引擎上挂载 MCP 端点：
// 显式注册 /mcp 与 /mcp/ 两条路径的 GET/POST/DELETE 方法，
// 全部请求先经 Bearer 密钥鉴权（含 IP 限流），再转发给 Streamable HTTP 处理器。
// 两条路径同时注册可避免 Gin 的 trailing-slash 自动重定向干扰；
// 不支持的 HTTP 方法由 MCP 处理器内部返回错误。
func Mount(r gin.IRouter, d *api.Deps) {
	h := NewHandler(d)
	auth := api.RequireKey(d.Keys)
	forward := func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
	for _, p := range []string{"/mcp", "/mcp/"} {
		r.GET(p, auth, forward)
		r.POST(p, auth, forward)
		r.DELETE(p, auth, forward)
	}
}

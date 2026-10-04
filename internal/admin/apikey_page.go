package admin

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/webutil"
)

// mcpServerName MCP 配置中本站服务的键名（与 MCP Server 名称保持一致）
const mcpServerName = "lylme-navigation"

// mcpPageData 「接口与 MCP」页面模板数据
type mcpPageData struct {
	HasKey      bool   // 是否已存在密钥
	Key         string // 密钥完整明文（仅后台登录会话可见）
	CreatedAt   string // 密钥创建时间（已格式化）
	APIBase     string // REST API 基础地址 <根>/api/v1
	MCPEndpoint string // MCP 接入地址 <根>/mcp
	MCPConfig   string // 已内嵌当前 key 的 MCP 配置 JSON
}

// mcpServerEntry MCP 客户端配置中的单个服务项
type mcpServerEntry struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

// mcpClientCfg Streamable HTTP 客户端格式的 MCP 配置
type mcpClientCfg struct {
	MCPServers map[string]mcpServerEntry `json:"mcpServers"`
}

// mcpConfigJSON 生成已内嵌 key 的 MCP 配置 JSON（纯函数，便于单测）：
// url 指向 <root>/mcp，headers.Authorization 为 Bearer <key>。
func mcpConfigJSON(root, key string) string {
	cfg := mcpClientCfg{
		MCPServers: map[string]mcpServerEntry{
			mcpServerName: {
				URL:     strings.TrimRight(root, "/") + "/mcp",
				Headers: map[string]string{"Authorization": "Bearer " + key},
			},
		},
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}

// MCPPage 接口与 MCP 管理页面（mcp），受登录中间件保护
func (h *Handler) MCPPage(c *gin.Context) {
	root := siteURL(c, 2)
	info := h.KeySvc.Get()
	page := &mcpPageData{
		HasKey:      info.Exists,
		APIBase:     strings.TrimRight(root, "/") + "/api/v1",
		MCPEndpoint: strings.TrimRight(root, "/") + "/mcp",
	}
	if info.Exists {
		page.Key = info.Key
		page.CreatedAt = info.CreatedAt.Format("2006-01-02 15:04:05")
		page.MCPConfig = mcpConfigJSON(root, info.Key)
	}
	data := h.newData(c, "接口与 MCP", "mcp", "mcp")
	data.Page = page
	h.render(c, "mcp.html", data)
}

// AjaxMCP 密钥管理 JSON 端点（ajax_mcp?set=ensure|regenerate），
// 经 frag 鉴权与 CSRF 校验，响应风格与现有 admin ajax 一致：{code,msg,data}。
func (h *Handler) AjaxMCP(c *gin.Context) {
	if c.Request.Method != http.MethodPost {
		jsonWrite(c, gin.H{"code": -1, "msg": "非法请求！"})
		return
	}
	token := c.GetHeader("X-CSRF-Token")
	if token == "" {
		token = c.PostForm("csrf_token")
	}
	if !webutil.CSRFVerifyToken(c, token) {
		jsonWrite(c, gin.H{"code": -1, "msg": "CSRF校验失败，请刷新页面后重试"})
		return
	}

	var (
		key string
		err error
		msg string
	)
	switch c.Query("set") {
	case "ensure":
		key, err = h.KeySvc.Ensure()
		msg = "操作成功！"
	case "regenerate":
		key, err = h.KeySvc.Regenerate()
		msg = "密钥已重新生成，旧密钥立即失效！"
	default:
		jsonWrite(c, gin.H{"code": -1, "msg": "非法请求！"})
		return
	}
	if err != nil {
		h.Log.Errorf("密钥操作失败 set=%s: %v", c.Query("set"), err)
		jsonWrite(c, gin.H{"code": 100, "msg": "操作失败，请重试"})
		return
	}

	info := h.KeySvc.Get()
	root := siteURL(c, 2)
	jsonWrite(c, gin.H{
		"code": 200,
		"msg":  msg,
		"data": gin.H{
			"key":          key,
			"created_at":   info.CreatedAt.Format("2006-01-02 15:04:05"),
			"api_base":     strings.TrimRight(root, "/") + "/api/v1",
			"mcp_endpoint": strings.TrimRight(root, "/") + "/mcp",
			"mcp_config":   mcpConfigJSON(root, key),
		},
	})
}

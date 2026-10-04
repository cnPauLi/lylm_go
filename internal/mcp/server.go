// Package mcp 提供六零导航页的 MCP（Model Context Protocol）服务。
//
// 本包为协议适配层：MCP 工具与 /api/v1 REST 操作一一对应，命名统一为
// 「<域>_<动作>」；所有业务规则一律调用 internal/service，不在此层重复实现。
// 传输使用 Streamable HTTP 无状态模式（见 http.go）。
//
// 依赖方向：mcp → api(Deps)/service → site/models/theme → config，禁止反向依赖。
package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"lylme_spage/internal/api"
	"lylme_spage/internal/service"
)

// MCPSet 汇总 MCP 服务实例与全部工具注册。
type MCPSet struct {
	deps   *api.Deps
	server *mcpserver.MCPServer
}

// NewServer 基于与 REST API 相同的 Deps 构建 MCP 服务，并注册全部工具。
func NewServer(d *api.Deps) *MCPSet {
	s := mcpserver.NewMCPServer(
		"lylme-navigation",
		d.AppVersion,
		mcpserver.WithToolCapabilities(true),
	)
	set := &MCPSet{deps: d, server: s}

	set.registerGroupTools()
	set.registerLinkTools()
	set.registerTagTools()
	set.registerSouTools()
	set.registerPwdTools()
	set.registerSettingsTools()
	set.registerApplyTools()
	set.registerThemeTools()
	set.registerDashboardTools()

	return set
}

// add 注册一个工具及其处理器。
func (set *MCPSet) add(tool mcp.Tool, h mcpserver.ToolHandlerFunc) {
	set.server.AddTool(tool, h)
}

// ----------------------- 工具结果辅助 -----------------------

// jsonResult 将出参序列化为 JSON 文本工具结果；JSON 结构与对应 REST 端点的 data 一致。
func jsonResult(v any) *mcp.CallToolResult {
	b, err := json.Marshal(v)
	if err != nil {
		// 序列化失败属于本服务内部问题，不向外暴露细节
		return mcp.NewToolResultError("结果序列化失败")
	}
	return mcp.NewToolResultText(string(b))
}

// expectedServiceError 判断是否为可对外说明的哨兵错误（参数错误/记录不存在等）。
func expectedServiceError(err error) bool {
	switch {
	case errors.Is(err, service.ErrInvalidInput),
		errors.Is(err, service.ErrGroupNotFound),
		errors.Is(err, service.ErrLinkNotFound),
		errors.Is(err, service.ErrTagNotFound),
		errors.Is(err, service.ErrSouNotFound),
		errors.Is(err, service.ErrPwdNotFound),
		errors.Is(err, service.ErrThemeNotFound),
		errors.Is(err, service.ErrApplyNotFound),
		errors.Is(err, service.ErrWrongPassword),
		errors.Is(err, service.ErrNoChange):
		return true
	}
	return false
}

// serviceErrorResult 按 service 错误返回 IsError 工具结果：
// 哨兵错误附带明确说明；未知内部错误统一提示，不泄露堆栈与内部信息。
func serviceErrorResult(err error) *mcp.CallToolResult {
	if expectedServiceError(err) {
		return mcp.NewToolResultError(err.Error())
	}
	return mcp.NewToolResultError("操作失败，请稍后再试")
}

// ----------------------- 参数解析辅助 -----------------------
//
// JSON 经 encoding/json 解析到 map[string]any 后，数字为 float64、
// 数组为 []any、对象为 map[string]any；下列辅助统一处理缺失与类型错误。

func requiredString(args map[string]any, key string) (string, error) {
	v, ok := args[key]
	if !ok {
		return "", fmt.Errorf("缺少必填参数 %q", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("参数 %q 必须是字符串", key)
	}
	return s, nil
}

func optionalString(args map[string]any, key, def string) string {
	v, ok := args[key]
	if !ok {
		return def
	}
	if s, ok := v.(string); ok {
		return s
	}
	return def
}

func toInt(v any, key string) (int, error) {
	switch n := v.(type) {
	case float64:
		return int(n), nil
	case int:
		return n, nil
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(n))
		if err != nil {
			return 0, fmt.Errorf("参数 %q 必须是整数", key)
		}
		return i, nil
	default:
		return 0, fmt.Errorf("参数 %q 必须是整数", key)
	}
}

func requiredInt(args map[string]any, key string) (int, error) {
	v, ok := args[key]
	if !ok {
		return 0, fmt.Errorf("缺少必填参数 %q", key)
	}
	return toInt(v, key)
}

func optionalInt(args map[string]any, key string, def int) int {
	v, ok := args[key]
	if !ok {
		return def
	}
	n, err := toInt(v, key)
	if err != nil {
		return def
	}
	return n
}

func requiredInts(args map[string]any, key string) ([]int, error) {
	v, ok := args[key]
	if !ok {
		return nil, fmt.Errorf("缺少必填参数 %q", key)
	}
	arr, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("参数 %q 必须是整数数组", key)
	}
	out := make([]int, 0, len(arr))
	for i, e := range arr {
		n, err := toInt(e, fmt.Sprintf("%s[%d]", key, i))
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

func requiredStrings(args map[string]any, key string) ([]string, error) {
	v, ok := args[key]
	if !ok {
		return nil, fmt.Errorf("缺少必填参数 %q", key)
	}
	arr, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("参数 %q 必须是字符串数组", key)
	}
	out := make([]string, 0, len(arr))
	for i, e := range arr {
		s, ok := e.(string)
		if !ok {
			return nil, fmt.Errorf("参数 %s[%d] 必须是字符串", key, i)
		}
		out = append(out, s)
	}
	return out, nil
}

func optionalStrings(args map[string]any, key string, def []string) []string {
	v, ok := args[key]
	if !ok {
		return def
	}
	arr, ok := v.([]any)
	if !ok {
		return def
	}
	out := make([]string, 0, len(arr))
	for _, e := range arr {
		if s, ok := e.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func requiredObject(args map[string]any, key string) (map[string]any, error) {
	v, ok := args[key]
	if !ok {
		return nil, fmt.Errorf("缺少必填参数 %q", key)
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("参数 %q 必须是 JSON 对象", key)
	}
	return m, nil
}

func optionalObject(args map[string]any, key string, def map[string]any) map[string]any {
	v, ok := args[key]
	if !ok {
		return def
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return def
}

func optionalBool(args map[string]any, key string, def bool) bool {
	v, ok := args[key]
	if !ok {
		return def
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return def
}

// itoa 整数转字符串。
func itoa(v int) string { return strconv.Itoa(v) }

// enumInt 校验 v 是否落在允许的整数枚举内。
func enumInt(v int, allowed ...int) bool {
	for _, a := range allowed {
		if v == a {
			return true
		}
	}
	return false
}

package mcp

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"lylme_spage/internal/service"
)

// 敏感配置键黑名单（与 REST GET /settings 一致）
var mcpSensitiveConfigKeys = map[string]struct{}{
	"admin_pwd": {},
	"api_key":   {},
}

const mcpMaskedKeyHead = 4

// registerSettingsTools 注册设置域工具：网站设置、关于页、管理员账号、系统安全、
// 微信推送、网站授权、API 密钥；与 /api/v1 对应端点一一对应。
func (set *MCPSet) registerSettingsTools() {
	d := set.deps

	// settings_get：GET /settings
	set.add(mcp.NewTool("settings_get",
		mcp.WithDescription("读取全部站点配置项（管理员密码哈希、API 密钥等敏感键会被剔除）。"),
	), func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		conf := d.Config.AllConfig()
		for k := range mcpSensitiveConfigKeys {
			delete(conf, k)
		}
		return jsonResult(conf), nil
	})

	// settings_update：PUT /settings
	set.add(mcp.NewTool("settings_update",
		mcp.WithDescription("批量更新网站基本设置（名称、LOGO、背景、关键词、描述、版权、CDN 等）；空字符串表示清空对应项。"),
		mcp.WithString("title", mcp.Description("网站名称")),
		mcp.WithString("logo", mcp.Description("网站 LOGO")),
		mcp.WithString("background", mcp.Description("PC 端背景图片")),
		mcp.WithString("wap_background", mcp.Description("手机端背景图片")),
		mcp.WithString("keywords", mcp.Description("网站关键词")),
		mcp.WithString("description", mcp.Description("网站描述")),
		mcp.WithString("mode", mcp.Description("网站运行模式")),
		mcp.WithString("snapshot", mcp.Description("详情页快照 API")),
		mcp.WithString("copyright", mcp.Description("底部版权")),
		mcp.WithString("icp", mcp.Description("ICP 备案号")),
		mcp.WithString("wztj", mcp.Description("自定义 footer 内容")),
		mcp.WithString("cdnpublic", mcp.Description("CDN 地址")),
		mcp.WithString("yan", mcp.Description("自定义统计：仅恰好传 \"true\" 时开启，其余一律关闭")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		if err := d.Config.UpdateSiteSettings(service.SiteSettingsInput{
			Title:         optionalString(args, "title", ""),
			Logo:          optionalString(args, "logo", ""),
			Background:    optionalString(args, "background", ""),
			WapBackground: optionalString(args, "wap_background", ""),
			Keywords:      optionalString(args, "keywords", ""),
			Description:   optionalString(args, "description", ""),
			Mode:          optionalString(args, "mode", ""),
			Snapshot:      optionalString(args, "snapshot", ""),
			Copyright:     optionalString(args, "copyright", ""),
			ICP:           optionalString(args, "icp", ""),
			WzTJ:          optionalString(args, "wztj", ""),
			CDNPublic:     optionalString(args, "cdnpublic", ""),
			Yan:           optionalString(args, "yan", ""),
		}); err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"updated": true}), nil
	})

	// settings_about_get：GET /settings/about
	set.add(mcp.NewTool("settings_about_get",
		mcp.WithDescription("读取关于页 HTML 内容；未配置时返回内置默认内容。"),
	), func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return jsonResult(map[string]any{"content": d.Config.AboutContent()}), nil
	})

	// settings_about_update：PUT /settings/about
	set.add(mcp.NewTool("settings_about_update",
		mcp.WithDescription("更新关于页 HTML 内容。"),
		mcp.WithString("content", mcp.Required(), mcp.Description("关于页 HTML 内容（必填）")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		content, err := requiredString(args, "content")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if err := d.Config.UpdateAboutContent(content); err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"updated": true}), nil
	})

	// settings_about_reset：POST /settings/about/reset
	set.add(mcp.NewTool("settings_about_reset",
		mcp.WithDescription("将关于页内容恢复为内置默认内容，并返回恢复后的内容。"),
	), func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if err := d.Config.ResetAboutContent(); err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"content": service.DefaultAboutContent}), nil
	})

	// settings_account_update：PUT /settings/account
	set.add(mcp.NewTool("settings_account_update",
		mcp.WithDescription("修改管理员用户名和/或密码。必须提供当前密码校验；新密码需与确认密码一致。出于安全考虑响应不回显新密码。"),
		mcp.WithString("old_password", mcp.Required(), mcp.Description("当前管理员密码（必填）")),
		mcp.WithString("username", mcp.Description("新用户名（可空，空表示不改用户名）")),
		mcp.WithString("new_password", mcp.Description("新密码（可空，空表示不改密码）")),
		mcp.WithString("confirm_password", mcp.Description("确认新密码（与新密码一致）")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		oldPassword, err := requiredString(args, "old_password")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		res, err := d.Config.UpdateAccount(service.AccountInput{
			Username:        optionalString(args, "username", ""),
			OldPassword:     oldPassword,
			NewPassword:     optionalString(args, "new_password", ""),
			ConfirmPassword: optionalString(args, "confirm_password", ""),
		})
		if err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{
			"changed_user": res.ChangedUser,
			"changed_pwd":  res.ChangedPwd,
			"username":     res.Username,
		}), nil
	})

	// settings_system_get：GET /settings/system
	set.add(mcp.NewTool("settings_system_get",
		mcp.WithDescription("读取系统安全设置：后台目录（admin_path）与调试模式（debug）。"),
	), func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		conf := d.Config.AllConfig()
		adminPath := conf["admin_path"]
		if adminPath == "" {
			adminPath = d.Cfg.AdminPath
			if adminPath == "" {
				adminPath = "admin"
			}
		}
		debug := conf["debug"]
		if debug != "1" {
			debug = "0"
		}
		return jsonResult(map[string]any{"admin_path": adminPath, "debug": debug}), nil
	})

	// settings_system_update：PUT /settings/system
	set.add(mcp.NewTool("settings_system_update",
		mcp.WithDescription("修改系统安全设置：后台目录（仅字母/数字/下划线/中划线）与调试模式。校验全部通过才会写入。"),
		mcp.WithString("admin_path", mcp.Required(), mcp.Description("后台目录（必填，仅支持字母、数字、下划线、中划线）")),
		mcp.WithString("debug", mcp.Required(), mcp.Enum("0", "1"), mcp.Description("调试模式（必填）：0 关闭，1 开启")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		adminPath, err := requiredString(args, "admin_path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		debug, err := requiredString(args, "debug")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if err := d.Config.UpdateSystem(service.SystemInput{
			AdminPath: adminPath,
			Debug:     debug,
		}); err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"updated": true}), nil
	})

	// wxplus_get：GET /wxplus
	set.add(mcp.NewTool("wxplus_get",
		mcp.WithDescription("读取微信推送配置：推送密钥（key）与每日推送时间（time）。"),
	), func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cfg := d.Config.WxPlus()
		return jsonResult(map[string]any{"key": cfg.Key, "time": cfg.Time}), nil
	})

	// wxplus_update：PUT /wxplus
	set.add(mcp.NewTool("wxplus_update",
		mcp.WithDescription("更新微信推送密钥与每日推送时间；空密钥表示关闭推送。"),
		mcp.WithString("key", mcp.Description("微信推送密钥（空串表示关闭）")),
		mcp.WithString("time", mcp.Description("每日推送时间")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		if err := d.Config.UpdateWxPlus(
			optionalString(args, "key", ""),
			optionalString(args, "time", ""),
		); err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"updated": true}), nil
	})

	// license_get：GET /license
	set.add(mcp.NewTool("license_get",
		mcp.WithDescription("读取网站授权配置：授权码（code）与域名获取方式（hostmode，1=HTTP_HOST / 2=SERVER_NAME）。"),
	), func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cfg := d.Config.License()
		return jsonResult(map[string]any{"code": cfg.Code, "hostmode": cfg.HostMode}), nil
	})

	// license_update：PUT /license
	set.add(mcp.NewTool("license_update",
		mcp.WithDescription("更新网站授权码与域名获取方式。"),
		mcp.WithString("code", mcp.Description("程序授权码")),
		mcp.WithString("hostmode", mcp.Required(), mcp.Enum("1", "2"), mcp.Description("域名获取方式（必填）：1 HTTP_HOST，2 SERVER_NAME")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		hostmode, err := requiredString(args, "hostmode")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if err := d.Config.UpdateLicense(
			optionalString(args, "code", ""),
			hostmode,
		); err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"updated": true}), nil
	})

	// apikey_get：GET /apikey
	set.add(mcp.NewTool("apikey_get",
		mcp.WithDescription("读取当前 API/MCP 密钥信息：是否存在、创建时间与掩码前缀；不会返回完整明文。"),
	), func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		info := d.Keys.Get()
		masked := ""
		if info.Exists {
			masked = mcpMaskAPIKey(info.Key)
		}
		return jsonResult(map[string]any{
			"exists":     info.Exists,
			"created_at": mcpFormatCreatedAt(info.CreatedAt),
			"masked":     masked,
		}), nil
	})

	// apikey_regenerate：POST /apikey/regenerate
	set.add(mcp.NewTool("apikey_regenerate",
		mcp.WithDescription("重新生成 API/MCP 密钥。旧密钥在成功的瞬间立即失效，所有客户端必须更换新密钥。这是唯一会返回完整密钥明文的时机。"),
	), func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key, err := d.Keys.Regenerate()
		if err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{
			"key":        key,
			"created_at": mcpFormatCreatedAt(d.Keys.Get().CreatedAt),
		}), nil
	})
}

// mcpMaskAPIKey 返回密钥掩码（如 sk-1a2b…），不暴露完整明文。
func mcpMaskAPIKey(key string) string {
	if len(key) < len("sk-")+mcpMaskedKeyHead {
		return "sk-…"
	}
	return key[:len("sk-")+mcpMaskedKeyHead] + "…"
}

// mcpFormatCreatedAt 将时间格式化为 RFC3339；零值返回空串。
func mcpFormatCreatedAt(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

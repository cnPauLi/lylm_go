package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	"lylme_spage/internal/service"
)

// registerSouTools 注册搜索引擎（sou）域工具，与 /api/v1/sou 端点一一对应。
func (set *MCPSet) registerSouTools() {
	d := set.deps

	// sou_list：GET /sou
	set.add(mcp.NewTool("sou_list",
		mcp.WithDescription("查询全部搜索引擎（按 sou_order 升序），返回列表与总数。"),
	), func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		rows, err := d.Sou.List()
		if err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"list": rows, "total": len(rows)}), nil
	})

	// sou_create：POST /sou
	set.add(mcp.NewTool("sou_create",
		mcp.WithDescription("新增一个搜索引擎。别名与名称必填；排序自动追加到末尾。"),
		mcp.WithString("alias", mcp.Required(), mcp.Description("搜索引擎别名（前台 data-alias 使用，必填，最长 30 字符）")),
		mcp.WithString("name", mcp.Required(), mcp.Description("搜索引擎名称（必填，最长 30 字符）")),
		mcp.WithString("hint", mcp.Description("输入框提示语（最长 60 字符）")),
		mcp.WithString("color", mcp.Description("标识颜色，如 #2932e1（最长 20 字符）")),
		mcp.WithString("link", mcp.Description("PC 端搜索链接模板（最长 255 字符）")),
		mcp.WithString("waplink", mcp.Description("移动端搜索链接模板（最长 255 字符）")),
		mcp.WithString("icon", mcp.Description("图标：URL 或 SVG 名称/内容（最长 2000 字符）")),
		mcp.WithBoolean("st", mcp.Description("是否启用：true 启用，false 停用；不传默认 false")),
		mcp.WithNumber("order", mcp.Description("排序值（新增时通常不传，由服务端自动追加）")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		alias, err := requiredString(args, "alias")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		name, err := requiredString(args, "name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		row, err := d.Sou.Create(service.SouInput{
			Alias:   alias,
			Name:    name,
			Hint:    optionalString(args, "hint", ""),
			Color:   optionalString(args, "color", ""),
			Link:    optionalString(args, "link", ""),
			Waplink: optionalString(args, "waplink", ""),
			Icon:    optionalString(args, "icon", ""),
			St:      souStFromBool(optionalBool(args, "st", false)),
			Order:   optionalInt(args, "order", 0),
		})
		if err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(row), nil
	})

	// sou_update：PUT /sou/:id
	set.add(mcp.NewTool("sou_update",
		mcp.WithDescription("修改指定搜索引擎的全部字段，包括排序值。"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("搜索引擎 ID（必填，正整数）")),
		mcp.WithString("alias", mcp.Required(), mcp.Description("搜索引擎别名（必填，最长 30 字符）")),
		mcp.WithString("name", mcp.Required(), mcp.Description("搜索引擎名称（必填，最长 30 字符）")),
		mcp.WithString("hint", mcp.Description("输入框提示语（最长 60 字符）")),
		mcp.WithString("color", mcp.Description("标识颜色（最长 20 字符）")),
		mcp.WithString("link", mcp.Description("PC 端搜索链接模板（最长 255 字符）")),
		mcp.WithString("waplink", mcp.Description("移动端搜索链接模板（最长 255 字符）")),
		mcp.WithString("icon", mcp.Description("图标：URL 或 SVG 名称/内容（最长 2000 字符）")),
		mcp.WithBoolean("st", mcp.Description("是否启用：true 启用，false 停用；不传默认启用")),
		mcp.WithNumber("order", mcp.Description("排序值（修改时写入 sou_order，不传为 0）")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		id, err := requiredInt(args, "id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		alias, err := requiredString(args, "alias")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		name, err := requiredString(args, "name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		row, err := d.Sou.Update(id, service.SouInput{
			Alias:   alias,
			Name:    name,
			Hint:    optionalString(args, "hint", ""),
			Color:   optionalString(args, "color", ""),
			Link:    optionalString(args, "link", ""),
			Waplink: optionalString(args, "waplink", ""),
			Icon:    optionalString(args, "icon", ""),
			St:      souStFromBool(optionalBool(args, "st", false)),
			Order:   optionalInt(args, "order", 0),
		})
		if err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(row), nil
	})

	// sou_delete：DELETE /sou/:id
	set.add(mcp.NewTool("sou_delete",
		mcp.WithDescription("删除指定的搜索引擎。"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("搜索引擎 ID（必填，正整数）")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		id, err := requiredInt(args, "id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if _, err := d.Sou.Get(id); err != nil {
			return serviceErrorResult(err), nil
		}
		if err := d.Sou.Delete(id); err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"id": id}), nil
	})
}

// souStFromBool 布尔转落库整数（true=1 / false=0）
func souStFromBool(st bool) int {
	if st {
		return 1
	}
	return 0
}

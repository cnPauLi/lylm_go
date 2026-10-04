package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	"lylme_spage/internal/service"
)

// registerTagTools 注册导航菜单（tag）域工具，与 /api/v1/tags 端点一一对应。
func (set *MCPSet) registerTagTools() {
	d := set.deps

	// tag_list：GET /tags
	set.add(mcp.NewTool("tag_list",
		mcp.WithDescription("查询全部顶部导航菜单（按 sort 升序），返回列表与总数。"),
	), func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		rows, err := d.Tag.List()
		if err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"list": rows, "total": len(rows)}), nil
	})

	// tag_create：POST /tags
	set.add(mcp.NewTool("tag_create",
		mcp.WithDescription("新增一个顶部导航菜单。名称与链接必填；sort 为 0 或不传时取默认值 10。"),
		mcp.WithString("name", mcp.Required(), mcp.Description("导航名称（必填，最长 30 字符）")),
		mcp.WithString("link", mcp.Required(), mcp.Description("导航链接（必填，最长 255 字符）")),
		mcp.WithNumber("sort", mcp.Description("排序值（升序），0 或不传时默认 10")),
		mcp.WithNumber("target", mcp.Description("打开方式：1 新标签页，0 当前页（默认 0）")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		name, err := requiredString(args, "name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		link, err := requiredString(args, "link")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		target := optionalInt(args, "target", 0)
		if !enumInt(target, 0, 1) {
			return mcp.NewToolResultError("参数 \"target\" 只能是 0 或 1"), nil
		}
		row, err := d.Tag.Create(service.TagInput{
			Name:   name,
			Link:   link,
			Sort:   optionalInt(args, "sort", 0),
			Target: target,
		})
		if err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(row), nil
	})

	// tag_update：PUT /tags/:id
	set.add(mcp.NewTool("tag_update",
		mcp.WithDescription("修改指定的顶部导航菜单。名称与链接必填。"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("导航菜单 ID（必填，正整数）")),
		mcp.WithString("name", mcp.Required(), mcp.Description("导航名称（必填，最长 30 字符）")),
		mcp.WithString("link", mcp.Required(), mcp.Description("导航链接（必填，最长 255 字符）")),
		mcp.WithNumber("sort", mcp.Description("排序值（升序），0 或不传时默认 10")),
		mcp.WithNumber("target", mcp.Description("打开方式：1 新标签页，0 当前页（默认 0）")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		id, err := requiredInt(args, "id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		name, err := requiredString(args, "name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		link, err := requiredString(args, "link")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		target := optionalInt(args, "target", 0)
		if !enumInt(target, 0, 1) {
			return mcp.NewToolResultError("参数 \"target\" 只能是 0 或 1"), nil
		}
		row, err := d.Tag.Update(id, service.TagInput{
			Name:   name,
			Link:   link,
			Sort:   optionalInt(args, "sort", 0),
			Target: target,
		})
		if err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(row), nil
	})

	// tag_delete：DELETE /tags/:id
	set.add(mcp.NewTool("tag_delete",
		mcp.WithDescription("删除指定的顶部导航菜单。"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("导航菜单 ID（必填，正整数）")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		id, err := requiredInt(args, "id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if _, err := d.Tag.Get(id); err != nil {
			return serviceErrorResult(err), nil
		}
		if err := d.Tag.Delete(id); err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"id": id}), nil
	})
}

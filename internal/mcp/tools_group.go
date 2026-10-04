package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	"lylme_spage/internal/service"
)

// registerGroupTools 注册分组（group）域工具，与 /api/v1/groups 端点一一对应。
func (set *MCPSet) registerGroupTools() {
	d := set.deps

	// group_list：GET /groups
	set.add(mcp.NewTool("group_list",
		mcp.WithDescription("查询全部分组（按 group_order 升序），返回分组列表、总数以及各组链接计数。"),
	), func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		list, err := d.Group.List()
		if err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{
			"total":      list.Total,
			"groups":     list.Groups,
			"link_count": list.LinkCount,
		}), nil
	})

	// group_create：POST /groups
	set.add(mcp.NewTool("group_create",
		mcp.WithDescription("新增一个链接分组。纯 URL 图标会自动包裹 img 标签；排序自动追加到末尾。"),
		mcp.WithString("name", mcp.Required(), mcp.Description("分组名称（必填，最长 30 字符）")),
		mcp.WithString("icon", mcp.Description("分组图标：图片 URL 或 SVG/HTML 片段（可空）")),
		mcp.WithNumber("pwd", mcp.Description("关联的加密访问密码组 ID，0 表示不加密（默认 0）")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		name, err := requiredString(args, "name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		row, err := d.Group.Create(service.GroupInput{
			Name: name,
			Icon: optionalString(args, "icon", ""),
			Pwd:  optionalInt(args, "pwd", 0),
		})
		if err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"id": row.GroupID, "group": row}), nil
	})

	// group_update：PUT /groups/:id
	set.add(mcp.NewTool("group_update",
		mcp.WithDescription("修改指定分组的名称、图标或关联加密组。"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("分组 ID（必填，正整数）")),
		mcp.WithString("name", mcp.Required(), mcp.Description("分组名称（必填，最长 30 字符）")),
		mcp.WithString("icon", mcp.Description("分组图标：图片 URL 或 SVG/HTML 片段（可空）")),
		mcp.WithNumber("pwd", mcp.Description("关联的加密访问密码组 ID，0 表示不加密（默认 0）")),
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
		row, err := d.Group.Update(id, service.GroupInput{
			Name: name,
			Icon: optionalString(args, "icon", ""),
			Pwd:  optionalInt(args, "pwd", 0),
		})
		if err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"id": row.GroupID, "group": row}), nil
	})

	// group_delete：DELETE /groups/:id
	set.add(mcp.NewTool("group_delete",
		mcp.WithDescription("删除指定分组，同时删除该分组内的全部链接，此操作不可恢复。"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("分组 ID（必填，正整数）")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		id, err := requiredInt(args, "id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		// 与 REST 一致：先校验存在性（不存在返回错误），再执行删除
		if _, err := d.Group.Get(id); err != nil {
			return serviceErrorResult(err), nil
		}
		if err := d.Group.Delete(id); err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"id": id}), nil
	})

	// group_set_status：POST /groups/:id/status
	set.add(mcp.NewTool("group_set_status",
		mcp.WithDescription("启用或停用指定分组。停用后前台不再展示该分组。"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("分组 ID（必填，正整数）")),
		mcp.WithNumber("status", mcp.Required(), mcp.Description("目标状态：1 启用，0 停用")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		id, err := requiredInt(args, "id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		status, err := requiredInt(args, "status")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if !enumInt(status, 0, 1) {
			return mcp.NewToolResultError("参数 \"status\" 只能是 0 或 1"), nil
		}
		if _, err := d.Group.Get(id); err != nil {
			return serviceErrorResult(err), nil
		}
		if err := d.Group.SetStatus(id, status); err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"id": id, "status": status}), nil
	})

	// group_sort：POST /groups/sort
	set.add(mcp.NewTool("group_sort",
		mcp.WithDescription("按传入的分组 ID 顺序重新排列分组展示顺序。"),
		mcp.WithArray("ids",
			mcp.Required(),
			mcp.Items(map[string]any{"type": "number"}),
			mcp.MinItems(1),
			mcp.Description("分组 ID 列表，数组顺序即新的展示顺序（至少 1 个）"),
		),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		ids, err := requiredInts(args, "ids")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if len(ids) == 0 {
			return mcp.NewToolResultError("参数 \"ids\" 至少包含 1 个分组 ID"), nil
		}
		if err := d.Group.Sort(ids); err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"ids": ids}), nil
	})
}

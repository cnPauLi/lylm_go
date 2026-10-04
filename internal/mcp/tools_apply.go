package mcp

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"lylme_spage/internal/models"
	"lylme_spage/internal/service"
)

// mcpApplyItem 收录申请列表/详情元素（结构与 REST applyItem 一致）
type mcpApplyItem struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	GroupID   int       `json:"group_id"`
	GroupName string    `json:"group_name"`
	Icon      string    `json:"icon"`
	Desc      string    `json:"desc"`
	ApplyTime time.Time `json:"apply_time"`
	Status    int       `json:"status"`
}

func toMcpApplyItem(row models.Apply, groupMap map[int]string) mcpApplyItem {
	return mcpApplyItem{
		ID:        row.ApplyID,
		Name:      row.ApplyName,
		URL:       row.ApplyURL,
		GroupID:   row.ApplyGroup,
		GroupName: groupMap[row.ApplyGroup],
		Icon:      row.ApplyIcon,
		Desc:      row.ApplyDesc,
		ApplyTime: row.ApplyTime,
		Status:    row.ApplyStatus,
	}
}

// registerApplyTools 注册收录申请（apply）域工具，与 /api/v1/apply 端点一一对应。
func (set *MCPSet) registerApplyTools() {
	d := set.deps

	// apply_list：GET /apply
	set.add(mcp.NewTool("apply_list",
		mcp.WithDescription("分页查询收录申请，按申请时间倒序。不传 status 返回全部；可按状态过滤（0 待审核 / 1 已通过 / 2 已拒绝）。"),
		mcp.WithNumber("status", mcp.Description("状态过滤：0 待审核，1 已通过，2 已拒绝；不传返回全部")),
		mcp.WithNumber("page", mcp.Description("页码，从 1 开始，默认 1")),
		mcp.WithNumber("page_size", mcp.Description("每页数量，默认 10，最大 100")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		status := optionalInt(args, "status", -1)
		if status >= 0 && !enumInt(status, 0, 1, 2) {
			return mcp.NewToolResultError("参数 \"status\" 只能是 0、1 或 2"), nil
		}
		page := optionalInt(args, "page", 1)
		if page < 1 {
			page = 1
		}
		pageSize := optionalInt(args, "page_size", 10)
		if pageSize < 1 {
			pageSize = 10
		}
		if pageSize > 100 {
			pageSize = 100
		}

		res, err := d.Apply.List(status)
		if err != nil {
			return serviceErrorResult(err), nil
		}

		total := int64(len(res.Rows))
		start := (page - 1) * pageSize
		end := start + pageSize
		if start > len(res.Rows) {
			start = len(res.Rows)
		}
		if end > len(res.Rows) {
			end = len(res.Rows)
		}
		items := make([]mcpApplyItem, 0, end-start)
		for _, row := range res.Rows[start:end] {
			items = append(items, toMcpApplyItem(row, res.GroupMap))
		}
		return jsonResult(map[string]any{
			"list":      items,
			"page":      page,
			"page_size": pageSize,
			"total":     total,
		}), nil
	})

	// apply_settings_get：GET /apply/settings
	set.add(mcp.NewTool("apply_settings_get",
		mcp.WithDescription("读取收录设置：开关（open：0 开启需审核 / 1 开启免审核 / 2 关闭）与公告（gg）。"),
	), func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s := d.Apply.Settings()
		return jsonResult(map[string]any{"open": s.Open, "gg": s.GG}), nil
	})

	// apply_settings_update：PUT /apply/settings
	set.add(mcp.NewTool("apply_settings_update",
		mcp.WithDescription("更新收录开关与公告。"),
		mcp.WithNumber("open", mcp.Required(), mcp.Description("开关（必填）：0 开启需审核，1 开启免审核，2 关闭")),
		mcp.WithString("gg", mcp.Description("收录公告（最长 255 字符）")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		open, err := requiredInt(args, "open")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if !enumInt(open, 0, 1, 2) {
			return mcp.NewToolResultError("参数 \"open\" 只能是 0、1 或 2"), nil
		}
		gg := optionalString(args, "gg", "")
		if err := d.Apply.UpdateSettings(open, gg); err != nil {
			return serviceErrorResult(err), nil
		}
		saved := d.Apply.Settings()
		return jsonResult(map[string]any{"open": saved.Open, "gg": saved.GG}), nil
	})

	// apply_update：PUT /apply/:id
	set.add(mcp.NewTool("apply_update",
		mcp.WithDescription("修改指定收录申请的名称、URL、分组或图标。"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("申请 ID（必填，正整数）")),
		mcp.WithString("name", mcp.Required(), mcp.Description("申请名称（必填，最长 60 字符）")),
		mcp.WithString("url", mcp.Description("申请 URL（最长 255 字符）")),
		mcp.WithNumber("group_id", mcp.Description("所属分组 ID")),
		mcp.WithString("icon", mcp.Description("图标（URL 或 SVG）")),
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
		if _, err := d.Apply.Get(id); err != nil {
			return serviceErrorResult(err), nil
		}
		if err := d.Apply.Update(id, service.ApplyUpdateInput{
			Name:    name,
			URL:     optionalString(args, "url", ""),
			GroupID: optionalInt(args, "group_id", 0),
			Icon:    optionalString(args, "icon", ""),
		}); err != nil {
			return serviceErrorResult(err), nil
		}
		row, err := d.Apply.Get(id)
		if err != nil {
			return serviceErrorResult(err), nil
		}
		groupName := ""
		if g, err := d.Group.Get(row.ApplyGroup); err == nil {
			groupName = g.GroupName
		}
		return jsonResult(map[string]any{
			"id":      id,
			"updated": true,
			"row":     toMcpApplyItem(row, map[int]string{row.ApplyGroup: groupName}),
		}), nil
	})

	// apply_review：POST /apply/:id/review
	set.add(mcp.NewTool("apply_review",
		mcp.WithDescription("审核收录申请：status=1 通过（自动写入链接表，仅对待审核记录生效、重复审核幂等）；status=2 拒绝。"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("申请 ID（必填，正整数）")),
		mcp.WithNumber("status", mcp.Required(), mcp.Description("审核结果（必填）：1 通过，2 拒绝")),
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
		if !enumInt(status, 1, 2) {
			return mcp.NewToolResultError("参数 \"status\" 只能是 1 或 2"), nil
		}
		if _, err := d.Apply.Get(id); err != nil {
			return serviceErrorResult(err), nil
		}
		if err := d.Apply.SetStatus([]int{id}, status); err != nil {
			return serviceErrorResult(err), nil
		}
		row, err := d.Apply.Get(id)
		if err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"id": id, "status": row.ApplyStatus}), nil
	})

	// apply_delete：DELETE /apply
	set.add(mcp.NewTool("apply_delete",
		mcp.WithDescription("按 ID 列表批量删除收录申请。"),
		mcp.WithArray("ids",
			mcp.Required(),
			mcp.Items(map[string]any{"type": "number"}),
			mcp.MinItems(1),
			mcp.Description("待删除申请 ID 列表（至少 1 个）"),
		),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		ids, err := requiredInts(args, "ids")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if err := d.Apply.BatchDelete(ids); err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"deleted": len(ids)}), nil
	})
}

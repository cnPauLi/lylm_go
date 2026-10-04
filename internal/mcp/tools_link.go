package mcp

import (
	"context"
	"net/url"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"lylme_spage/internal/models"
	"lylme_spage/internal/service"
)

// link 参数/分页常量，与 REST 层保持一致
const (
	mcpLinkDefaultPageSize = 10
	mcpLinkMaxPageSize     = 100
)

// registerLinkTools 注册链接（link）域工具，与 /api/v1/links 端点一一对应。
func (set *MCPSet) registerLinkTools() {
	d := set.deps

	// link_list：GET /links
	set.add(mcp.NewTool("link_list",
		mcp.WithDescription("分页查询链接。可按分组过滤（group_id）或按关键词检索（name/url/描述/关键词）；关键词非空时忽略分组过滤。"),
		mcp.WithNumber("page", mcp.Description("页码，从 1 开始，默认 1")),
		mcp.WithNumber("page_size", mcp.Description("每页数量，默认 10，最大 100")),
		mcp.WithNumber("group_id", mcp.Description("仅返回该分组的链接，0 或不传表示全部")),
		mcp.WithString("keyword", mcp.Description("关键词，在名称、URL、描述、检索关键词上做子串匹配")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		page := optionalInt(args, "page", 1)
		if page < 1 {
			page = 1
		}
		pageSize := optionalInt(args, "page_size", mcpLinkDefaultPageSize)
		if pageSize < 1 {
			pageSize = mcpLinkDefaultPageSize
		}
		if pageSize > mcpLinkMaxPageSize {
			pageSize = mcpLinkMaxPageSize
		}
		groupID := optionalInt(args, "group_id", 0)
		keyword := strings.TrimSpace(optionalString(args, "keyword", ""))

		var (
			rows  []models.Link
			total int64
			err   error
		)
		if keyword != "" {
			rows, total, err = d.Link.SearchLinks(keyword, page, pageSize)
		} else {
			rows, total, err = d.Link.ListLinks(groupID, page, pageSize)
		}
		if err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{
			"list":      rows,
			"page":      page,
			"page_size": pageSize,
			"total":     total,
		}), nil
	})

	// link_all：GET /links/all
	set.add(mcp.NewTool("link_all",
		mcp.WithDescription("返回全站链接简表（id、url、name，按 id 升序），不分页。"),
	), func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		links, err := d.Link.All()
		if err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"count": len(links), "links": links}), nil
	})

	// link_get：GET /links/:id
	set.add(mcp.NewTool("link_get",
		mcp.WithDescription("按 ID 查询单条链接的完整信息。"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("链接 ID（必填，正整数）")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		id, err := requiredInt(args, "id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		row, err := d.Link.Get(id)
		if err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(row), nil
	})

	// link_create：POST /links
	set.add(mcp.NewTool("link_create",
		mcp.WithDescription("新增一条链接。名称/URL 必填；URL 去掉末尾斜杠后查重，重复时返回错误并给出已存在链接 ID；color 非空时名称会以指定颜色展示。"),
		mcp.WithString("name", mcp.Required(), mcp.Description("链接名称（必填，最长 255 字符）")),
		mcp.WithString("url", mcp.Required(), mcp.Description("链接地址（必填，最长 255 字符）")),
		mcp.WithString("color", mcp.Description("名称颜色（十六进制色值或颜色名，最长 32 字符）")),
		mcp.WithString("icon", mcp.Description("链接图标：URL、图标类名或 SVG（最长 2000 字符）")),
		mcp.WithNumber("group_id", mcp.Description("所属分组 ID，0 为未分组（默认 0）")),
		mcp.WithString("link_desc", mcp.Description("链接描述（最长 255 字符）")),
		mcp.WithString("link_keywords", mcp.Description("检索关键词（最长 512 字符）")),
		mcp.WithNumber("pwd", mcp.Description("访问密码组 ID，0 为无密码（默认 0）")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		name, err := requiredString(args, "name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		rawURL, err := requiredString(args, "url")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		res, err := d.Link.Create(service.LinkInput{
			Name:     name,
			URL:      rawURL,
			Color:    optionalString(args, "color", ""),
			Icon:     optionalString(args, "icon", ""),
			GroupID:  optionalInt(args, "group_id", 0),
			Desc:     optionalString(args, "link_desc", ""),
			Keywords: optionalString(args, "link_keywords", ""),
			Pwd:      optionalInt(args, "pwd", 0),
		})
		if err != nil {
			return serviceErrorResult(err), nil
		}
		if res.Duplicate {
			return mcp.NewToolResultError("链接已存在，已存在链接 ID: " + itoa(res.Link.ID)), nil
		}
		return jsonResult(map[string]any{"id": res.Link.ID}), nil
	})

	// link_update：PUT /links/:id
	set.add(mcp.NewTool("link_update",
		mcp.WithDescription("修改指定链接的全部字段。名称/URL 必填；color 非空时名称以指定颜色展示。"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("链接 ID（必填，正整数）")),
		mcp.WithString("name", mcp.Required(), mcp.Description("链接名称（必填，最长 255 字符）")),
		mcp.WithString("url", mcp.Required(), mcp.Description("链接地址（必填，最长 255 字符）")),
		mcp.WithString("color", mcp.Description("名称颜色（十六进制色值或颜色名，最长 32 字符）")),
		mcp.WithString("icon", mcp.Description("链接图标：URL、图标类名或 SVG（最长 2000 字符）")),
		mcp.WithNumber("group_id", mcp.Description("所属分组 ID，0 为未分组（默认 0）")),
		mcp.WithString("link_desc", mcp.Description("链接描述（最长 255 字符）")),
		mcp.WithString("link_keywords", mcp.Description("检索关键词（最长 512 字符）")),
		mcp.WithNumber("pwd", mcp.Description("访问密码组 ID，0 为无密码（默认 0）")),
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
		rawURL, err := requiredString(args, "url")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		row, err := d.Link.Update(id, service.LinkInput{
			Name:     name,
			URL:      rawURL,
			Color:    optionalString(args, "color", ""),
			Icon:     optionalString(args, "icon", ""),
			GroupID:  optionalInt(args, "group_id", 0),
			Desc:     optionalString(args, "link_desc", ""),
			Keywords: optionalString(args, "link_keywords", ""),
			Pwd:      optionalInt(args, "pwd", 0),
		})
		if err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"id": row.ID, "link": row}), nil
	})

	// link_delete：DELETE /links
	set.add(mcp.NewTool("link_delete",
		mcp.WithDescription("按 ID 列表批量删除链接，此操作不可恢复。"),
		mcp.WithArray("ids",
			mcp.Required(),
			mcp.Items(map[string]any{"type": "number"}),
			mcp.MinItems(1),
			mcp.Description("待删除链接 ID 列表（至少 1 个，最多 500 个）"),
		),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		ids, err := requiredInts(args, "ids")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if err := d.Link.BatchDelete(ids); err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"ids": ids}), nil
	})

	// link_set_status：POST /links/status
	set.add(mcp.NewTool("link_set_status",
		mcp.WithDescription("批量启用或停用链接。"),
		mcp.WithArray("ids",
			mcp.Required(),
			mcp.Items(map[string]any{"type": "number"}),
			mcp.MinItems(1),
			mcp.Description("链接 ID 列表（至少 1 个）"),
		),
		mcp.WithNumber("status", mcp.Required(), mcp.Description("目标状态：1 启用，0 停用")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		ids, err := requiredInts(args, "ids")
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
		if err := d.Link.BatchSetStatus(ids, status); err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"ids": ids, "status": status}), nil
	})

	// link_move_group：POST /links/group
	set.add(mcp.NewTool("link_move_group",
		mcp.WithDescription("把指定链接批量移动到目标分组。"),
		mcp.WithArray("ids",
			mcp.Required(),
			mcp.Items(map[string]any{"type": "number"}),
			mcp.MinItems(1),
			mcp.Description("链接 ID 列表（至少 1 个）"),
		),
		mcp.WithNumber("group_id", mcp.Required(), mcp.Description("目标分组 ID（必填，正整数）")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		ids, err := requiredInts(args, "ids")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		groupID, err := requiredInt(args, "group_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if err := d.Link.BatchMoveGroup(ids, groupID); err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"ids": ids, "group_id": groupID}), nil
	})

	// link_set_pwd：POST /links/pwd
	set.add(mcp.NewTool("link_set_pwd",
		mcp.WithDescription("批量设置或取消链接的访问密码组。"),
		mcp.WithArray("ids",
			mcp.Required(),
			mcp.Items(map[string]any{"type": "number"}),
			mcp.MinItems(1),
			mcp.Description("链接 ID 列表（至少 1 个）"),
		),
		mcp.WithNumber("pwd", mcp.Description("密码组 ID，0 表示取消密码（默认 0）")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		ids, err := requiredInts(args, "ids")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		pwd := optionalInt(args, "pwd", 0)
		if err := d.Link.BatchSetPwd(ids, pwd); err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"ids": ids, "pwd": pwd}), nil
	})

	// link_sort：POST /links/sort
	set.add(mcp.NewTool("link_sort",
		mcp.WithDescription("按传入的链接 ID（字符串形式）顺序重新排列链接展示顺序。"),
		mcp.WithArray("ordered",
			mcp.Required(),
			mcp.Items(map[string]any{"type": "string"}),
			mcp.MinItems(1),
			mcp.Description("链接 ID 字符串数组，数组顺序即新的展示顺序"),
		),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		ordered, err := requiredStrings(args, "ordered")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if len(ordered) == 0 {
			return mcp.NewToolResultError("参数 \"ordered\" 至少包含 1 个链接 ID"), nil
		}
		if err := d.Link.Sort(ordered); err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"ordered": ordered}), nil
	})

	// link_fetch_head：GET /links/fetch
	set.add(mcp.NewTool("link_fetch_head",
		mcp.WithDescription("抓取目标站点的标题（title）与图标（icon）等 head 信息，服务端带 SSRF 防护，仅允许访问公网 HTTP(S) 站点。"),
		mcp.WithString("url", mcp.Required(), mcp.Description("目标站点 URL（必填，http/https 且包含主机名）")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		rawURL := strings.TrimSpace(optionalString(args, "url", ""))
		if rawURL == "" {
			return mcp.NewToolResultError("缺少必填参数 \"url\""), nil
		}
		if !mcpValidHTTPURL(rawURL) {
			return mcp.NewToolResultError("参数 \"url\" 格式非法，仅支持 http/https 且必须包含主机名"), nil
		}
		head, err := d.Link.FetchHead(rawURL)
		if err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(head), nil
	})
}

// mcpValidHTTPURL 校验为 http/https 且包含主机名（与 REST 校验一致）
func mcpValidHTTPURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	return (scheme == "http" || scheme == "https") && u.Hostname() != ""
}

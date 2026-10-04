package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	"lylme_spage/internal/service"
)

// registerPwdTools 注册加密组（pwd）域工具，与 /api/v1/pwd 端点一一对应。
func (set *MCPSet) registerPwdTools() {
	d := set.deps

	// pwd_list：GET /pwd
	set.add(mcp.NewTool("pwd_list",
		mcp.WithDescription("查询全部加密访问密码组，返回列表与总数。"),
	), func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		rows, err := d.Pwd.List()
		if err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"list": rows, "total": len(rows)}), nil
	})

	// pwd_create：POST /pwd
	set.add(mcp.NewTool("pwd_create",
		mcp.WithDescription("新增一个加密访问密码组。名称与访问密码必填。"),
		mcp.WithString("name", mcp.Required(), mcp.Description("加密组名称（必填，最长 30 字符）")),
		mcp.WithString("key", mcp.Required(), mcp.Description("访问密码（链接的访问口令，必填，最长 60 字符）")),
		mcp.WithString("ps", mcp.Description("密码提示语（最长 60 字符）")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		name, err := requiredString(args, "name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		key, err := requiredString(args, "key")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		row, err := d.Pwd.Create(service.PwdInput{
			Name: name,
			Key:  key,
			Ps:   optionalString(args, "ps", ""),
		})
		if err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(row), nil
	})

	// pwd_update：PUT /pwd/:id
	set.add(mcp.NewTool("pwd_update",
		mcp.WithDescription("修改指定加密访问密码组。名称与访问密码必填。"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("加密组 ID（必填，正整数）")),
		mcp.WithString("name", mcp.Required(), mcp.Description("加密组名称（必填，最长 30 字符）")),
		mcp.WithString("key", mcp.Required(), mcp.Description("访问密码（必填，最长 60 字符）")),
		mcp.WithString("ps", mcp.Description("密码提示语（最长 60 字符）")),
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
		key, err := requiredString(args, "key")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		row, err := d.Pwd.Update(id, service.PwdInput{
			Name: name,
			Key:  key,
			Ps:   optionalString(args, "ps", ""),
		})
		if err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(row), nil
	})

	// pwd_delete：DELETE /pwd/:id
	set.add(mcp.NewTool("pwd_delete",
		mcp.WithDescription("删除指定的加密访问密码组。"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("加密组 ID（必填，正整数）")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		id, err := requiredInt(args, "id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if _, err := d.Pwd.Get(id); err != nil {
			return serviceErrorResult(err), nil
		}
		if err := d.Pwd.Delete(id); err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"id": id}), nil
	})
}

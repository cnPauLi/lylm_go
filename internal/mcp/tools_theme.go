package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	"lylme_spage/internal/theme"
)

// mcpThemeInfoItem 主题列表项（结构与 REST themeInfoItem 一致）
type mcpThemeInfoItem struct {
	Dir         string `json:"dir"`
	Name        string `json:"name"`
	Author      string `json:"author"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// mcpThemeFieldDTO 主题配置字段定义（结构与 REST themeFieldDTO 一致）
type mcpThemeFieldDTO struct {
	Type        string            `json:"type"`
	Name        string            `json:"name"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Verify      string            `json:"verify"`
	Placeholder string            `json:"placeholder"`
	Value       any               `json:"value"`
	Enum        []mcpThemeEnumDTO `json:"enum"`
}

type mcpThemeEnumDTO struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

func toMcpFieldDTOs(fields []theme.Field) []mcpThemeFieldDTO {
	items := make([]mcpThemeFieldDTO, 0, len(fields))
	for _, f := range fields {
		enum := make([]mcpThemeEnumDTO, 0, len(f.Enum))
		for _, e := range f.Enum {
			enum = append(enum, mcpThemeEnumDTO{Value: e.Value, Label: e.Label})
		}
		items = append(items, mcpThemeFieldDTO{
			Type:        f.Type,
			Name:        f.Name,
			Title:       f.Title,
			Description: f.Description,
			Verify:      f.Verify,
			Placeholder: f.Placeholder,
			Value:       f.Value,
			Enum:        enum,
		})
	}
	return items
}

// registerThemeTools 注册主题（theme）域工具，与 /api/v1/themes 端点一一对应。
func (set *MCPSet) registerThemeTools() {
	d := set.deps

	// theme_list：GET /themes
	set.add(mcp.NewTool("theme_list",
		mcp.WithDescription("查询全部可用主题（目录名、名称、作者、版本、说明）与当前启用主题目录名。"),
	), func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		themes, current := d.Theme.List()
		items := make([]mcpThemeInfoItem, 0, len(themes))
		for _, ti := range themes {
			items = append(items, mcpThemeInfoItem{
				Dir:         ti.Dir,
				Name:        ti.Name,
				Author:      ti.AuthorName,
				Version:     ti.Version,
				Description: ti.Explain,
			})
		}
		return jsonResult(map[string]any{"current": current, "themes": items}), nil
	})

	// theme_switch：PUT /themes/current
	set.add(mcp.NewTool("theme_switch",
		mcp.WithDescription("切换当前启用的主题（按主题目录名）。"),
		mcp.WithString("name", mcp.Required(), mcp.Description("目标主题目录名（必填，如 default）")),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		name, err := requiredString(args, "name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if err := d.Theme.Switch(name); err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(map[string]any{"current": d.Theme.Current()}), nil
	})

	// theme_config_get：GET /themes/config
	set.add(mcp.NewTool("theme_config_get",
		mcp.WithDescription("读取当前主题的自定义配置字段定义与已保存值（theme_config_<主题>）。"),
	), func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fields, tpl := d.Theme.ConfigFields()
		values := d.Theme.ConfigValues()
		if values == nil {
			values = map[string]any{}
		}
		return jsonResult(map[string]any{
			"theme":  tpl,
			"fields": toMcpFieldDTOs(fields),
			"values": values,
		}), nil
	})

	// theme_config_save：PUT /themes/config
	set.add(mcp.NewTool("theme_config_save",
		mcp.WithDescription("保存主题自定义配置（全量替换语义）；未提交的 switch 补 0、checkbox 补空数组。theme 为空或主题不存在时回退当前主题。"),
		mcp.WithString("theme", mcp.Description("主题目录名（可空，空表示当前主题）")),
		mcp.WithObject("values",
			mcp.Description("配置键值对象（全量替换），键为字段名、值为字段值"),
			mcp.AdditionalProperties(true),
		),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		tpl := optionalString(args, "theme", "")
		values := optionalObject(args, "values", map[string]any{})
		if err := d.Theme.SaveConfig(tpl, values); err != nil {
			return serviceErrorResult(err), nil
		}
		_, effTpl := d.Theme.ConfigFields()
		saved := d.Theme.ConfigValues()
		if saved == nil {
			saved = map[string]any{}
		}
		return jsonResult(map[string]any{"theme": effTpl, "values": saved}), nil
	})
}

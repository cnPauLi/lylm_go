package api

import (
	"errors"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/theme"
)

// RegisterThemeRoutes 在 /api/v1 下注册主题域端点：
//
//	GET  /themes          主题列表与当前主题
//	PUT  /themes/current  切换当前主题
//	GET  /themes/config   当前主题配置字段定义与已保存值
//	PUT  /themes/config   保存主题自定义配置
//
// 鉴权与限流由 /api/v1 组统一挂载；错误统一经 WriteServiceError 映射。
func RegisterThemeRoutes(g gin.IRouter, d *Deps) {
	g.GET("/themes", d.listThemes)
	g.PUT("/themes/current", d.setCurrentTheme)
	g.GET("/themes/config", d.getThemeConfig)
	g.PUT("/themes/config", d.saveThemeConfig)
}

// ===== 出参 DTO =====

// themeInfoItem 主题列表项（对应 theme.Info 的对外裁剪）
type themeInfoItem struct {
	Dir         string `json:"dir"`         // 主题目录名（稳定标识）
	Name        string `json:"name"`        // 主题显示名
	Author      string `json:"author"`      // 作者
	Version     string `json:"version"`     // 版本
	Description string `json:"description"` // 主题说明
}

// themeFieldDTO 主题配置字段定义（对应 theme.Field）
type themeFieldDTO struct {
	Type        string         `json:"type"`        // 表单类型 text/textarea/select/checkbox/radio/color/switch
	Name        string         `json:"name"`        // 配置键（唯一）
	Title       string         `json:"title"`       // 字段标题
	Description string         `json:"description"` // 字段说明（可能含 HTML）
	Verify      string         `json:"verify"`      // 校验方式 required/url/number/email
	Placeholder string         `json:"placeholder"` // 占位提示
	Value       any            `json:"value"`       // 默认值（string 或 []string）
	Enum        []themeEnumDTO `json:"enum"`        // 枚举项（select/radio/checkbox）
}

// themeEnumDTO 枚举项
type themeEnumDTO struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

func toFieldDTOs(fields []theme.Field) []themeFieldDTO {
	items := make([]themeFieldDTO, 0, len(fields))
	for _, f := range fields {
		enum := make([]themeEnumDTO, 0, len(f.Enum))
		for _, e := range f.Enum {
			enum = append(enum, themeEnumDTO{Value: e.Value, Label: e.Label})
		}
		items = append(items, themeFieldDTO{
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

// ===== 处理器 =====

// listThemes GET /themes：返回全部可用主题及当前主题目录名
func (d *Deps) listThemes(c *gin.Context) {
	themes, current := d.Theme.List()
	items := make([]themeInfoItem, 0, len(themes))
	for _, ti := range themes {
		items = append(items, themeInfoItem{
			Dir:         ti.Dir,
			Name:        ti.Name,
			Author:      ti.AuthorName,
			Version:     ti.Version,
			Description: ti.Explain,
		})
	}
	OK(c, gin.H{"current": current, "themes": items})
}

// setCurrentTheme PUT /themes/current：切换当前主题
func (d *Deps) setCurrentTheme(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		FailBadRequest(c, "请求体无效：name 为必填项")
		return
	}
	if err := d.Theme.Switch(req.Name); err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"current": d.Theme.Current()})
}

// getThemeConfig GET /themes/config：返回字段定义与已保存值（theme_config_<tpl>）
func (d *Deps) getThemeConfig(c *gin.Context) {
	fields, tpl := d.Theme.ConfigFields()
	values := d.Theme.ConfigValues()
	if values == nil {
		values = map[string]any{}
	}
	OK(c, gin.H{
		"theme":  tpl,
		"fields": toFieldDTOs(fields),
		"values": values,
	})
}

// saveThemeConfig PUT /themes/config：保存主题自定义配置。
// 支持三种 body 形式：
//
//	{"theme":"default","values":{...}}
//	{"theme":"default","data":{...}}
//	{"theme":"default", 平铺配置键...}
//
// theme 可空（空或指向不存在主题时由 service 回退当前主题）。
func (d *Deps) saveThemeConfig(c *gin.Context) {
	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		FailBadRequest(c, "请求体必须为 JSON 对象")
		return
	}
	tpl, _ := body["theme"].(string)
	values, err := extractThemeValues(body)
	if err != nil {
		FailBadRequest(c, err.Error())
		return
	}
	if err := d.Theme.SaveConfig(tpl, values); err != nil {
		WriteServiceError(c, err)
		return
	}
	_, effTpl := d.Theme.ConfigFields()
	saved := d.Theme.ConfigValues()
	if saved == nil {
		saved = map[string]any{}
	}
	OK(c, gin.H{"theme": effTpl, "values": saved})
}

// extractThemeValues 从请求体提取配置值 map：优先 values，其次 data，
// 最后将 body（去掉 theme）视为平铺配置。
func extractThemeValues(body map[string]any) (map[string]any, error) {
	if raw, ok := body["values"]; ok {
		switch v := raw.(type) {
		case nil:
			return map[string]any{}, nil
		case map[string]any:
			return v, nil
		default:
			return nil, errors.New("values 必须为 JSON 对象")
		}
	}
	if raw, ok := body["data"]; ok {
		if v, ok := raw.(map[string]any); ok {
			return v, nil
		}
		return nil, errors.New("data 必须为 JSON 对象")
	}
	values := make(map[string]any, len(body))
	for k, v := range body {
		if k == "theme" {
			continue
		}
		values[k] = v
	}
	return values, nil
}

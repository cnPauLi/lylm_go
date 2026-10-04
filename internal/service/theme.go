package service

import (
	"encoding/json"
	"strings"

	"lylme_spage/internal/site"
	"lylme_spage/internal/theme"
)

// ThemeService 主题域业务规则：主题列表、当前主题、切换主题、
// 主题自定义配置字段/取值读取与保存。
type ThemeService struct {
	store *site.Store
}

// NewThemeService 创建主题服务
func NewThemeService(store *site.Store) *ThemeService {
	return &ThemeService{store: store}
}

// List 返回全部可用主题与当前启用主题目录名
func (s *ThemeService) List() (themes []theme.Info, current string) {
	return s.store.Theme().List(), s.Current()
}

// Current 返回当前启用主题目录名；配置缺失或指向不存在的主题时回退 default
func (s *ThemeService) Current() string {
	return s.store.Template()
}

// Switch 切换当前主题：name（trim 后）为空 → ErrInvalidInput；
// 主题不存在 → ErrThemeNotFound；成功后写配置键 template。
func (s *ThemeService) Switch(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidInput
	}
	if !s.store.Theme().Exists(name) {
		return ErrThemeNotFound
	}
	return s.store.Set("template", name, "网站主题")
}

// ConfigFields 返回当前主题的配置字段定义与主题目录名
func (s *ThemeService) ConfigFields() (fields []theme.Field, tpl string) {
	tpl = s.Current()
	return s.store.Theme().ConfigFields(tpl, s.store.Conf()), tpl
}

// ConfigValues 返回当前主题已保存的自定义配置（theme_config_<tpl>）；
// 未保存过时返回 nil（len 为 0）
func (s *ThemeService) ConfigValues() map[string]any {
	return theme.ConfigMap(s.store.Conf(), s.Current())
}

// SaveConfig 保存主题自定义配置：
// tpl 为空或主题不存在时回退当前主题；未提交（或提交空字符串）的 checkbox
// 补 []string{}、未提交的 switch 补 0；JSON 序列化后写配置键 theme_config_<tpl>。
func (s *ThemeService) SaveConfig(tpl string, data map[string]any) error {
	tpl = strings.TrimSpace(tpl)
	if tpl == "" || !s.store.Theme().Exists(tpl) {
		tpl = s.Current()
	}
	conf := s.store.Conf()
	// 补回未提交的 checkbox / switch 默认值
	for _, f := range s.store.Theme().ConfigFields(tpl, conf) {
		switch f.Type {
		case "checkbox":
			if v, ok := data[f.Name]; !ok || isEmptyValue(v) {
				data[f.Name] = []string{}
			}
		case "switch":
			if _, ok := data[f.Name]; !ok {
				data[f.Name] = 0
			}
		}
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	desc := s.store.Theme().Info(tpl).Name + "主题自定义设置"
	return s.store.Set("theme_config_"+tpl, string(raw), desc)
}

// isEmptyValue 判断值是否为空标量（空字符串）
func isEmptyValue(v any) bool {
	if s, ok := v.(string); ok {
		return s == ""
	}
	return false
}

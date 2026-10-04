package theme

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Info 主题元信息（theme.ini）
type Info struct {
	Dir         string
	Name        string
	AuthorName  string
	AuthorLink  string
	Version     string
	Explain     string
	Demo        string
	Course      string
	Requires    string
	UpdatedAt   string
	License     string
}

// Manager 主题管理器
type Manager struct {
	root string
}

// NewManager 创建主题管理器，root 为项目根目录
func NewManager(root string) *Manager {
	return &Manager{root: root}
}

// TemplateRoot 主题目录
func (m *Manager) TemplateRoot() string {
	return filepath.Join(m.root, "template")
}

// Dir 主题目录绝对路径
func (m *Manager) Dir(name string) string {
	return filepath.Join(m.TemplateRoot(), filepath.Base(name))
}

// Exists 主题是否存在
func (m *Manager) Exists(name string) bool {
	fi, err := os.Stat(m.Dir(name))
	return err == nil && fi.IsDir()
}

// List 列出所有主题
func (m *Manager) List() []Info {
	entries, err := os.ReadDir(m.TemplateRoot())
	if err != nil {
		return nil
	}
	var list []Info
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		list = append(list, m.Info(e.Name()))
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Dir < list[j].Dir })
	return list
}

// Info 读取指定主题信息
func (m *Manager) Info(name string) Info {
	info := Info{Dir: filepath.Base(name), Name: filepath.Base(name)}
	data, err := os.ReadFile(filepath.Join(m.Dir(name), "theme.ini"))
	if err != nil {
		return info
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return info
	}
	get := func(k string) string {
		if v, ok := raw[k].(string); ok {
			return v
		}
		return ""
	}
	info.AuthorName = get("author_name")
	info.AuthorLink = get("author_link")
	info.Version = get("theme_version")
	info.Explain = get("theme_explain")
	info.Demo = get("theme_demo")
	info.Course = get("theme_course")
	info.Requires = get("requires")
	info.UpdatedAt = get("updated_at")
	info.License = get("license")
	if n := get("theme_name"); n != "" {
		info.Name = n
	}
	return info
}

// ConfigFields 解析主题 config.php 定义
func (m *Manager) ConfigFields(name string, conf map[string]string) []Field {
	path := filepath.Join(m.Dir(name), "config.php")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return ParseConfig(string(data), conf)
}

// ConfigMap 读取数据库中保存的主题配置（theme_config_<name>）
func ConfigMap(conf map[string]string, name string) map[string]any {
	raw, ok := conf["theme_config_"+name]
	if !ok || raw == "" {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil
	}
	return m
}

// ConfigValue 获取主题配置值：数据库优先，其次 config.php 默认值，最后 def
func (m *Manager) ConfigValue(name, key, def string, conf map[string]string) string {
	if db := ConfigMap(conf, name); db != nil {
		if v, ok := db[key]; ok {
			return anyToString(v)
		}
	}
	for _, f := range m.ConfigFields(name, conf) {
		if f.Name == key {
			if s := f.DefaultValue(); s != "" {
				return s
			}
			return def
		}
	}
	return def
}

// ConfigValues 返回全部配置项（已合并数据库值与默认值）
func (m *Manager) ConfigValues(name string, conf map[string]string) map[string]any {
	out := map[string]any{}
	for _, f := range m.ConfigFields(name, conf) {
		out[f.Name] = f.Value
	}
	if db := ConfigMap(conf, name); db != nil {
		for k, v := range db {
			out[k] = v
		}
	}
	return out
}

func anyToString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%v", t)
	case bool:
		if t {
			return "1"
		}
		return "0"
	case []any:
		parts := make([]string, 0, len(t))
		for _, it := range t {
			parts = append(parts, anyToString(it))
		}
		return strings.Join(parts, ",")
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", t)
	}
}

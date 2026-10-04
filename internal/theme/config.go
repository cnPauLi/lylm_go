package theme

import (
	"bytes"
	"encoding/json"
	"strings"
)

// EnumOption 下拉/单选/多选的枚举项
type EnumOption struct {
	Value string
	Label string
}

// Field 主题自定义配置项（对应 theme.ini 中 theme_config 数组的一个元素）
type Field struct {
	Type        string
	Name        string
	Title       string
	Description string
	Verify      string
	Placeholder string
	Value       any // string 或 []string
	Enum        []EnumOption
}

// DefaultValue 返回默认值字符串形式
func (f Field) DefaultValue() string {
	switch v := f.Value.(type) {
	case string:
		return v
	case []string:
		return strings.Join(v, ",")
	default:
		return ""
	}
}

// DefaultValueList 返回多选默认值
func (f Field) DefaultValueList() []string {
	if v, ok := f.Value.([]string); ok {
		return v
	}
	if s, ok := f.Value.(string); ok && s != "" {
		return strings.Split(s, ",")
	}
	return nil
}

// ParseConfig 解析 theme.ini 文本，提取顶层 theme_config 数组定义。
// 解析失败或缺少 theme_config 键时返回 nil。
func ParseConfig(src string, conf map[string]string) []Field {
	dec := json.NewDecoder(bytes.NewReader([]byte(src)))
	dec.UseNumber()
	var doc map[string]json.RawMessage
	if err := dec.Decode(&doc); err != nil {
		return nil
	}
	raw, ok := doc["theme_config"]
	if !ok {
		return nil
	}
	dec = json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var arr []map[string]any
	if err := dec.Decode(&arr); err != nil {
		return nil
	}
	fields := make([]Field, 0, len(arr))
	for _, item := range arr {
		fields = append(fields, docToField(item, conf))
	}
	return fields
}

// docToField 将 theme_config 数组的一个字段对象转换为 Field
func docToField(m map[string]any, conf map[string]string) Field {
	return Field{
		Type:        jsonStr(m["type"]),
		Name:        jsonStr(m["name"]),
		Title:       jsonStr(m["title"]),
		Description: jsonStr(m["description"]),
		Verify:      jsonStr(m["verify"]),
		Placeholder: jsonStr(m["placeholder"]),
		Value:       jsonValue(m["value"], conf),
		Enum:        jsonEnum(m["enum"]),
	}
}

// jsonStr 字符串与数字统一转为字符串，其余类型返回空串
func jsonStr(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	default:
		return ""
	}
}

// jsonValue 解析默认值：字符串、数字（转为字符串）、字符串数组（多选）、
// 或对象 {"$conf":"站点配置键"}（解析期代入 conf 的值，键不存在得空串）
func jsonValue(v any, conf map[string]string) any {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case []any:
		list := make([]string, 0, len(t))
		for _, it := range t {
			if s := jsonStr(it); s != "" {
				list = append(list, s)
			}
		}
		return list
	case map[string]any:
		if key, ok := t["$conf"].(string); ok {
			if conf == nil {
				return ""
			}
			return conf[key]
		}
		return ""
	default:
		return ""
	}
}

// jsonEnum 解析枚举：二维字符串数组 [[value,label],...]，保持声明顺序
func jsonEnum(v any) []EnumOption {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	var opts []EnumOption
	for _, it := range arr {
		pair, ok := it.([]any)
		if !ok || len(pair) != 2 {
			continue
		}
		opts = append(opts, EnumOption{Value: jsonStr(pair[0]), Label: jsonStr(pair[1])})
	}
	return opts
}

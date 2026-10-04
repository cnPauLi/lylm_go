package theme

import (
	"os"
	"path/filepath"
	"testing"
)

func loadConfig(t *testing.T, name string, conf map[string]string) []Field {
	t.Helper()
	path := filepath.Join("..", "..", "template", name, "theme.ini")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 %s 失败: %v", path, err)
	}
	fields := ParseConfig(string(data), conf)
	if fields == nil {
		t.Fatalf("%s 的 theme.ini 未解析出 theme_config 字段", name)
	}
	return fields
}

func findField(fields []Field, name string) *Field {
	for i := range fields {
		if fields[i].Name == name {
			return &fields[i]
		}
	}
	return nil
}

func TestParseDefaultThemeConfig(t *testing.T) {
	conf := map[string]string{"home-title": `<h2 class="title">上网，从这里开始！</h2>`}
	fields := loadConfig(t, "default", conf)
	if len(fields) != 5 {
		t.Fatalf("期望 5 个配置项，实际 %d", len(fields))
	}
	home := findField(fields, "home_title")
	if home == nil {
		t.Fatal("未找到 home_title")
	}
	if home.Type != "textarea" {
		t.Errorf("home_title type 期望 textarea，实际 %s", home.Type)
	}
	if v, _ := home.Value.(string); v != conf["home-title"] {
		t.Errorf("home_title 默认值解析错误: %q", v)
	}

	lytoday := findField(fields, "lytoday")
	if lytoday == nil {
		t.Fatal("未找到 lytoday")
	}
	if v, _ := lytoday.Value.(string); v != "0" {
		t.Errorf("lytoday 默认值期望 0，实际 %q", v)
	}
	if len(lytoday.Enum) != 3 {
		t.Fatalf("lytoday enum 期望 3 项，实际 %d", len(lytoday.Enum))
	}
	if lytoday.Enum[1].Value != "1" || lytoday.Enum[1].Label != "搜索栏下方" {
		t.Errorf("lytoday enum[1] 错误: %+v", lytoday.Enum[1])
	}

	bgPos := findField(fields, "background_position")
	if bgPos == nil {
		t.Fatal("未找到 background_position")
	}
	if v, _ := bgPos.Value.(string); v != "0" {
		t.Errorf("background_position 默认值期望 0，实际 %q", v)
	}
}

func TestParseCyberwaveThemeConfig(t *testing.T) {
	fields := loadConfig(t, "cyberwave", nil)
	if len(fields) != 8 {
		t.Fatalf("cyberwave 期望 8 个配置项，实际 %d", len(fields))
	}
	for _, f := range fields {
		switch f.Name {
		case "color":
			if v, _ := f.Value.(string); v != "#00f0ff" {
				t.Errorf("color 默认值错误: %q", v)
			}
		case "modules":
			list, ok := f.Value.([]string)
			if !ok || len(list) != 1 || list[0] != "clock" {
				t.Errorf("modules 默认值错误: %#v", f.Value)
			}
			if len(f.Enum) != 2 {
				t.Errorf("modules enum 期望 2 项，实际 %d", len(f.Enum))
			}
			if f.Enum[0].Value != "clock" || f.Enum[0].Label != "霓虹时钟" {
				t.Errorf("modules enum[0] 键值错误: %+v", f.Enum[0])
			}
		case "glow":
			if len(f.Enum) != 3 {
				t.Fatalf("glow enum 期望 3 项，实际 %d", len(f.Enum))
			}
			want := []EnumOption{{Value: "low", Label: "弱"}, {Value: "mid", Label: "中"}, {Value: "high", Label: "强"}}
			for i, opt := range f.Enum {
				if opt != want[i] {
					t.Errorf("glow enum[%d] 顺序/键值错误: got %+v want %+v", i, opt, want[i])
				}
			}
		}
	}
}

func TestParseDevThemeConfig(t *testing.T) {
	fields := loadConfig(t, "dev-theme", nil)
	if len(fields) != 5 {
		t.Fatalf("dev-theme 期望 5 个配置项，实际 %d", len(fields))
	}
	for _, f := range fields {
		if f.Name == "link_cols" {
			if v, _ := f.Value.(string); v != "4" {
				t.Errorf("link_cols 默认值错误: %q", v)
			}
			if len(f.Enum) != 4 {
				t.Errorf("link_cols enum 期望 4 项，实际 %d", len(f.Enum))
			}
		}
	}
}

// TestAllMigratedThemes 校验迁移后的全部主题 theme.ini 均可解析出字段
func TestAllMigratedThemes(t *testing.T) {
	cases := map[string]int{
		"5iux":        4,
		"baisu":       6,
		"cyberwave":   8,
		"dashlite":    3,
		"default":     5,
		"dev-theme":   5,
		"liquidglass": 12,
		"ltab":        4,
		"lylme":       2,
		"page":        1,
		"quality6.0":  2,
	}
	conf := map[string]string{"home-title": "提示内容"}
	for name, want := range cases {
		fields := loadConfig(t, name, conf)
		if len(fields) != want {
			t.Errorf("%s 期望 %d 个配置项，实际 %d", name, want, len(fields))
		}
	}
}

// TestParseConfigInvalid 校验非法输入返回 nil 且不 panic
func TestParseConfigInvalid(t *testing.T) {
	cases := []string{
		"",
		"not json",
		`{"author_name": "x"}`,
		`{"theme_config": "str"}`,
		`{"theme_config": [1, true, null]}`,
	}
	for _, src := range cases {
		fields := ParseConfig(src, nil)
		if len(fields) != 0 {
			t.Errorf("非法输入 %q 应返回空/nil，实际 %d 项", src, len(fields))
		}
	}
}

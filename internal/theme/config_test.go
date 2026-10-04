package theme

import (
	"os"
	"testing"
)

func loadConfig(t *testing.T, path string, conf map[string]string) []Field {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 %s 失败: %v", path, err)
	}
	return ParseConfig(string(data), conf)
}

func TestParseDefaultThemeConfig(t *testing.T) {
	conf := map[string]string{"home-title": `<h2 class="title">上网，从这里开始！</h2>`}
	fields := loadConfig(t, "../../template/default/config.php", conf)
	if len(fields) != 5 {
		t.Fatalf("期望 5 个配置项，实际 %d", len(fields))
	}
	var home *Field
	for i := range fields {
		if fields[i].Name == "home_title" {
			home = &fields[i]
		}
	}
	if home == nil {
		t.Fatal("未找到 home_title")
	}
	if home.Type != "textarea" {
		t.Errorf("home_title type 期望 textarea，实际 %s", home.Type)
	}
	if v, _ := home.Value.(string); v != conf["home-title"] {
		t.Errorf("home_title 默认值解析错误: %q", v)
	}

	for _, f := range fields {
		if f.Name == "lytoday" {
			if len(f.Enum) != 3 {
				t.Errorf("lytoday enum 期望 3 项，实际 %d", len(f.Enum))
			}
			if f.Enum[1].Value != "1" || f.Enum[1].Label != "搜索栏下方" {
				t.Errorf("lytoday enum[1] 错误: %+v", f.Enum[1])
			}
		}
	}
}

func TestParseArraySyntaxThemeConfig(t *testing.T) {
	fields := loadConfig(t, "../../template/cyberwave/config.php", nil)
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
		}
	}
}

func TestParseDevThemeConfig(t *testing.T) {
	fields := loadConfig(t, "../../template/dev-theme/config.php", nil)
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

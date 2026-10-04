package service

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"lylme_spage/internal/models"
	"lylme_spage/internal/site"
)

// newThemeStore 构建内存 sqlite + 真实项目 template 目录的 store。
// 测试运行目录为 internal/service，回退两级（filepath.Join("..","..")）
// 得到项目根，使 theme.Manager 能发现真实主题（含 default）。
func newThemeStore(t *testing.T) *site.Store {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(models.AllModels()...); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	root := filepath.Join("..", "..")
	store, err := site.NewStore(db, root)
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}
	return store
}

func TestThemeListAndCurrent(t *testing.T) {
	store := newThemeStore(t)
	svc := NewThemeService(store)

	themes, current := svc.List()
	if len(themes) == 0 {
		t.Fatal("应能发现主题，实际为 0")
	}
	found := false
	for _, info := range themes {
		if info.Dir == "default" {
			found = true
		}
	}
	if !found {
		t.Fatal("主题列表应包含 default")
	}
	if current != "default" {
		t.Fatalf("无配置时当前主题应 default，实际 %q", current)
	}
	if svc.Current() != "default" {
		t.Fatalf("Current 应 default，实际 %q", svc.Current())
	}

	// 配置指向不存在的主题时回退 default（Store.Template 既有语义）
	if err := store.Set("template", "no-such-theme", ""); err != nil {
		t.Fatalf("写入非法 template 失败: %v", err)
	}
	if got := svc.Current(); got != "default" {
		t.Fatalf("不存在主题应回退 default，实际 %q", got)
	}
	themes, current = svc.List()
	if current != "default" {
		t.Fatalf("List 当前主题应回退 default，实际 %q", current)
	}
}

func TestThemeSwitch(t *testing.T) {
	store := newThemeStore(t)
	svc := NewThemeService(store)

	if err := svc.Switch("default"); err != nil {
		t.Fatalf("切换到 default 失败: %v", err)
	}
	if svc.Current() != "default" {
		t.Fatalf("切换后 Current 应 default，实际 %q", svc.Current())
	}

	err := svc.Switch("not-exist-theme")
	if !errors.Is(err, ErrThemeNotFound) {
		t.Fatalf("切换不存在主题应 ErrThemeNotFound，实际: %v", err)
	}
	for _, name := range []string{"", "  "} {
		if err := svc.Switch(name); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("切换空主题名 %q 应 ErrInvalidInput，实际: %v", name, err)
		}
	}

	// 失败时 template 键值不应被改写（仍为成功切换时写入的 default）
	if got := store.Get("template", ""); got != "default" {
		t.Fatalf("切换失败后 template 不应被改写，实际 %q", got)
	}
}

func TestThemeConfigFields(t *testing.T) {
	store := newThemeStore(t)
	svc := NewThemeService(store)

	fields, name := svc.ConfigFields()
	if name != "default" {
		t.Fatalf("字段定义所属主题应 default，实际 %q", name)
	}
	if len(fields) == 0 {
		t.Fatal("default 主题应存在配置字段定义")
	}
	found := false
	for _, f := range fields {
		if f.Name == "home_title" {
			found = true
		}
	}
	if !found {
		t.Fatal("default 字段中应包含 home_title")
	}
}

func TestThemeConfigValuesEmpty(t *testing.T) {
	store := newThemeStore(t)
	svc := NewThemeService(store)

	vals := svc.ConfigValues()
	if len(vals) != 0 {
		t.Fatalf("无配置时应返回空 map，实际 %v", vals)
	}
}

func TestThemeSaveConfig(t *testing.T) {
	store := newThemeStore(t)
	svc := NewThemeService(store)

	in := map[string]any{
		"home_title": "你好首页",
		"lytoday":    "2",
		"gonganbei":  "",
	}
	if err := svc.SaveConfig("", in); err != nil {
		t.Fatalf("SaveConfig 失败: %v", err)
	}
	raw := store.Get("theme_config_default", "")
	if raw == "" {
		t.Fatal("应写入配置键 theme_config_default")
	}
	var saved map[string]any
	if err := json.Unmarshal([]byte(raw), &saved); err != nil {
		t.Fatalf("保存内容应为合法 JSON: %v", err)
	}
	if saved["home_title"] != "你好首页" {
		t.Fatalf("home_title 保存异常: %#v", saved["home_title"])
	}
	if saved["lytoday"] != "2" {
		t.Fatalf("lytoday 保存异常: %#v", saved["lytoday"])
	}

	// 再次读取可解析
	vals := svc.ConfigValues()
	if len(vals) == 0 {
		t.Fatal("保存后 ConfigValues 应非空")
	}
	if vals["home_title"] != "你好首页" {
		t.Fatalf("ConfigValues 回读异常: %#v", vals["home_title"])
	}

	// tpl 不存在时回退当前主题，且不写不存在主题的键
	if err := svc.SaveConfig("not-exist-theme", map[string]any{"home_title": "回退保存"}); err != nil {
		t.Fatalf("回退保存失败: %v", err)
	}
	var saved2 map[string]any
	if err := json.Unmarshal([]byte(store.Get("theme_config_default", "")), &saved2); err != nil {
		t.Fatalf("回退保存 JSON 异常: %v", err)
	}
	if saved2["home_title"] != "回退保存" {
		t.Fatalf("应回退写 default，实际 %#v", saved2["home_title"])
	}
	if _, ok := store.Conf()["theme_config_not-exist-theme"]; ok {
		t.Fatal("不应为不存在主题写配置键")
	}
}

// TestThemeSaveConfigDefaults 使用临时 root 构造含 checkbox/switch 字段的
// default 主题（真实 default 无此类字段），验证未提交默认值补齐逻辑。
func TestThemeSaveConfigDefaults(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "template", "default")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("创建临时主题目录失败: %v", err)
	}
	configPHP := `<?php
$theme_config = [
  ['type'=>'checkbox','name'=>'opts','title'=>'多选项','enum'=>['a'=>'选项A','b'=>'选项B']],
  ['type'=>'switch','name'=>'enabled','title'=>'开关'],
  ['type'=>'text','name'=>'title','title'=>'标题','value'=>'默认标题'],
];
`
	if err := os.WriteFile(filepath.Join(dir, "config.php"), []byte(configPHP), 0o644); err != nil {
		t.Fatalf("写入 config.php 失败: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(models.AllModels()...); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	store, err := site.NewStore(db, root)
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}
	svc := NewThemeService(store)

	themes, current := svc.List()
	if len(themes) != 1 || themes[0].Dir != "default" || current != "default" {
		t.Fatalf("临时主题发现异常: %+v / %q", themes, current)
	}

	// 未提交 checkbox/switch：补 [] / 0
	if err := svc.SaveConfig("", map[string]any{"title": "新标题"}); err != nil {
		t.Fatalf("SaveConfig 失败: %v", err)
	}
	saved := readThemeConfig(t, store)
	opts, ok := saved["opts"].([]any)
	if !ok || len(opts) != 0 {
		t.Fatalf("未提交 checkbox 应补空数组，实际 %#v", saved["opts"])
	}
	if saved["enabled"] != float64(0) {
		t.Fatalf("未提交 switch 应补 0，实际 %#v", saved["enabled"])
	}
	if saved["title"] != "新标题" {
		t.Fatalf("title 保存异常: %#v", saved["title"])
	}

	// checkbox 提交空字符串同样补空数组；已提交的非空值与 switch 值保留
	if err := svc.SaveConfig("", map[string]any{
		"opts":    "",
		"enabled": "1",
	}); err != nil {
		t.Fatalf("第二次 SaveConfig 失败: %v", err)
	}
	saved = readThemeConfig(t, store)
	if arr, ok := saved["opts"].([]any); !ok || len(arr) != 0 {
		t.Fatalf("空字符串 checkbox 应补空数组，实际 %#v", saved["opts"])
	}
	if saved["enabled"] != "1" {
		t.Fatalf("已提交 switch 应保留原值，实际 %#v", saved["enabled"])
	}

	if err := svc.SaveConfig("", map[string]any{
		"opts": []string{"a", "b"},
	}); err != nil {
		t.Fatalf("第三次 SaveConfig 失败: %v", err)
	}
	saved = readThemeConfig(t, store)
	arr, ok := saved["opts"].([]any)
	if !ok || len(arr) != 2 || arr[0] != "a" || arr[1] != "b" {
		t.Fatalf("已选 checkbox 应原样保留，实际 %#v", saved["opts"])
	}
	if saved["enabled"] != float64(0) {
		t.Fatalf("再次未提交 switch 仍应补 0，实际 %#v", saved["enabled"])
	}
}

func readThemeConfig(t *testing.T, store *site.Store) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(store.Get("theme_config_default", "")), &m); err != nil {
		t.Fatalf("读取 theme_config_default JSON 失败: %v", err)
	}
	return m
}

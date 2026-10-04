package service

import (
	"errors"
	"testing"

	"gorm.io/gorm"

	"lylme_spage/internal/models"
)

// getConfigRaw 直接查表读取配置原始值；exists 表示该键是否存在
func getConfigRaw(t *testing.T, db *gorm.DB, key string) (val string, exists bool) {
	t.Helper()
	var c models.Config
	err := db.Where("k = ?", key).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false
	}
	if err != nil {
		t.Fatalf("读取配置 %s 失败: %v", key, err)
	}
	return c.V, true
}

func sampleSettings(yan string) SiteSettingsInput {
	return SiteSettingsInput{
		Title:         "测试站名",
		Logo:          "./logo.png",
		Background:    "./bg.jpg",
		WapBackground: "./wapbg.jpg",
		Keywords:      "导航,搜索",
		Description:   "这是一个测试站",
		Mode:          "2",
		Snapshot:      "https://api.example.com/snap?url=",
		Copyright:     "© 2026 测试",
		ICP:           "京ICP备00000000号",
		WzTJ:          "<script>tj</script>",
		CDNPublic:     "https://cdn.example.com/",
		Yan:           yan,
	}
}

func TestConfigUpdateSiteSettings(t *testing.T) {
	store, _ := newLinkStore(t)
	svc := NewConfigService(store)

	in := sampleSettings("true")
	if err := svc.UpdateSiteSettings(in); err != nil {
		t.Fatalf("UpdateSiteSettings 失败: %v", err)
	}

	want := map[string]string{
		"title":          in.Title,
		"logo":           in.Logo,
		"background":     in.Background,
		"wap_background": in.WapBackground,
		"keywords":       in.Keywords,
		"description":    in.Description,
		"mode":           in.Mode,
		"snapshot":       in.Snapshot,
		"copyright":      in.Copyright,
		"icp":            in.ICP,
		"wztj":           in.WzTJ,
		"cdnpublic":      in.CDNPublic,
		"yan":            "true",
		"tq":             "false",
	}
	for key, v := range want {
		got, ok := getConfigRaw(t, store.DB(), key)
		if !ok {
			t.Fatalf("配置 %s 未落库", key)
		}
		if got != v {
			t.Fatalf("配置 %s = %q，期望 %q", key, got, v)
		}
	}

	// yan 仅恰好为 "true" 才落 true，其余取值（含空串/1/yes/false）一律 false
	for _, yan := range []string{"", "1", "yes", "TRUE", "false", " true "} {
		in2 := sampleSettings(yan)
		if err := svc.UpdateSiteSettings(in2); err != nil {
			t.Fatalf("UpdateSiteSettings(yan=%q) 失败: %v", yan, err)
		}
		got, _ := getConfigRaw(t, store.DB(), "yan")
		if got != "false" {
			t.Fatalf("yan=%q 应落 false，实际 %q", yan, got)
		}
		// tq 始终固定 false
		gotTQ, _ := getConfigRaw(t, store.DB(), "tq")
		if gotTQ != "false" {
			t.Fatalf("tq 应固定 false，实际 %q", gotTQ)
		}
	}
}

func TestConfigAbout(t *testing.T) {
	store, _ := newLinkStore(t)
	svc := NewConfigService(store)

	// 未配置时返回默认内容
	if got := svc.AboutContent(); got != DefaultAboutContent {
		t.Fatalf("未配置时应返回默认内容，实际 %q", got)
	}

	custom := "<p>这是自定义关于页</p>"
	if err := svc.UpdateAboutContent(custom); err != nil {
		t.Fatalf("UpdateAboutContent 失败: %v", err)
	}
	if got, ok := getConfigRaw(t, store.DB(), "about_content"); !ok || got != custom {
		t.Fatalf("关于页内容异常: %q (exists=%v)", got, ok)
	}
	if got := svc.AboutContent(); got != custom {
		t.Fatalf("AboutContent 应返回自定义内容，实际 %q", got)
	}

	if err := svc.ResetAboutContent(); err != nil {
		t.Fatalf("ResetAboutContent 失败: %v", err)
	}
	if got := svc.AboutContent(); got != DefaultAboutContent {
		t.Fatalf("重置后应返回默认内容，实际 %q", got)
	}
}

func TestConfigUpdateAccountErrors(t *testing.T) {
	// 旧密码错误：拒绝，且用户名/密码均不变
	t.Run("旧密码错误", func(t *testing.T) {
		store, _ := newLinkStore(t)
		if err := store.Set("admin_user", "admin", ""); err != nil {
			t.Fatal(err)
		}
		oldHash := AdminPasswordHash("oldpass1")
		if err := store.Set("admin_pwd", oldHash, ""); err != nil {
			t.Fatal(err)
		}
		svc := NewConfigService(store)

		_, err := svc.UpdateAccount(AccountInput{Username: "hacker", OldPassword: "wrong", NewPassword: "n1", ConfirmPassword: "n1"})
		if !errors.Is(err, ErrWrongPassword) {
			t.Fatalf("应返回 ErrWrongPassword，实际: %v", err)
		}
		if errors.Is(err, ErrInvalidInput) {
			t.Fatalf("ErrWrongPassword 不应归属 ErrInvalidInput")
		}
		if v, _ := getConfigRaw(t, store.DB(), "admin_user"); v != "admin" {
			t.Fatalf("旧密码错误时用户名不应改变，实际 %q", v)
		}
		if v, _ := getConfigRaw(t, store.DB(), "admin_pwd"); v != oldHash {
			t.Fatalf("旧密码错误时密码不应改变")
		}
	})

	// 两次新密码不一致：拒绝（空用户名与非空用户名两种入口）
	t.Run("两次不一致", func(t *testing.T) {
		for _, user := range []string{"", "newuser"} {
			store, _ := newLinkStore(t)
			if err := store.Set("admin_user", "admin", ""); err != nil {
				t.Fatal(err)
			}
			if err := store.Set("admin_pwd", AdminPasswordHash("oldpass1"), ""); err != nil {
				t.Fatal(err)
			}
			svc := NewConfigService(store)

			_, err := svc.UpdateAccount(AccountInput{Username: user, OldPassword: "oldpass1", NewPassword: "a", ConfirmPassword: "b"})
			if !errors.Is(err, ErrPasswordMismatch) {
				t.Fatalf("user=%q 应返回 ErrPasswordMismatch，实际: %v", user, err)
			}
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("ErrPasswordMismatch 应归属 ErrInvalidInput")
			}
			if v, _ := getConfigRaw(t, store.DB(), "admin_pwd"); v != AdminPasswordHash("oldpass1") {
				t.Fatalf("不一致时密码不应改变")
			}
		}
	})

	// 未做出更改：用户名与新密码均为空
	t.Run("空更改", func(t *testing.T) {
		store, _ := newLinkStore(t)
		if err := store.Set("admin_user", "admin", ""); err != nil {
			t.Fatal(err)
		}
		if err := store.Set("admin_pwd", AdminPasswordHash("oldpass1"), ""); err != nil {
			t.Fatal(err)
		}
		svc := NewConfigService(store)

		_, err := svc.UpdateAccount(AccountInput{OldPassword: "oldpass1"})
		if !errors.Is(err, ErrNoChange) {
			t.Fatalf("应返回 ErrNoChange，实际: %v", err)
		}
	})
}

func TestConfigUpdateAccountSuccess(t *testing.T) {
	// 仅改名
	t.Run("仅改名", func(t *testing.T) {
		store, _ := newLinkStore(t)
		if err := store.Set("admin_user", "admin", ""); err != nil {
			t.Fatal(err)
		}
		oldHash := AdminPasswordHash("oldpass1")
		if err := store.Set("admin_pwd", oldHash, ""); err != nil {
			t.Fatal(err)
		}
		svc := NewConfigService(store)

		res, err := svc.UpdateAccount(AccountInput{Username: "  newadmin  ", OldPassword: "oldpass1"})
		if err != nil {
			t.Fatalf("改名失败: %v", err)
		}
		if !res.ChangedUser || res.ChangedPwd {
			t.Fatalf("应仅 ChangedUser，实际 %+v", res)
		}
		if res.Username != "newadmin" {
			t.Fatalf("结果用户名应为 trim 后的 newadmin，实际 %q", res.Username)
		}
		if v, _ := getConfigRaw(t, store.DB(), "admin_user"); v != "newadmin" {
			t.Fatalf("admin_user 应为 newadmin，实际 %q", v)
		}
		if v, _ := getConfigRaw(t, store.DB(), "admin_pwd"); v != oldHash {
			t.Fatalf("仅改名时密码哈希不应变化")
		}
	})

	// 仅改密
	t.Run("仅改密", func(t *testing.T) {
		store, _ := newLinkStore(t)
		if err := store.Set("admin_user", "admin", ""); err != nil {
			t.Fatal(err)
		}
		if err := store.Set("admin_pwd", AdminPasswordHash("oldpass1"), ""); err != nil {
			t.Fatal(err)
		}
		svc := NewConfigService(store)

		res, err := svc.UpdateAccount(AccountInput{OldPassword: "oldpass1", NewPassword: "newpass2", ConfirmPassword: "newpass2"})
		if err != nil {
			t.Fatalf("改密失败: %v", err)
		}
		if !res.ChangedPwd || res.ChangedUser {
			t.Fatalf("应仅 ChangedPwd，实际 %+v", res)
		}
		if res.Password != "newpass2" {
			t.Fatalf("结果 Password 应为 newpass2，实际 %q", res.Password)
		}
		wantHash := AdminPasswordHash("newpass2")
		if v, _ := getConfigRaw(t, store.DB(), "admin_pwd"); v != wantHash {
			t.Fatalf("admin_pwd 哈希不正确")
		}
		if v, _ := getConfigRaw(t, store.DB(), "admin_user"); v != "admin" {
			t.Fatalf("仅改密时用户名不应变化，实际 %q", v)
		}
	})

	// 用户名与密码都改
	t.Run("两者都改", func(t *testing.T) {
		store, _ := newLinkStore(t)
		if err := store.Set("admin_user", "admin", ""); err != nil {
			t.Fatal(err)
		}
		if err := store.Set("admin_pwd", AdminPasswordHash("oldpass1"), ""); err != nil {
			t.Fatal(err)
		}
		svc := NewConfigService(store)

		res, err := svc.UpdateAccount(AccountInput{Username: "boss", OldPassword: "oldpass1", NewPassword: "newpass3", ConfirmPassword: "newpass3"})
		if err != nil {
			t.Fatalf("修改失败: %v", err)
		}
		if !res.ChangedUser || !res.ChangedPwd {
			t.Fatalf("应 ChangedUser 与 ChangedPwd 均为 true，实际 %+v", res)
		}
		if res.Username != "boss" || res.Password != "newpass3" {
			t.Fatalf("结果回显异常: %+v", res)
		}
		if v, _ := getConfigRaw(t, store.DB(), "admin_user"); v != "boss" {
			t.Fatalf("admin_user 应为 boss，实际 %q", v)
		}
		if v, _ := getConfigRaw(t, store.DB(), "admin_pwd"); v != AdminPasswordHash("newpass3") {
			t.Fatalf("admin_pwd 哈希不正确")
		}
	})
}

func TestConfigAllConfig(t *testing.T) {
	store, _ := newLinkStore(t)
	svc := NewConfigService(store)

	// 空库时返回空 map（非 nil）
	got := svc.AllConfig()
	if got == nil {
		t.Fatal("空库时 AllConfig 应返回非 nil map")
	}
	if len(got) != 0 {
		t.Fatalf("空库时 AllConfig 应为空，实际 %d 项", len(got))
	}

	// 写入含空串的配置后，全部键值（含空值）均可读出
	if err := store.Set("title", "测试站", ""); err != nil {
		t.Fatal(err)
	}
	if err := store.Set("icp", "", ""); err != nil {
		t.Fatal(err)
	}
	got = svc.AllConfig()
	if v, ok := got["title"]; !ok || v != "测试站" {
		t.Fatalf("AllConfig[title] = %q (ok=%v)", v, ok)
	}
	if v, ok := got["icp"]; !ok || v != "" {
		t.Fatalf("空值配置应原样返回，AllConfig[icp] = %q (ok=%v)", v, ok)
	}

	// 修改返回 map 不影响 service 后续读取（副本语义）
	got["title"] = "篡改"
	if v := svc.AllConfig()["title"]; v != "测试站" {
		t.Fatalf("返回 map 被外部修改后污染了内部数据：%q", v)
	}
}

func TestConfigAdminPasswordHash(t *testing.T) {
	// md5("lylme"+明文) 32 位小写 hex，空密码同样可哈希
	if got := AdminPasswordHash(""); got != "0799adf31e720fdd940953c12d4fe220" {
		// md5("lylme")
		t.Fatalf("空明文哈希异常: %s", got)
	}
	for _, p := range []string{"a", "admin", "包含中文密码#1"} {
		got := AdminPasswordHash(p)
		if len(got) != 32 {
			t.Fatalf("哈希长度应为 32，实际 %d", len(got))
		}
		if AdminPasswordHash(p) != got {
			t.Fatalf("同一明文哈希应稳定")
		}
		if AdminPasswordHash(p+" ") == got {
			t.Fatalf("不同明文哈希不应相同")
		}
	}
}

func TestConfigUpdateSystem(t *testing.T) {
	// admin_path 非法值
	for _, p := range []string{"", "a b", "a/b", "a.b", "../etc", "中文目录", "a:b"} {
		store, _ := newLinkStore(t)
		svc := NewConfigService(store)
		err := svc.UpdateSystem(SystemInput{AdminPath: "  " + p + "  ", Debug: "1"})
		// 注意：trim 后 "  " + "" + "  " 仍为空
		if !errors.Is(err, ErrAdminPathInvalid) {
			t.Fatalf("admin_path=%q 应返回 ErrAdminPathInvalid，实际: %v", p, err)
		}
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("ErrAdminPathInvalid 应归属 ErrInvalidInput")
		}
		if _, ok := getConfigRaw(t, store.DB(), "admin_path"); ok {
			t.Fatalf("admin_path=%q 校验失败时不应落库", p)
		}
	}

	// debug 非法值：拒绝且 admin_path 也不落库（两项均校验通过后才写）
	for _, d := range []string{"", "2", "true", "yes", "01"} {
		store, _ := newLinkStore(t)
		svc := NewConfigService(store)
		err := svc.UpdateSystem(SystemInput{AdminPath: "admin", Debug: d})
		if !errors.Is(err, ErrDebugInvalid) {
			t.Fatalf("debug=%q 应返回 ErrDebugInvalid，实际: %v", d, err)
		}
		if _, ok := getConfigRaw(t, store.DB(), "admin_path"); ok {
			t.Fatalf("debug=%q 非法时 admin_path 不应落库", d)
		}
		if _, ok := getConfigRaw(t, store.DB(), "debug"); ok {
			t.Fatalf("debug=%q 不应落库", d)
		}
	}

	// 合法组合落库
	cases := []SystemInput{
		{AdminPath: "admin", Debug: "1"},
		{AdminPath: "ad-min_test1", Debug: "0"},
	}
	for _, in := range cases {
		store, _ := newLinkStore(t)
		svc := NewConfigService(store)
		if err := svc.UpdateSystem(in); err != nil {
			t.Fatalf("合法入参 %+v 失败: %v", in, err)
		}
		if v, _ := getConfigRaw(t, store.DB(), "admin_path"); v != in.AdminPath {
			t.Fatalf("admin_path = %q，期望 %q", v, in.AdminPath)
		}
		if v, _ := getConfigRaw(t, store.DB(), "debug"); v != in.Debug {
			t.Fatalf("debug = %q，期望 %q", v, in.Debug)
		}
	}
}

package service

import (
	"regexp"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"lylme_spage/internal/models"
	"lylme_spage/internal/site"
)

var keyRe = regexp.MustCompile(`^sk-[0-9a-f]{48}$`)

func newKeyStore(t *testing.T) *site.Store {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Config{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	store, err := site.NewStore(db, t.TempDir())
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}
	return store
}

func TestGenerateKeyFormat(t *testing.T) {
	k1, err := GenerateKey()
	if err != nil {
		t.Fatalf("生成密钥失败: %v", err)
	}
	k2, err := GenerateKey()
	if err != nil {
		t.Fatalf("第二次生成失败: %v", err)
	}
	if !keyRe.MatchString(k1) || !keyRe.MatchString(k2) {
		t.Fatalf("密钥格式不符合 sk-+48hex: %s / %s", k1, k2)
	}
	if k1 == k2 {
		t.Fatal("两次生成的密钥相同，随机性不足")
	}
}

func TestEnsureIdempotent(t *testing.T) {
	svc := NewAPIKeyService(newKeyStore(t))
	k1, err := svc.Ensure()
	if err != nil {
		t.Fatalf("Ensure 失败: %v", err)
	}
	k2, err := svc.Ensure()
	if err != nil {
		t.Fatalf("第二次 Ensure 失败: %v", err)
	}
	if k1 != k2 {
		t.Fatalf("Ensure 不幂等: %s != %s", k1, k2)
	}
	if info := svc.Get(); !info.Exists || info.Key != k1 {
		t.Fatalf("Get 返回异常: %+v", info)
	}
}

func TestValidateBeforeAndAfterSet(t *testing.T) {
	svc := NewAPIKeyService(newKeyStore(t))
	// 未设置密钥时任何输入均拒绝
	if svc.Validate("sk-anything") || svc.Validate("") {
		t.Fatal("未设置密钥时 Validate 应一律 false")
	}
	key, err := svc.Ensure()
	if err != nil {
		t.Fatalf("Ensure 失败: %v", err)
	}
	if !svc.Validate(key) {
		t.Fatal("正确密钥 Validate 应为 true")
	}
	// 构造末位必然不同的近似 key（直接拼 "0" 在 key 末位恰为 0 时会与真 key 相同）
	last := byte('0')
	if key[len(key)-1] == '0' {
		last = '1'
	}
	mutated := key[:len(key)-1] + string(last)
	for _, bad := range []string{"", "sk-x", mutated, "Bearer " + key} {
		if svc.Validate(bad) {
			t.Fatalf("错误密钥不应通过: %q", bad)
		}
	}
}

func TestRegenerateInvalidatesOldKey(t *testing.T) {
	svc := NewAPIKeyService(newKeyStore(t))
	old, err := svc.Ensure()
	if err != nil {
		t.Fatalf("Ensure 失败: %v", err)
	}
	newKey, err := svc.Regenerate()
	if err != nil {
		t.Fatalf("Regenerate 失败: %v", err)
	}
	if newKey == old {
		t.Fatal("重新生成后密钥未变化")
	}
	if svc.Validate(old) {
		t.Fatal("旧密钥重新生成后必须立即失效")
	}
	if !svc.Validate(newKey) {
		t.Fatal("新密钥必须有效")
	}
	info := svc.Get()
	if !info.Exists || info.CreatedAt.IsZero() {
		t.Fatalf("重新生成后 Get 信息异常: %+v", info)
	}
	if time.Since(info.CreatedAt) > time.Minute {
		t.Fatalf("创建时间不合理: %v", info.CreatedAt)
	}
}

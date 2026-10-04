package service

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"lylme_spage/internal/models"
	"lylme_spage/internal/site"
)

func newDashboardStore(t *testing.T) *site.Store {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Config{}, &models.Group{}, &models.Link{}, &models.Apply{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	store, err := site.NewStore(db, t.TempDir())
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}
	return store
}

func TestDashboardSummaryCounts(t *testing.T) {
	store := newDashboardStore(t)
	db := store.DB()

	for i := 0; i < 2; i++ {
		if err := db.Create(&models.Group{GroupName: "分组"}).Error; err != nil {
			t.Fatalf("创建分组失败: %v", err)
		}
	}
	for i := 0; i < 3; i++ {
		if err := db.Create(&models.Link{Name: "链接", LinkStatus: 1}).Error; err != nil {
			t.Fatalf("创建链接失败: %v", err)
		}
	}
	// 2 条待审 + 1 条已通过 + 1 条已拒绝
	for _, st := range []int{0, 0, 1, 2} {
		if err := db.Create(&models.Apply{ApplyName: "申请", ApplyStatus: st}).Error; err != nil {
			t.Fatalf("创建申请失败: %v", err)
		}
	}
	if err := store.Set("version", "20240101", ""); err != nil {
		t.Fatalf("写 version 失败: %v", err)
	}
	if err := store.Set("build", "2024-01-01", ""); err != nil {
		t.Fatalf("写 build 失败: %v", err)
	}

	sum := NewDashboardService(store).Summary("2.7.0")
	if sum.Links != 3 {
		t.Fatalf("Links 应为 3，实际 %d", sum.Links)
	}
	if sum.Groups != 2 {
		t.Fatalf("Groups 应为 2，实际 %d", sum.Groups)
	}
	if sum.ApplyRows != 2 {
		t.Fatalf("ApplyRows 应为 2，实际 %d", sum.ApplyRows)
	}
	if sum.AppVersion != "2.7.0" {
		t.Fatalf("AppVersion 异常: %q", sum.AppVersion)
	}
	if sum.DBVersion != "20240101" {
		t.Fatalf("DBVersion 异常: %q", sum.DBVersion)
	}
	if sum.Build != "2024-01-01" {
		t.Fatalf("Build 异常: %q", sum.Build)
	}
}

func TestDashboardSummaryEmpty(t *testing.T) {
	store := newDashboardStore(t)
	sum := NewDashboardService(store).Summary("2.7.0")
	if sum.Links != 0 || sum.Groups != 0 || sum.ApplyRows != 0 {
		t.Fatalf("空库计数应为零: %+v", sum)
	}
	if sum.AppVersion != "2.7.0" {
		t.Fatalf("AppVersion 异常: %q", sum.AppVersion)
	}
	if sum.DBVersion != "" || sum.Build != "" {
		t.Fatalf("空库 version/build 应为空: %q %q", sum.DBVersion, sum.Build)
	}
}

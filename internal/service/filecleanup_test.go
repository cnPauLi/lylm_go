package service

import (
	"os"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"lylme_spage/internal/config"
	"lylme_spage/internal/models"
	"lylme_spage/internal/site"
)

func newFileCleanup(t *testing.T) (*FileCleanupService, *config.Config, *site.Store) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Config{}, &models.Group{}, &models.Link{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	root := t.TempDir()
	store, err := site.NewStore(db, root)
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}
	cfg := &config.Config{RootDir: root}
	for _, d := range []string{"download", "upload"} {
		if err := os.MkdirAll(cfg.Path("files", d), 0o755); err != nil {
			t.Fatalf("创建 files/%s 失败: %v", d, err)
		}
	}
	return NewFileCleanupService(store, cfg), cfg, store
}

func writeCleanupFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写文件 %s 失败: %v", path, err)
	}
}

func cleanupDirReport(r *CleanupReport, name string) *CleanupDir {
	for i := range r.Dirs {
		if r.Dirs[i].Name == name {
			return &r.Dirs[i]
		}
	}
	return nil
}

func TestFileCleanupScan(t *testing.T) {
	svc, cfg, store := newFileCleanup(t)

	// DB 引用：组图标引用 download/used.png；链接图标引用 upload/used2.gif
	if err := store.DB().Create(&models.Group{
		GroupName: "分组", GroupIcon: `<img src="/files/download/used.png">`,
	}).Error; err != nil {
		t.Fatalf("创建分组失败: %v", err)
	}
	if err := store.DB().Create(&models.Link{
		Name: "链接", Icon: "https://example.com/files/upload/used2.gif",
	}).Error; err != nil {
		t.Fatalf("创建链接失败: %v", err)
	}

	writeCleanupFile(t, cfg.Path("files", "download", "used.png"), "used")
	writeCleanupFile(t, cfg.Path("files", "download", "unused1.ico"), strings.Repeat("A", 2048))
	writeCleanupFile(t, cfg.Path("files", "upload", "used2.gif"), "used2")
	writeCleanupFile(t, cfg.Path("files", "upload", "unused2.txt"), strings.Repeat("B", 1000))
	// 子目录在扫描时跳过
	if err := os.MkdirAll(cfg.Path("files", "download", "subdir"), 0o755); err != nil {
		t.Fatalf("创建子目录失败: %v", err)
	}

	report, err := svc.Scan()
	if err != nil {
		t.Fatalf("Scan 失败: %v", err)
	}
	if report.Total != 4 {
		t.Fatalf("Total 应为 4（子目录跳过），实际 %d", report.Total)
	}
	if report.Used != 2 {
		t.Fatalf("Used 应为 2，实际 %d", report.Used)
	}
	if report.Unused != 2 {
		t.Fatalf("Unused 应为 2，实际 %d", report.Unused)
	}
	if report.TotalSize != 3048 {
		t.Fatalf("TotalSize 应为 3048，实际 %d", report.TotalSize)
	}
	if report.TotalSizeStr != "2.98 KB" {
		t.Fatalf("TotalSizeStr 异常: %q", report.TotalSizeStr)
	}

	dl := cleanupDirReport(report, "download")
	if dl == nil || dl.Unused != 1 || len(dl.Files) != 1 {
		t.Fatalf("download 目录统计异常: %+v", dl)
	}
	f := dl.Files[0]
	if f.Rel != "download/unused1.ico" || f.Name != "unused1.ico" || f.Dir != "download" {
		t.Fatalf("download 未使用文件标识异常: %+v", f)
	}
	if f.URL != "/files/download/unused1.ico" {
		t.Fatalf("URL 异常: %q", f.URL)
	}
	if f.Ext != "ico" || !f.IsImg || f.Size != 2048 || f.SizeStr != "2.00 KB" {
		t.Fatalf("download 文件类型/大小异常: %+v", f)
	}

	up := cleanupDirReport(report, "upload")
	if up == nil || up.Unused != 1 || len(up.Files) != 1 {
		t.Fatalf("upload 目录统计异常: %+v", up)
	}
	u := up.Files[0]
	if u.Rel != "upload/unused2.txt" || u.Ext != "txt" || u.IsImg {
		t.Fatalf("upload 未使用文件异常: %+v", u)
	}
	if u.Size != 1000 || u.SizeStr != "1000 B" {
		t.Fatalf("upload 文件大小异常: %+v", u)
	}
}

func TestFileCleanupDelete(t *testing.T) {
	svc, cfg, _ := newFileCleanup(t)
	a := cfg.Path("files", "download", "a.png")
	b := cfg.Path("files", "upload", "b.png")
	writeCleanupFile(t, a, "a")
	writeCleanupFile(t, b, "b")

	res := svc.Delete([]string{"download/a.png", "upload/b.png"})
	if res.Deleted != 2 || res.Failed != 0 {
		t.Fatalf("删除计数异常: %+v", res)
	}
	if _, err := os.Stat(a); !os.IsNotExist(err) {
		t.Fatalf("a.png 应已删除")
	}
	if _, err := os.Stat(b); !os.IsNotExist(err) {
		t.Fatalf("b.png 应已删除")
	}

	// 再次删除（文件已不存在）→ 失败
	res2 := svc.Delete([]string{"download/a.png"})
	if res2.Deleted != 0 || res2.Failed != 1 {
		t.Fatalf("删除不存在文件计数异常: %+v", res2)
	}

	// rel 合法但目标为目录 → 失败且不删目录
	dir := cfg.Path("files", "download", "adir")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	res3 := svc.Delete([]string{"download/adir"})
	if res3.Deleted != 0 || res3.Failed != 1 {
		t.Fatalf("删除目录应失败: %+v", res3)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("目录不应被删除: %v", err)
	}
}

func TestFileCleanupDeleteInvalid(t *testing.T) {
	svc, cfg, _ := newFileCleanup(t)
	keep := cfg.Path("files", "download", "keep.png")
	writeCleanupFile(t, keep, "keep")

	rels := []string{
		"",
		"   ",
		"download/",
		"download/a/b.png",
		"other/keep.png",
		"download/..",
		"../files/download/keep.png",
		"upload/../download/keep.png",
		`download\..\..\keep.png`,
		"download/keep.png/",
	}
	res := svc.Delete(rels)
	if res.Deleted != 0 {
		t.Fatalf("全部非法时 Deleted 应为 0，实际 %d", res.Deleted)
	}
	if res.Failed != len(rels) {
		t.Fatalf("Failed 应为 %d，实际 %d", len(rels), res.Failed)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("keep.png 不应被删除: %v", err)
	}
}

func TestFileCleanupDeleteMixed(t *testing.T) {
	svc, cfg, _ := newFileCleanup(t)
	del := cfg.Path("files", "download", "del.png")
	keep := cfg.Path("files", "download", "keep.png")
	writeCleanupFile(t, del, "del")
	writeCleanupFile(t, keep, "keep")

	res := svc.Delete([]string{"download/del.png", "download/..", ""})
	if res.Deleted != 1 || res.Failed != 2 {
		t.Fatalf("混合批次计数异常: %+v", res)
	}
	if _, err := os.Stat(del); !os.IsNotExist(err) {
		t.Fatalf("del.png 应已删除")
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("keep.png 不应被删除: %v", err)
	}
}

func TestFormatSize(t *testing.T) {
	cases := []struct {
		b    int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KB"},
		{2048, "2.00 KB"},
		{1 << 20, "1.00 MB"},
	}
	for _, c := range cases {
		if got := FormatSize(c.b); got != c.want {
			t.Fatalf("FormatSize(%d) = %q, want %q", c.b, got, c.want)
		}
	}
}

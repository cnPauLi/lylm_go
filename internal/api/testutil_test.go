package api

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"lylme_spage/internal/config"
	"lylme_spage/internal/models"
	"lylme_spage/internal/site"
)

// newTestV1 构建一套内存数据库 + 全部 service 的测试环境，
// register 回调负责挂载被测域的路由；返回引擎、依赖与可用 key。
// 中间件关闭限流，未携带 Authorization 的请求仍返回 401。
func newTestV1(t *testing.T, register func(v1 *gin.RouterGroup, d *Deps)) (*gin.Engine, *Deps, string) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(models.AllModels()...); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	root := t.TempDir()
	store, err := site.NewStore(db, root)
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}
	d := NewDeps(store, &config.Config{RootDir: root}, "test")
	key, err := d.Keys.Ensure()
	if err != nil {
		t.Fatalf("Ensure key 失败: %v", err)
	}
	r := gin.New()
	v1 := r.Group("/api/v1", RequireKeyWithLimit(d.Keys, 0, 0))
	register(v1, d)
	return r, d, key
}

package service

import (
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"lylme_spage/internal/models"
	"lylme_spage/internal/site"
)

func newGroupStore(t *testing.T) *site.Store {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Config{}, &models.Group{}, &models.Link{}, &models.Pwd{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	store, err := site.NewStore(db, t.TempDir())
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}
	return store
}

func TestGroupCreateDefaults(t *testing.T) {
	svc := NewGroupService(newGroupStore(t))

	rec, err := svc.Create(GroupInput{Name: " 分组A ", Icon: " https://x/a.png ", Pwd: 0})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if rec.GroupName != "分组A" {
		t.Fatalf("名称未 trim: %q", rec.GroupName)
	}
	if rec.GroupOrder != 1 || rec.GroupStatus != 1 {
		t.Fatalf("默认 order/status 异常: order=%d status=%d", rec.GroupOrder, rec.GroupStatus)
	}
	wantIcon := `<img src="https://x/a.png" class="lylme-group-img" />`
	if rec.GroupIcon != wantIcon {
		t.Fatalf("图标未规范化: %q", rec.GroupIcon)
	}

	// SVG/HTML 片段原样保留；order 递增
	svg := `<svg xmlns="http://www.w3.org/2000/svg"></svg>`
	rec2, err := svc.Create(GroupInput{Name: "分组B", Icon: svg})
	if err != nil {
		t.Fatalf("第二次 Create 失败: %v", err)
	}
	if rec2.GroupOrder != 2 || rec2.GroupIcon != svg {
		t.Fatalf("第二组 order/icon 异常: order=%d icon=%q", rec2.GroupOrder, rec2.GroupIcon)
	}

	// 空图标保持空串
	rec3, err := svc.Create(GroupInput{Name: "分组C", Icon: "  "})
	if err != nil {
		t.Fatalf("第三次 Create 失败: %v", err)
	}
	if rec3.GroupIcon != "" || rec3.GroupOrder != 3 {
		t.Fatalf("空图标组异常: icon=%q order=%d", rec3.GroupIcon, rec3.GroupOrder)
	}
}

func TestGroupCreateEmptyName(t *testing.T) {
	svc := NewGroupService(newGroupStore(t))
	for _, name := range []string{"", "   ", "\t\n"} {
		if _, err := svc.Create(GroupInput{Name: name}); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("空名称 %q 应返回 ErrInvalidInput，实际: %v", name, err)
		}
	}
}

func TestGroupUpdate(t *testing.T) {
	svc := NewGroupService(newGroupStore(t))
	rec, err := svc.Create(GroupInput{Name: "旧名", Icon: "http://x/old.png", Pwd: 0})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}

	updated, err := svc.Update(rec.GroupID, GroupInput{Name: " 新名 ", Icon: "http://x/new.png", Pwd: 2})
	if err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	if updated.GroupName != "新名" || updated.GroupPwd != 2 {
		t.Fatalf("更新字段异常: %+v", updated)
	}
	if updated.GroupIcon != `<img src="http://x/new.png" class="lylme-group-img" />` {
		t.Fatalf("更新后图标未规范化: %q", updated.GroupIcon)
	}

	// 空名称
	if _, err := svc.Update(rec.GroupID, GroupInput{Name: " "}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("空名称应返回 ErrInvalidInput，实际: %v", err)
	}
}

func TestGroupUpdateNotFound(t *testing.T) {
	svc := NewGroupService(newGroupStore(t))
	if _, err := svc.Update(999, GroupInput{Name: "x"}); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("不存在的分组应返回 ErrGroupNotFound，实际: %v", err)
	}
	if _, err := svc.Get(999); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("Get 不存在分组应返回 ErrGroupNotFound，实际: %v", err)
	}
}

func TestGroupDeleteCascade(t *testing.T) {
	store := newGroupStore(t)
	svc := NewGroupService(store)

	g1, err := svc.Create(GroupInput{Name: "组1"})
	if err != nil {
		t.Fatalf("Create g1 失败: %v", err)
	}
	g2, err := svc.Create(GroupInput{Name: "组2"})
	if err != nil {
		t.Fatalf("Create g2 失败: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := store.DB().Create(&models.Link{Name: "l1", GroupID: g1.GroupID, LinkStatus: 1}).Error; err != nil {
			t.Fatalf("创建 g1 链接失败: %v", err)
		}
	}
	if err := store.DB().Create(&models.Link{Name: "l2", GroupID: g2.GroupID, LinkStatus: 1}).Error; err != nil {
		t.Fatalf("创建 g2 链接失败: %v", err)
	}

	if err := svc.Delete(g1.GroupID); err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}

	var g1Links, g2Links int64
	store.DB().Model(&models.Link{}).Where("group_id = ?", g1.GroupID).Count(&g1Links)
	store.DB().Model(&models.Link{}).Where("group_id = ?", g2.GroupID).Count(&g2Links)
	if g1Links != 0 {
		t.Fatalf("被删组链接应级联删除，剩余 %d 条", g1Links)
	}
	if g2Links != 1 {
		t.Fatalf("其他组链接不应受影响，剩余 %d 条", g2Links)
	}
	var groups int64
	store.DB().Model(&models.Group{}).Count(&groups)
	if groups != 1 {
		t.Fatalf("删除后应剩 1 个分组，实际 %d", groups)
	}
}

func TestGroupSetStatus(t *testing.T) {
	store := newGroupStore(t)
	svc := NewGroupService(store)
	rec, err := svc.Create(GroupInput{Name: "组1"})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if err := svc.SetStatus(rec.GroupID, 0); err != nil {
		t.Fatalf("停用失败: %v", err)
	}
	row, err := svc.Get(rec.GroupID)
	if err != nil {
		t.Fatalf("Get 失败: %v", err)
	}
	if row.GroupStatus != 0 {
		t.Fatalf("停用后 status 应为 0，实际 %d", row.GroupStatus)
	}
	if err := svc.SetStatus(rec.GroupID, 1); err != nil {
		t.Fatalf("启用失败: %v", err)
	}
	if row, _ = svc.Get(rec.GroupID); row.GroupStatus != 1 {
		t.Fatalf("启用后 status 应为 1，实际 %d", row.GroupStatus)
	}
}

func TestGroupSort(t *testing.T) {
	store := newGroupStore(t)
	svc := NewGroupService(store)
	var ids []int
	for _, n := range []string{"A", "B", "C"} {
		rec, err := svc.Create(GroupInput{Name: n})
		if err != nil {
			t.Fatalf("Create %s 失败: %v", n, err)
		}
		ids = append(ids, rec.GroupID)
	}
	// 逆序排列：C,A,B
	if err := svc.Sort([]int{ids[2], ids[0], ids[1]}); err != nil {
		t.Fatalf("Sort 失败: %v", err)
	}
	list, err := svc.List()
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	var gotOrder []int
	for _, g := range list.Groups {
		gotOrder = append(gotOrder, g.GroupOrder)
	}
	wantIDs := []int{ids[2], ids[0], ids[1]}
	var gotIDs []int
	for _, g := range list.Groups {
		gotIDs = append(gotIDs, g.GroupID)
	}
	if len(gotOrder) != 3 || gotOrder[0] != 0 || gotOrder[1] != 1 || gotOrder[2] != 2 {
		t.Fatalf("排序下标异常: %v", gotOrder)
	}
	if len(gotIDs) != 3 || gotIDs[0] != wantIDs[0] || gotIDs[1] != wantIDs[1] || gotIDs[2] != wantIDs[2] {
		t.Fatalf("排序顺序异常: %v，期望 %v", gotIDs, wantIDs)
	}
}

func TestGroupListAggregations(t *testing.T) {
	store := newGroupStore(t)
	svc := NewGroupService(store)

	pwd := models.Pwd{PwdName: "密码组", PwdKey: "secret", PwdPs: ""}
	if err := store.DB().Create(&pwd).Error; err != nil {
		t.Fatalf("创建加密组失败: %v", err)
	}
	g1, err := svc.Create(GroupInput{Name: "组1", Pwd: pwd.PwdID})
	if err != nil {
		t.Fatalf("Create g1 失败: %v", err)
	}
	if _, err := svc.Create(GroupInput{Name: "组2"}); err != nil {
		t.Fatalf("Create g2 失败: %v", err)
	}
	for i := 0; i < 2; i++ {
		if err := store.DB().Create(&models.Link{Name: "l", GroupID: g1.GroupID, LinkStatus: 1}).Error; err != nil {
			t.Fatalf("创建链接失败: %v", err)
		}
	}

	list, err := svc.List()
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if list.Total != 2 {
		t.Fatalf("Total 应为 2，实际 %d", list.Total)
	}
	if list.PwdMap[pwd.PwdID] != "secret" {
		t.Fatalf("PwdMap 异常: %v", list.PwdMap)
	}
	if list.LinkCount[g1.GroupID] != 2 {
		t.Fatalf("链接计数异常: %v", list.LinkCount)
	}
	if len(list.Pwds) != 1 {
		t.Fatalf("Pwds 列表异常: %d", len(list.Pwds))
	}
}

func TestNormalizeGroupIcon(t *testing.T) {
	cases := map[string]string{
		"":                          "",
		"http://x/a.png":            `<img src="http://x/a.png" class="lylme-group-img" />`,
		"<svg></svg>":               "<svg></svg>",
		`<img src="a.png" />`:       `<img src="a.png" />`,
	}
	for in, want := range cases {
		if got := NormalizeGroupIcon(in); got != want {
			t.Fatalf("NormalizeGroupIcon(%q) = %q, want %q", in, got, want)
		}
	}
}

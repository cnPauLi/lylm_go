package service

import (
	"errors"
	"testing"
	"time"

	"lylme_spage/internal/models"
)

func TestApplyCreate(t *testing.T) {
	store, _ := newLinkStore(t)
	svc := NewApplyService(store)

	rec, err := svc.Create(ApplyCreateInput{
		Name: "六零导航", URL: "https://lylme.com", GroupID: 1, Icon: "icon", Desc: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if rec.ApplyID <= 0 {
		t.Fatalf("ApplyID 未生成")
	}
	if rec.ApplyStatus != 0 {
		t.Fatalf("新申请 status 应为 0，实际 %d", rec.ApplyStatus)
	}
	if rec.ApplyTime.IsZero() {
		t.Fatalf("apply_time 应自动生成")
	}

	got, err := svc.Get(rec.ApplyID)
	if err != nil {
		t.Fatalf("Get 失败: %v", err)
	}
	if got.ApplyName != "六零导航" || got.ApplyURL != "https://lylme.com" ||
		got.ApplyGroup != 1 || got.ApplyIcon != "icon" || got.ApplyDesc != "127.0.0.1" {
		t.Fatalf("落库字段异常: %+v", got)
	}

	if _, err := svc.Get(9999); !errors.Is(err, ErrApplyNotFound) {
		t.Fatalf("不存在应返回 ErrApplyNotFound，实际: %v", err)
	}
}

func TestApplyList(t *testing.T) {
	store, _ := newLinkStore(t)
	svc := NewApplyService(store)

	g1 := models.Group{GroupName: "分组A", GroupOrder: 2}
	g2 := models.Group{GroupName: "分组B", GroupOrder: 1}
	if err := store.DB().Create(&g1).Error; err != nil {
		t.Fatalf("建分组1失败: %v", err)
	}
	if err := store.DB().Create(&g2).Error; err != nil {
		t.Fatalf("建分组2失败: %v", err)
	}

	// 依次写入 3 条不同状态的申请（时间递增），直接落模型以构造非 0 状态
	for i, st := range []int{0, 1, 2} {
		row := models.Apply{
			ApplyName:   "站点" + string(rune('A'+i)),
			ApplyURL:    "https://example.com/" + string(rune('a'+i)),
			ApplyGroup:  g1.GroupID,
			ApplyStatus: st,
			ApplyTime:   time.Now().Add(time.Duration(i) * time.Hour),
		}
		if err := store.DB().Create(&row).Error; err != nil {
			t.Fatalf("建申请失败: %v", err)
		}
	}

	all, err := svc.List(-1)
	if err != nil {
		t.Fatalf("List(-1) 失败: %v", err)
	}
	if len(all.Rows) != 3 {
		t.Fatalf("全部应返回 3 条，实际 %d", len(all.Rows))
	}
	// apply_time DESC：最新一条（站点C）在最前
	if all.Rows[0].ApplyName != "站点C" {
		t.Fatalf("排序异常，首条: %q", all.Rows[0].ApplyName)
	}
	// 分组 group_order ASC：分组B 在前
	if len(all.Groups) != 2 || all.Groups[0].GroupName != "分组B" {
		t.Fatalf("分组排序异常: %+v", all.Groups)
	}
	if name := all.GroupMap[g1.GroupID]; name != "分组A" {
		t.Fatalf("GroupMap[%d] 应为 分组A，实际 %q", g1.GroupID, name)
	}
	if name := all.GroupMap[g2.GroupID]; name != "分组B" {
		t.Fatalf("GroupMap[%d] 应为 分组B，实际 %q", g2.GroupID, name)
	}

	for st, want := range map[int]int{0: 1, 1: 1, 2: 1, 9: 0} {
		res, err := svc.List(st)
		if err != nil {
			t.Fatalf("List(%d) 失败: %v", st, err)
		}
		if len(res.Rows) != want {
			t.Fatalf("List(%d) 应返回 %d 条，实际 %d", st, want, len(res.Rows))
		}
	}
}

func TestApplySettings(t *testing.T) {
	store, _ := newLinkStore(t)
	svc := NewApplyService(store)

	def := svc.Settings()
	if def.Open != 0 || def.GG != "" {
		t.Fatalf("默认设置异常: %+v", def)
	}

	if err := svc.UpdateSettings(1, "公告内容"); err != nil {
		t.Fatalf("UpdateSettings 失败: %v", err)
	}
	got := svc.Settings()
	if got.Open != 1 || got.GG != "公告内容" {
		t.Fatalf("设置写入后异常: %+v", got)
	}

	if err := svc.UpdateSettings(2, ""); err != nil {
		t.Fatalf("二次 UpdateSettings 失败: %v", err)
	}
	got = svc.Settings()
	if got.Open != 2 || got.GG != "" {
		t.Fatalf("关闭设置异常: %+v", got)
	}
}

func TestApplyUpdate(t *testing.T) {
	store, _ := newLinkStore(t)
	svc := NewApplyService(store)

	rec, err := svc.Create(ApplyCreateInput{Name: "旧名", URL: "https://old.com", GroupID: 1})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}

	// name 空 → ErrInvalidInput
	if err := svc.Update(rec.ApplyID, ApplyUpdateInput{Name: ""}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("空 name 应返回 ErrInvalidInput，实际: %v", err)
	}

	// 不存在：原版按 WHERE 更新，影响 0 行也不报错
	if err := svc.Update(9999, ApplyUpdateInput{Name: "新名", URL: "https://x.com"}); err != nil {
		t.Fatalf("不存在记录更新不应报错，实际: %v", err)
	}

	// 正常更新
	if err := svc.Update(rec.ApplyID, ApplyUpdateInput{
		Name: "新名称", URL: "https://new.com", GroupID: 5, Icon: "ic",
	}); err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	got, err := svc.Get(rec.ApplyID)
	if err != nil {
		t.Fatalf("Get 失败: %v", err)
	}
	if got.ApplyName != "新名称" || got.ApplyURL != "https://new.com" ||
		got.ApplyGroup != 5 || got.ApplyIcon != "ic" {
		t.Fatalf("更新字段异常: %+v", got)
	}
}

func TestApplySetStatusApprove(t *testing.T) {
	store, _ := newLinkStore(t)
	svc := NewApplyService(store)

	g := models.Group{GroupName: "目标分组", GroupOrder: 1}
	if err := store.DB().Create(&g).Error; err != nil {
		t.Fatalf("建分组失败: %v", err)
	}
	rec, err := svc.Create(ApplyCreateInput{
		Name: "<b>好站</b>", URL: "https://good.com/<x>", GroupID: g.GroupID, Icon: "<svg></svg>",
	})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}

	if err := svc.SetStatus([]int{rec.ApplyID}, 1); err != nil {
		t.Fatalf("SetStatus 通过失败: %v", err)
	}

	got, err := svc.Get(rec.ApplyID)
	if err != nil {
		t.Fatalf("Get 失败: %v", err)
	}
	if got.ApplyStatus != 1 {
		t.Fatalf("通过后状态应为 1，实际 %d", got.ApplyStatus)
	}

	var links []models.Link
	if err := store.DB().Find(&links).Error; err != nil {
		t.Fatalf("查询链接失败: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("通过应恰好新增 1 条链接，实际 %d", len(links))
	}
	l := links[0]
	if l.Name != "好站" {
		t.Fatalf("链接名称应 stripTags，实际 %q", l.Name)
	}
	if l.URL != "https://good.com/" {
		t.Fatalf("链接 URL 应 stripTags，实际 %q", l.URL)
	}
	if l.Icon != "<svg></svg>" {
		t.Fatalf("icon 应原样写入，实际 %q", l.Icon)
	}
	if l.GroupID != g.GroupID {
		t.Fatalf("group_id 异常: %d", l.GroupID)
	}
	if l.LinkOrder != 1 {
		t.Fatalf("link_order 应为 max(id)+1=1，实际 %d", l.LinkOrder)
	}
	if l.LinkStatus != 1 {
		t.Fatalf("link_status 应为 1，实际 %d", l.LinkStatus)
	}
}

func TestApplySetStatusRejectAndIdempotent(t *testing.T) {
	store, _ := newLinkStore(t)
	svc := NewApplyService(store)

	approved, err := svc.Create(ApplyCreateInput{Name: "通过站", URL: "https://ok.com", GroupID: 1})
	if err != nil {
		t.Fatalf("Create approved 失败: %v", err)
	}
	rejected, err := svc.Create(ApplyCreateInput{Name: "拒绝站", URL: "https://no.com", GroupID: 1})
	if err != nil {
		t.Fatalf("Create rejected 失败: %v", err)
	}

	// 拒绝：仅置 2，不新增链接
	if err := svc.SetStatus([]int{rejected.ApplyID}, 2); err != nil {
		t.Fatalf("SetStatus 拒绝失败: %v", err)
	}
	rj, err := svc.Get(rejected.ApplyID)
	if err != nil {
		t.Fatalf("Get rejected 失败: %v", err)
	}
	if rj.ApplyStatus != 2 {
		t.Fatalf("拒绝后状态应为 2，实际 %d", rj.ApplyStatus)
	}

	// 通过
	if err := svc.SetStatus([]int{approved.ApplyID}, 1); err != nil {
		t.Fatalf("SetStatus 通过失败: %v", err)
	}

	var linkCount int64
	if err := store.DB().Model(&models.Link{}).Count(&linkCount).Error; err != nil {
		t.Fatalf("统计链接失败: %v", err)
	}
	if linkCount != 1 {
		t.Fatalf("此刻应恰好 1 条链接，实际 %d", linkCount)
	}

	// 对已通过记录重复通过：幂等，不新增链接
	if err := svc.SetStatus([]int{approved.ApplyID}, 1); err != nil {
		t.Fatalf("重复通过报错: %v", err)
	}
	// 对已拒绝记录再通过：不新增链接、状态保持 2
	if err := svc.SetStatus([]int{rejected.ApplyID}, 1); err != nil {
		t.Fatalf("已拒绝再通过报错: %v", err)
	}
	still, _ := svc.Get(rejected.ApplyID)
	if still.ApplyStatus != 2 {
		t.Fatalf("已拒绝记录状态应保持 2，实际 %d", still.ApplyStatus)
	}

	if err := store.DB().Model(&models.Link{}).Count(&linkCount).Error; err != nil {
		t.Fatalf("二次统计链接失败: %v", err)
	}
	if linkCount != 1 {
		t.Fatalf("重复审核不应新增链接，实际 %d 条", linkCount)
	}
}

func TestApplySetStatusInvalid(t *testing.T) {
	store, _ := newLinkStore(t)
	svc := NewApplyService(store)

	rec, err := svc.Create(ApplyCreateInput{Name: "站", URL: "https://x.com", GroupID: 1})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	for _, st := range []int{0, 3, -1} {
		if err := svc.SetStatus([]int{rec.ApplyID}, st); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("status=%d 应返回 ErrInvalidInput，实际: %v", st, err)
		}
	}
	got, _ := svc.Get(rec.ApplyID)
	if got.ApplyStatus != 0 {
		t.Fatalf("非法 status 不应改变记录，实际状态 %d", got.ApplyStatus)
	}
}

func TestApplyBatchDelete(t *testing.T) {
	store, _ := newLinkStore(t)
	svc := NewApplyService(store)

	var ids []int
	for _, u := range []string{"https://1.com", "https://2.com", "https://3.com"} {
		rec, err := svc.Create(ApplyCreateInput{Name: "站", URL: u, GroupID: 1})
		if err != nil {
			t.Fatalf("Create 失败: %v", err)
		}
		ids = append(ids, rec.ApplyID)
	}

	// 空切片不报错
	if err := svc.BatchDelete(nil); err != nil {
		t.Fatalf("空切片不应报错，实际: %v", err)
	}

	if err := svc.BatchDelete([]int{ids[0], ids[1]}); err != nil {
		t.Fatalf("BatchDelete 失败: %v", err)
	}
	if _, err := svc.Get(ids[0]); !errors.Is(err, ErrApplyNotFound) {
		t.Fatalf("删除后应查不到，实际: %v", err)
	}
	if _, err := svc.Get(ids[1]); !errors.Is(err, ErrApplyNotFound) {
		t.Fatalf("删除后应查不到，实际: %v", err)
	}
	if _, err := svc.Get(ids[2]); err != nil {
		t.Fatalf("未删除记录应仍可查，实际: %v", err)
	}
}

package service

import (
	"errors"
	"testing"

	"lylme_spage/internal/models"
)

func TestTagList(t *testing.T) {
	store, _ := newLinkStore(t)
	svc := NewTagService(store)

	// 空表：返回非报错的空结果
	rows, err := svc.List()
	if err != nil {
		t.Fatalf("空表 List 失败: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("空表应返回 0 条，实际 %d", len(rows))
	}

	seeds := []models.Tag{
		{TagName: "菜单B", TagLink: "/b", Sort: 20},
		{TagName: "菜单A", TagLink: "/a", Sort: 1},
		{TagName: "菜单C", TagLink: "/c", Sort: 10},
	}
	for i := range seeds {
		if err := store.DB().Create(&seeds[i]).Error; err != nil {
			t.Fatalf("创建种子失败: %v", err)
		}
	}

	got, err := svc.List()
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("应返回 3 条，实际 %d", len(got))
	}
	wantOrder := []string{"菜单A", "菜单C", "菜单B"}
	for i, want := range wantOrder {
		if got[i].TagName != want {
			t.Fatalf("sort ASC 排序异常，位置 %d 应为 %q，实际 %q", i, want, got[i].TagName)
		}
	}
}

func TestTagCreate(t *testing.T) {
	store, _ := newLinkStore(t)
	svc := NewTagService(store)

	// name/link 去空白；sort=0 默认 10；target 按 int 原样
	rec, err := svc.Create(TagInput{Name: " 首页 ", Link: " / ", Sort: 0, Target: 1})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if rec.TagID <= 0 {
		t.Fatalf("ID 未生成")
	}
	if rec.TagName != "首页" || rec.TagLink != "/" {
		t.Fatalf("name/link 未 trim: %+v", rec)
	}
	if rec.Sort != 10 {
		t.Fatalf("默认 sort 应为 10，实际 %d", rec.Sort)
	}
	if rec.TagTarget != 1 {
		t.Fatalf("target 应为 1，实际 %d", rec.TagTarget)
	}

	// 显式 sort 保留。tag_target 列带 default:1：Create 传 0 时 GORM 省略零值、
	// 落库为默认值 1（与改造前处理器直接 Create 的行为一致）
	rec2, err := svc.Create(TagInput{Name: "外部", Link: "https://x.com", Sort: 25, Target: 0})
	if err != nil {
		t.Fatalf("第二次 Create 失败: %v", err)
	}
	if rec2.Sort != 25 || rec2.TagTarget != 1 {
		t.Fatalf("显式 sort / target 默认值异常: %+v", rec2)
	}

	// 必填校验
	for _, in := range []TagInput{
		{Name: "", Link: "/x"},
		{Name: "x", Link: ""},
		{Name: "   ", Link: "	"},
	} {
		if _, err := svc.Create(in); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("入参 %+v 应返回 ErrInvalidInput，实际: %v", in, err)
		}
	}
	var total int64
	store.DB().Model(&models.Tag{}).Count(&total)
	if total != 2 {
		t.Fatalf("校验失败不应落库，实际总数 %d", total)
	}
}

func TestTagUpdate(t *testing.T) {
	store, _ := newLinkStore(t)
	svc := NewTagService(store)

	rec, err := svc.Create(TagInput{Name: "旧名", Link: "/old", Sort: 3, Target: 0})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}

	row, err := svc.Update(rec.TagID, TagInput{Name: " 新名 ", Link: " /new ", Sort: 0, Target: 1})
	if err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	if row.TagName != "新名" || row.TagLink != "/new" {
		t.Fatalf("字段未更新/trim 异常: %+v", row)
	}
	if row.Sort != 10 || row.TagTarget != 1 {
		t.Fatalf("默认 sort / target 异常: %+v", row)
	}

	// 不存在：先于必填校验
	if _, err := svc.Update(9999, TagInput{Name: "", Link: ""}); !errors.Is(err, ErrTagNotFound) {
		t.Fatalf("不存在应返回 ErrTagNotFound，实际: %v", err)
	}

	// 已存在但必填缺失
	if _, err := svc.Update(rec.TagID, TagInput{Name: " ", Link: " "}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("必填缺失应返回 ErrInvalidInput，实际: %v", err)
	}
}

func TestTagDelete(t *testing.T) {
	store, _ := newLinkStore(t)
	svc := NewTagService(store)

	rec, err := svc.Create(TagInput{Name: "待删", Link: "/del", Sort: 1})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}

	if err := svc.Delete(rec.TagID); err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}
	if _, err := svc.Get(rec.TagID); !errors.Is(err, ErrTagNotFound) {
		t.Fatalf("删除后应查不到，实际: %v", err)
	}

	// 删除不存在的记录：影响 0 行，不报错（与原版一致）
	if err := svc.Delete(9999); err != nil {
		t.Fatalf("删除不存在的记录不应报错，实际: %v", err)
	}
}

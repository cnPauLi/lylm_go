package service

import (
	"errors"
	"testing"

	"lylme_spage/internal/models"
)

func TestSouList(t *testing.T) {
	store, _ := newLinkStore(t)
	svc := NewSouService(store)

	// 空表
	rows, err := svc.List()
	if err != nil {
		t.Fatalf("空表 List 失败: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("空表应返回 0 条，实际 %d", len(rows))
	}

	seeds := []models.Sou{
		{SouName: "引擎B", SouOrder: 2},
		{SouName: "引擎A", SouOrder: 1},
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
	if len(got) != 2 || got[0].SouName != "引擎A" || got[1].SouName != "引擎B" {
		t.Fatalf("sou_order ASC 排序异常: %+v", got)
	}
}

func TestSouCreate(t *testing.T) {
	store, _ := newLinkStore(t)
	svc := NewSouService(store)

	// 字段按原始值落库；st 按 int 原样；order=count+1
	rec, err := svc.Create(SouInput{
		Alias: " bd ", Name: " 百度 ", Hint: "搜一下", Color: "#fff",
		Link: "https://www.baidu.com/s?wd=", Waplink: "https://wap.baidu.com",
		Icon: "<svg></svg>", St: 1,
	})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if rec.SouID <= 0 {
		t.Fatalf("ID 未生成")
	}
	if rec.SouName != " 百度 " || rec.SouAlias != " bd " {
		t.Fatalf("字段应按原始值落库: %+v", rec)
	}
	if rec.SouOrder != 1 || rec.SouSt != 1 {
		t.Fatalf("默认 order / st 异常: order=%d st=%d", rec.SouOrder, rec.SouSt)
	}

	// 第二条 order 递增。sou_st 列带 default:1：Create 传 0 时 GORM 省略零值、
	// 落库为默认值 1（与改造前处理器直接 Create 的行为一致；Update 走 Save 才会写 0）
	rec2, err := svc.Create(SouInput{Name: "B站", Link: "https://b.com", St: 0})
	if err != nil {
		t.Fatalf("第二次 Create 失败: %v", err)
	}
	if rec2.SouOrder != 2 || rec2.SouSt != 1 {
		t.Fatalf("第二条 order / st 默认值异常: %+v", rec2)
	}

	// 仅 waplink 有值：原版校验只看 name/alias/hint/link/color/icon 六字段，
	// 六字段全空即报错，waplink 单独有值不能通过
	if _, err := svc.Create(SouInput{Waplink: "https://only-wap.com"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("仅 waplink 有值应返回 ErrInvalidInput，实际: %v", err)
	}

	// 参与校验的六字段全部 trim 后为空 → ErrInvalidInput
	for _, in := range []SouInput{
		{},
		{Name: "  ", Alias: "	", Hint: "", Link: "", Color: "", Icon: ""},
	} {
		if _, err := svc.Create(in); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("入参 %+v 应返回 ErrInvalidInput，实际: %v", in, err)
		}
	}
	// 任一字段非空即通过
	if _, err := svc.Create(SouInput{Hint: " "}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("仅空白 hint 应返回 ErrInvalidInput，实际: %v", err)
	}
	if _, err := svc.Create(SouInput{Icon: "i"}); err != nil {
		t.Fatalf("icon 非空应通过，实际: %v", err)
	}
}

func TestSouUpdate(t *testing.T) {
	store, _ := newLinkStore(t)
	svc := NewSouService(store)

	rec, err := svc.Create(SouInput{Name: "旧引擎", Alias: "old", Order: 1})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}

	row, err := svc.Update(rec.SouID, SouInput{
		Alias: "new", Name: "新引擎", Hint: "hint", Color: "#000",
		Link: "https://new.com", Waplink: "https://new-wap.com", Icon: "icon",
		St: 0, Order: 99,
	})
	if err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	if row.SouName != "新引擎" || row.SouAlias != "new" || row.SouHint != "hint" ||
		row.SouColor != "#000" || row.SouLink != "https://new.com" ||
		row.SouWaplink != "https://new-wap.com" || row.SouIcon != "icon" {
		t.Fatalf("字段更新异常: %+v", row)
	}
	if row.SouSt != 0 || row.SouOrder != 99 {
		t.Fatalf("st/order 更新异常: %+v", row)
	}

	// 原版修改不做必填校验：全空提交同样保存成功（清空列）
	empty, err := svc.Update(rec.SouID, SouInput{})
	if err != nil {
		t.Fatalf("全空 Update 不应报错，实际: %v", err)
	}
	if empty.SouName != "" || empty.SouOrder != 0 {
		t.Fatalf("全空 Update 应清空字段: %+v", empty)
	}

	// 不存在 → ErrSouNotFound
	if _, err := svc.Update(9999, SouInput{Name: "x"}); !errors.Is(err, ErrSouNotFound) {
		t.Fatalf("不存在应返回 ErrSouNotFound，实际: %v", err)
	}
}

func TestSouDelete(t *testing.T) {
	store, _ := newLinkStore(t)
	svc := NewSouService(store)

	rec, err := svc.Create(SouInput{Name: "待删", Order: 1})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}

	if err := svc.Delete(rec.SouID); err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}
	if _, err := svc.Get(rec.SouID); !errors.Is(err, ErrSouNotFound) {
		t.Fatalf("删除后应查不到，实际: %v", err)
	}

	// 删除不存在的记录不报错（与原版一致）
	if err := svc.Delete(9999); err != nil {
		t.Fatalf("删除不存在的记录不应报错，实际: %v", err)
	}
}

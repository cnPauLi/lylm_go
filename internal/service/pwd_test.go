package service

import (
	"errors"
	"testing"
)

func TestPwdList(t *testing.T) {
	store, _ := newLinkStore(t)
	svc := NewPwdService(store)

	// 空表
	rows, err := svc.List()
	if err != nil {
		t.Fatalf("空表 List 失败: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("空表应返回 0 条，实际 %d", len(rows))
	}

	if _, err := svc.Create(PwdInput{Name: "组A", Key: "key-a"}); err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if _, err := svc.Create(PwdInput{Name: "组B", Key: "key-b"}); err != nil {
		t.Fatalf("第二次 Create 失败: %v", err)
	}

	got, err := svc.List()
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("应返回 2 条，实际 %d", len(got))
	}
}

func TestPwdCreate(t *testing.T) {
	store, _ := newLinkStore(t)
	svc := NewPwdService(store)

	// name/key trim 后落库；ps 按原始值
	rec, err := svc.Create(PwdInput{Name: " 默认组 ", Key: " secret ", Ps: " 备注原文 "})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if rec.PwdID <= 0 {
		t.Fatalf("ID 未生成")
	}
	if rec.PwdName != "默认组" || rec.PwdKey != "secret" {
		t.Fatalf("name/key 未 trim: %+v", rec)
	}
	if rec.PwdPs != " 备注原文 " {
		t.Fatalf("ps 应按原始值落库: %q", rec.PwdPs)
	}

	// name/key 必填
	for _, in := range []PwdInput{
		{Name: "", Key: "k"},
		{Name: "n", Key: ""},
		{Name: "  ", Key: "	"},
	} {
		if _, err := svc.Create(in); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("入参 %+v 应返回 ErrInvalidInput，实际: %v", in, err)
		}
	}
}

func TestPwdUpdate(t *testing.T) {
	store, _ := newLinkStore(t)
	svc := NewPwdService(store)

	rec, err := svc.Create(PwdInput{Name: "旧组", Key: "old-key", Ps: "旧备注"})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}

	row, err := svc.Update(rec.PwdID, PwdInput{Name: " 新组 ", Key: " new-key ", Ps: "新备注"})
	if err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	if row.PwdName != "新组" || row.PwdKey != "new-key" || row.PwdPs != "新备注" {
		t.Fatalf("字段更新异常: %+v", row)
	}

	// 不存在：先于必填校验
	if _, err := svc.Update(9999, PwdInput{Name: "", Key: ""}); !errors.Is(err, ErrPwdNotFound) {
		t.Fatalf("不存在应返回 ErrPwdNotFound，实际: %v", err)
	}

	// 已存在但必填缺失
	if _, err := svc.Update(rec.PwdID, PwdInput{Name: " ", Key: " "}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("必填缺失应返回 ErrInvalidInput，实际: %v", err)
	}
}

func TestPwdDelete(t *testing.T) {
	store, _ := newLinkStore(t)
	svc := NewPwdService(store)

	rec, err := svc.Create(PwdInput{Name: "待删", Key: "key"})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}

	if err := svc.Delete(rec.PwdID); err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}
	if _, err := svc.Get(rec.PwdID); !errors.Is(err, ErrPwdNotFound) {
		t.Fatalf("删除后应查不到，实际: %v", err)
	}

	// 删除不存在的记录不报错（与原版一致）
	if err := svc.Delete(9999); err != nil {
		t.Fatalf("删除不存在的记录不应报错，实际: %v", err)
	}
}

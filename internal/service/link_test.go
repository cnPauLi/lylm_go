package service

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"lylme_spage/internal/config"
	"lylme_spage/internal/models"
	"lylme_spage/internal/site"
)

// newLinkStore 构建内存 sqlite（迁移全部模型）与最小 config，返回 store/cfg
func newLinkStore(t *testing.T) (*site.Store, *config.Config) {
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
	return store, &config.Config{RootDir: root}
}

func TestLinkGroupTable(t *testing.T) {
	store, cfg := newLinkStore(t)
	svc := NewLinkService(store, cfg)

	pwd := models.Pwd{PwdName: "加密组", PwdKey: "secret"}
	if err := store.DB().Create(&pwd).Error; err != nil {
		t.Fatalf("创建加密组失败: %v", err)
	}
	g1 := models.Group{GroupName: "分组A", GroupOrder: 1, GroupPwd: pwd.PwdID}
	g2 := models.Group{GroupName: "分组B", GroupOrder: 2}
	if err := store.DB().Create(&g1).Error; err != nil {
		t.Fatalf("创建 g1 失败: %v", err)
	}
	if err := store.DB().Create(&g2).Error; err != nil {
		t.Fatalf("创建 g2 失败: %v", err)
	}
	links := []models.Link{
		{Name: "L-高", GroupID: g1.GroupID, LinkOrder: 5, LinkStatus: 1},
		{Name: "L-低", GroupID: g1.GroupID, LinkOrder: 2, LinkStatus: 1},
		{Name: "L-B", GroupID: g2.GroupID, LinkOrder: 1, LinkStatus: 1},
	}
	for _, l := range links {
		if err := store.DB().Create(&l).Error; err != nil {
			t.Fatalf("创建链接失败: %v", err)
		}
	}

	gt := svc.GroupTable(g1.GroupID)
	if gt.Page != g1.GroupID {
		t.Fatalf("Page 异常: %d", gt.Page)
	}
	if gt.Total != 3 {
		t.Fatalf("Total 应为 3，实际 %d", gt.Total)
	}
	if gt.GroupCount != 2 {
		t.Fatalf("GroupCount 应为 2，实际 %d", gt.GroupCount)
	}
	if gt.GPwd != pwd.PwdID {
		t.Fatalf("GPwd 异常: %d", gt.GPwd)
	}
	if gt.GroupNames[g1.GroupID] != "分组A" || gt.GroupNames[g2.GroupID] != "分组B" {
		t.Fatalf("GroupNames 异常: %v", gt.GroupNames)
	}
	if len(gt.Links) != 2 || gt.Links[0].Name != "L-低" || gt.Links[1].Name != "L-高" {
		t.Fatalf("组内链接/排序异常: %+v", gt.Links)
	}

	// page<=0 归一为 1
	for _, in := range []int{0, -5} {
		zero := svc.GroupTable(in)
		if zero.Page != 1 || zero.GroupCount != 2 {
			t.Fatalf("GroupTable(%d) 未归一为分组1: %+v", in, zero)
		}
	}

	gt2 := svc.GroupTable(g2.GroupID)
	if gt2.GroupCount != 1 || gt2.GPwd != 0 || gt2.Links[0].Name != "L-B" {
		t.Fatalf("分组B 表格数据异常: %+v", gt2)
	}
}

func TestLinkGetNotFound(t *testing.T) {
	store, cfg := newLinkStore(t)
	svc := NewLinkService(store, cfg)

	if _, err := svc.Get(9999); !errors.Is(err, ErrLinkNotFound) {
		t.Fatalf("不存在的链接应返回 ErrLinkNotFound，实际: %v", err)
	}

	res, err := svc.Create(LinkInput{Name: "存在", URL: "https://exists.com"})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	row, err := svc.Get(res.Link.ID)
	if err != nil {
		t.Fatalf("Get 失败: %v", err)
	}
	if row.Name != "存在" {
		t.Fatalf("Get 数据异常: %+v", row)
	}
}

func TestLinkAll(t *testing.T) {
	store, cfg := newLinkStore(t)
	svc := NewLinkService(store, cfg)

	// 空表：返回非 nil 空切片
	if empty := mustLinks(t, svc); len(empty) != 0 {
		t.Fatalf("空表应返回空简表，实际 %d 条", len(empty))
	}

	in := []LinkInput{
		{Name: "甲", URL: "https://a.com"},
		{Name: "乙", URL: "https://b.com"},
	}
	var ids []int
	for _, i := range in {
		res, err := svc.Create(i)
		if err != nil {
			t.Fatalf("Create 失败: %v", err)
		}
		ids = append(ids, res.Link.ID)
	}

	all := mustLinks(t, svc)
	if len(all) != 2 {
		t.Fatalf("All 应返回 2 条，实际 %d", len(all))
	}
	if all[0].ID != ids[0] || all[0].URL != "https://a.com" || all[0].Name != "甲" {
		t.Fatalf("首条简表异常: %+v", all[0])
	}
	if all[1].ID != ids[1] || all[1].Name != "乙" {
		t.Fatalf("第二条简表异常: %+v", all[1])
	}
}

func mustLinks(t *testing.T, svc *LinkService) []LinkBrief {
	t.Helper()
	all, err := svc.All()
	if err != nil {
		t.Fatalf("All 失败: %v", err)
	}
	return all
}

func TestLinkCreate(t *testing.T) {
	store, cfg := newLinkStore(t)
	svc := NewLinkService(store, cfg)

	res, err := svc.Create(LinkInput{
		Name: " 百度 ", URL: " https://www.baidu.com ", Color: " red ",
		Icon: "icon-x", GroupID: 7, Desc: " 描述 ", Keywords: " 关键词 ",
	})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if !res.Created || res.Duplicate {
		t.Fatalf("结果标志异常: %+v", res)
	}
	if res.Link.ID <= 0 {
		t.Fatalf("ID 未生成")
	}
	if res.Link.Name != `<font color="red">百度</font>` {
		t.Fatalf("color 未包裹: %q", res.Link.Name)
	}
	if res.Link.LinkOrder != 1 || res.Link.LinkStatus != 1 {
		t.Fatalf("默认 order/status 异常: order=%d status=%d", res.Link.LinkOrder, res.Link.LinkStatus)
	}
	if res.Link.URL != "https://www.baidu.com" || res.Link.GroupID != 7 ||
		res.Link.Icon != "icon-x" || res.Link.LinkDesc != " 描述 " || res.Link.LinkKeywords != " 关键词 " {
		t.Fatalf("落库字段异常: %+v", res.Link)
	}
	if res.Submitted.Name != "百度" || res.Submitted.Color != "red" {
		t.Fatalf("Submitted 异常: %+v", res.Submitted)
	}

	// 第二条 order 递增
	res2, err := svc.Create(LinkInput{Name: "B站", URL: "https://www.b.com"})
	if err != nil {
		t.Fatalf("第二次 Create 失败: %v", err)
	}
	if res2.Link.LinkOrder != 2 {
		t.Fatalf("第二条 order 应为 2，实际 %d", res2.Link.LinkOrder)
	}

	// color 超长截断（32，追加省略号）
	longColor := strings.Repeat("a", 40)
	res3, err := svc.Create(LinkInput{Name: "C", URL: "https://www.c.com", Color: longColor})
	if err != nil {
		t.Fatalf("第三次 Create 失败: %v", err)
	}
	if n := len([]rune(res3.Submitted.Color)); n != 32 {
		t.Fatalf("color 截断长度应为 32，实际 %d", n)
	}
	if !strings.HasSuffix(res3.Submitted.Color, "...") {
		t.Fatalf("color 截断应追加省略号: %q", res3.Submitted.Color)
	}

	// 必填校验
	for _, in := range []LinkInput{
		{Name: "x", URL: ""},
		{Name: "", URL: "https://x.com"},
		{Name: "  ", URL: "	"},
	} {
		if _, err := svc.Create(in); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("入参 %+v 应返回 ErrInvalidInput，实际: %v", in, err)
		}
	}
}

func TestLinkCreateDuplicate(t *testing.T) {
	store, cfg := newLinkStore(t)
	svc := NewLinkService(store, cfg)

	first, err := svc.Create(LinkInput{Name: "首条", URL: "https://example.com"})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}

	// 末尾多一个斜杠：判重，不新增
	dup, err := svc.Create(LinkInput{Name: "重复", URL: "https://example.com/"})
	if err != nil {
		t.Fatalf("重复提交不应产生错误，实际: %v", err)
	}
	if !dup.Duplicate || dup.Created {
		t.Fatalf("应返回 Duplicate 结果: %+v", dup)
	}
	if dup.Link.ID != first.Link.ID {
		t.Fatalf("Duplicate 应携带既有 ID: got %d want %d", dup.Link.ID, first.Link.ID)
	}
	if dup.Submitted.URL != "https://example.com/" || dup.Submitted.Name != "重复" {
		t.Fatalf("Submitted 应保留本次提交数据: %+v", dup.Submitted)
	}

	// 反向：库内存的是带斜杠，提交不带斜杠
	rev, err := svc.Create(LinkInput{Name: "反向", URL: "https://demo.org/path/"})
	if err != nil {
		t.Fatalf("Create 反向首条失败: %v", err)
	}
	revDup, err := svc.Create(LinkInput{Name: "反向重复", URL: "https://demo.org/path"})
	if err != nil {
		t.Fatalf("反向重复提交不应产生错误，实际: %v", err)
	}
	if !revDup.Duplicate || revDup.Link.ID != rev.Link.ID {
		t.Fatalf("反向判重异常: %+v", revDup)
	}

	var total int64
	store.DB().Model(&models.Link{}).Count(&total)
	if total != 2 {
		t.Fatalf("判重场景不应新增记录，实际总数 %d", total)
	}
}

func TestLinkUpdate(t *testing.T) {
	store, cfg := newLinkStore(t)
	svc := NewLinkService(store, cfg)

	rec, err := svc.Create(LinkInput{Name: "旧名", URL: "https://old.com", Color: "red"})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}

	row, err := svc.Update(rec.Link.ID, LinkInput{
		Name: "新名", URL: "https://new.com", Color: "blue",
		GroupID: 8, Desc: "新描述", Keywords: "新关键词", Icon: "new-icon", Pwd: 3,
	})
	if err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	if row.Name != `<font color="blue">新名</font>` {
		t.Fatalf("color 包裹异常: %q", row.Name)
	}
	if row.URL != "https://new.com" || row.GroupID != 8 || row.LinkPwd != 3 ||
		row.Icon != "new-icon" || row.LinkDesc != "新描述" || row.LinkKeywords != "新关键词" {
		t.Fatalf("更新字段异常: %+v", row)
	}

	// 不带 color：去掉旧包裹
	plain, err := svc.Update(rec.Link.ID, LinkInput{Name: "纯白", URL: "https://plain.com"})
	if err != nil {
		t.Fatalf("Update 去色失败: %v", err)
	}
	if plain.Name != "纯白" {
		t.Fatalf("去 color 后名称异常: %q", plain.Name)
	}

	// 不存在：先于必填校验
	if _, err := svc.Update(9999, LinkInput{Name: "", URL: ""}); !errors.Is(err, ErrLinkNotFound) {
		t.Fatalf("不存在的链接应返回 ErrLinkNotFound，实际: %v", err)
	}

	// 已存在但必填缺失
	if _, err := svc.Update(rec.Link.ID, LinkInput{Name: " ", URL: " "}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("必填缺失应返回 ErrInvalidInput，实际: %v", err)
	}
}

func TestLinkBatchOps(t *testing.T) {
	store, cfg := newLinkStore(t)
	svc := NewLinkService(store, cfg)

	var ids []int
	for _, n := range []string{"1", "2", "3"} {
		res, err := svc.Create(LinkInput{Name: n, URL: "https://" + n + ".com", GroupID: 10})
		if err != nil {
			t.Fatalf("Create %s 失败: %v", n, err)
		}
		ids = append(ids, res.Link.ID)
	}

	if err := svc.BatchSetStatus([]int{ids[0], ids[1]}, 0); err != nil {
		t.Fatalf("BatchSetStatus 失败: %v", err)
	}
	if err := svc.BatchMoveGroup([]int{ids[0], ids[1]}, 55); err != nil {
		t.Fatalf("BatchMoveGroup 失败: %v", err)
	}
	if err := svc.BatchSetPwd([]int{ids[0], ids[1]}, 66); err != nil {
		t.Fatalf("BatchSetPwd 失败: %v", err)
	}

	for _, i := range []int{0, 1} {
		l, err := svc.Get(ids[i])
		if err != nil {
			t.Fatalf("Get %d 失败: %v", ids[i], err)
		}
		if l.LinkStatus != 0 || l.GroupID != 55 || l.LinkPwd != 66 {
			t.Fatalf("链接 %d 批量字段异常: %+v", ids[i], l)
		}
	}
	// 未选中的第三条不受影响
	l3, err := svc.Get(ids[2])
	if err != nil {
		t.Fatalf("Get 第三条失败: %v", err)
	}
	if l3.LinkStatus != 1 || l3.GroupID != 10 || l3.LinkPwd != 0 {
		t.Fatalf("第三条不应受批量操作影响: %+v", l3)
	}

	if err := svc.BatchDelete([]int{ids[0], ids[1]}); err != nil {
		t.Fatalf("BatchDelete 失败: %v", err)
	}
	var total int64
	store.DB().Model(&models.Link{}).Count(&total)
	if total != 1 {
		t.Fatalf("删除后应剩 1 条，实际 %d", total)
	}

	// 空 ids 一律 ErrInvalidInput
	empty := func(e error) {
		t.Helper()
		if !errors.Is(e, ErrInvalidInput) {
			t.Fatalf("空 ids 应返回 ErrInvalidInput，实际: %v", e)
		}
	}
	empty(svc.BatchDelete(nil))
	empty(svc.BatchSetStatus(nil, 0))
	empty(svc.BatchMoveGroup(nil, 1))
	empty(svc.BatchSetPwd(nil, 1))
}

func TestLinkSort(t *testing.T) {
	store, cfg := newLinkStore(t)
	svc := NewLinkService(store, cfg)

	var ids []int
	for _, n := range []string{"A", "B", "C"} {
		res, err := svc.Create(LinkInput{Name: n, URL: "https://sort-" + n + ".com"})
		if err != nil {
			t.Fatalf("Create %s 失败: %v", n, err)
		}
		ids = append(ids, res.Link.ID)
	}

	// 期望顺序 C,A,B；末尾追加无法解析的元素（原版按下标赋值，坏元素仅被跳过但不影响其余下标）
	ordered := []string{
		strconv.Itoa(ids[2]), strconv.Itoa(ids[0]), strconv.Itoa(ids[1]), "bad-id",
	}
	if err := svc.Sort(ordered); err != nil {
		t.Fatalf("Sort 失败: %v", err)
	}

	want := map[int]int{ids[2]: 0, ids[0]: 1, ids[1]: 2}
	for id, order := range want {
		l, err := svc.Get(id)
		if err != nil {
			t.Fatalf("Get %d 失败: %v", id, err)
		}
		if l.LinkOrder != order {
			t.Fatalf("链接 %d order 应为 %d，实际 %d", id, order, l.LinkOrder)
		}
	}

	// 空数组不报错
	if err := svc.Sort(nil); err != nil {
		t.Fatalf("空 Sort 不应报错，实际: %v", err)
	}
}

func TestLinkListAndSearch(t *testing.T) {
	store, cfg := newLinkStore(t)
	svc := NewLinkService(store, cfg)

	g := models.Group{GroupName: "组一", GroupOrder: 1}
	if err := store.DB().Create(&g).Error; err != nil {
		t.Fatalf("创建分组失败: %v", err)
	}
	in := []LinkInput{
		{Name: "苹果", URL: "https://apple.com", GroupID: g.GroupID, Desc: "水果", Keywords: "red"},
		{Name: "香蕉", URL: "https://banana.com", GroupID: g.GroupID, Desc: "黄色水果"},
		{Name: "汽车", URL: "https://car.example", Keywords: "vehicle"},
	}
	for _, i := range in {
		if _, err := svc.Create(i); err != nil {
			t.Fatalf("Create %+v 失败: %v", i, err)
		}
	}

	// 全量分页
	rows, total, err := svc.ListLinks(0, 1, 10)
	if err != nil {
		t.Fatalf("ListLinks 失败: %v", err)
	}
	if total != 3 || len(rows) != 3 {
		t.Fatalf("全量分页异常: total=%d len=%d", total, len(rows))
	}

	// 按组过滤 + 参数归一（page=0）
	gRows, gTotal, err := svc.ListLinks(g.GroupID, 0, 1)
	if err != nil {
		t.Fatalf("ListLinks 分组过滤失败: %v", err)
	}
	if gTotal != 2 || len(gRows) != 1 || gRows[0].Name != "苹果" {
		t.Fatalf("分组过滤/分页异常: total=%d rows=%+v", gTotal, gRows)
	}

	// 关键词命中描述
	hit, hitTotal, err := svc.SearchLinks("水果", 1, 10)
	if err != nil {
		t.Fatalf("SearchLinks 失败: %v", err)
	}
	if hitTotal != 2 || len(hit) != 2 {
		t.Fatalf("关键词检索异常: total=%d len=%d", hitTotal, len(hit))
	}

	// 关键词命中 keywords 列
	kw, _, err := svc.SearchLinks("vehicle", 1, 10)
	if err != nil {
		t.Fatalf("SearchLinks keywords 列失败: %v", err)
	}
	if len(kw) != 1 || kw[0].Name != "汽车" {
		t.Fatalf("keywords 列检索异常: %+v", kw)
	}

	// 无命中
	none, noneTotal, err := svc.SearchLinks("不存在的词xyz", 1, 10)
	if err != nil {
		t.Fatalf("SearchLinks 无命中失败: %v", err)
	}
	if noneTotal != 0 || len(none) != 0 {
		t.Fatalf("无命中应返回空: total=%d len=%d", noneTotal, len(none))
	}
}

func TestLinkFetchHeadUnreachable(t *testing.T) {
	store, cfg := newLinkStore(t)
	svc := NewLinkService(store, cfg)

	for _, raw := range []string{"", "not a url", "ftp://example.com", "://bad"} {
		head, err := svc.FetchHead(raw)
		if !errors.Is(err, ErrLinkHeadUnreachable) {
			t.Fatalf("FetchHead(%q) 应返回 ErrLinkHeadUnreachable，实际: %v", raw, err)
		}
		if head == nil {
			t.Fatalf("FetchHead(%q) 即使不可达也应返回原始 map", raw)
		}
	}
}

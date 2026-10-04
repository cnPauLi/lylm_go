package service

import (
	"errors"
	"strconv"
	"strings"

	"lylme_spage/internal/config"
	"lylme_spage/internal/fetch"
	"lylme_spage/internal/models"
	"lylme_spage/internal/site"
)

// LinkInput 链接新增/修改入参（强类型、原始值；trim / 截断 / color 包裹由 service 完成）
type LinkInput struct {
	Name     string
	URL      string
	Color    string
	Icon     string
	GroupID  int
	Desc     string
	Keywords string
	Pwd      int
}

// LinkCreateResult 新增结果：
// Created=true 时 Link 为新建记录；Duplicate=true 时 Link 为既有记录、Submitted 为本次提交数据。
type LinkCreateResult struct {
	Created   bool
	Duplicate bool
	Link      models.Link
	Submitted LinkInput
}

// LinkBrief 链接简表（id/url/name）
type LinkBrief struct {
	ID   int    `json:"id"`
	URL  string `json:"url"`
	Name string `json:"name"`
}

// LinkGroupTable 按组查看的链接表格数据（供 table_link.html 片段）
type LinkGroupTable struct {
	Page       int // 当前查看的分组 ID（即原版 table_link.php 的 page 参数）
	Total      int64
	GroupCount int64
	GPwd       int
	Groups     []models.Group
	Links      []models.Link
	GroupNames map[int]string
}

// LinkService 链接域业务规则：按组查询、详情、简表、新增/修改、批量操作、排序、抓取 head
type LinkService struct {
	store *site.Store
	cfg   *config.Config
}

// NewLinkService 创建链接服务
func NewLinkService(store *site.Store, cfg *config.Config) *LinkService {
	return &LinkService{store: store, cfg: cfg}
}

// GroupTable 返回 group_order ASC 的全部分组（含 GroupNames 映射、当前组 GPwd）、
// 链接总数 Total、指定组链接（link_order ASC）与组内计数。
// groupID 语义上即原版 table_link.php 的 page 参数；<=0 归一为 1。
// 各查询相互独立、失败不阻断（与原 table_link.php 一致）：单个查询失败时对应字段保持零值。
func (s *LinkService) GroupTable(groupID int) *LinkGroupTable {
	if groupID <= 0 {
		groupID = 1
	}
	db := s.store.DB()
	out := &LinkGroupTable{Page: groupID, GroupNames: map[int]string{}}

	if err := db.Order("group_order ASC").Find(&out.Groups).Error; err != nil {
		// 与原版一致：不阻断渲染，仅放弃该部分数据
		_ = err
	}
	for _, g := range out.Groups {
		out.GroupNames[g.GroupID] = g.GroupName
		if g.GroupID == groupID {
			out.GPwd = g.GroupPwd
		}
	}

	_ = db.Model(&models.Link{}).Count(&out.Total).Error

	_ = db.Where("group_id = ?", groupID).Order("link_order ASC").Find(&out.Links).Error
	out.GroupCount = int64(len(out.Links))

	return out
}

// Get 按主键读取单条链接；不存在返回 ErrLinkNotFound
func (s *LinkService) Get(id int) (models.Link, error) {
	var row models.Link
	if err := s.store.DB().Where("id = ?", id).First(&row).Error; err != nil {
		return row, ErrLinkNotFound
	}
	return row, nil
}

// All 返回全量链接简表（id/url/name，id ASC）
func (s *LinkService) All() ([]LinkBrief, error) {
	var rows []models.Link
	if err := s.store.DB().Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]LinkBrief, 0, len(rows))
	for _, l := range rows {
		out = append(out, LinkBrief{ID: l.ID, URL: l.URL, Name: l.Name})
	}
	return out, nil
}

// Create 新增链接：
//   - name/url 必填，否则 ErrInvalidInput；
//   - url 去末尾斜杠查重（url = base OR base+"/"）；命中时返回 Duplicate 结果（携带既有 ID 与本次提交数据，
//     语义对应原版 code=201），不新增记录；
//   - color 非空时 name 包裹 <font color="...">；
//   - icon/desc/keywords 长度截断规则与原版一致；link_order=count+1、link_status=1。
func (s *LinkService) Create(in LinkInput) (*LinkCreateResult, error) {
	sub := LinkInput{
		Color:    truncateRunes(strings.TrimSpace(in.Color), 32),
		Name:     truncateRunes(strings.TrimSpace(in.Name), 255),
		URL:      truncateRunes(strings.TrimSpace(in.URL), 255),
		Icon:     in.Icon,
		GroupID:  in.GroupID,
		Desc:     truncateRunes(in.Desc, 255),
		Keywords: truncateRunes(in.Keywords, 512),
	}
	if sub.Name == "" || sub.URL == "" {
		return nil, ErrInvalidInput
	}

	db := s.store.DB()
	base := strings.TrimRight(sub.URL, "/")
	var existing models.Link
	if err := db.Where("url = ? OR url = ?", base, base+"/").First(&existing).Error; err == nil {
		return &LinkCreateResult{Duplicate: true, Link: existing, Submitted: sub}, nil
	}

	var total int64
	_ = db.Model(&models.Link{}).Count(&total).Error
	rec := models.Link{
		Name:         sub.Name,
		GroupID:      sub.GroupID,
		URL:          sub.URL,
		Icon:         sub.Icon,
		LinkDesc:     sub.Desc,
		LinkKeywords: sub.Keywords,
		LinkOrder:    int(total) + 1,
		LinkStatus:   1,
	}
	if sub.Color != "" {
		rec.Name = `<font color="` + sub.Color + `">` + sub.Name + `</font>`
	}
	if err := db.Create(&rec).Error; err != nil {
		return nil, err
	}
	return &LinkCreateResult{Created: true, Link: rec, Submitted: sub}, nil
}

// Update 修改链接：先做存在性校验（ErrLinkNotFound），再做必填校验（ErrInvalidInput），
// 顺序与原版一致；color 同样包裹；更新全部字段含 link_pwd。
func (s *LinkService) Update(id int, in LinkInput) (models.Link, error) {
	row, err := s.Get(id)
	if err != nil {
		return row, err
	}

	color := truncateRunes(strings.TrimSpace(in.Color), 32)
	name := truncateRunes(strings.TrimSpace(in.Name), 255)
	url := truncateRunes(strings.TrimSpace(in.URL), 255)
	if name == "" || url == "" {
		return row, ErrInvalidInput
	}
	if color != "" {
		name = `<font color="` + color + `">` + name + `</font>`
	}

	row.Name = name
	row.LinkDesc = truncateRunes(in.Desc, 255)
	row.LinkKeywords = truncateRunes(in.Keywords, 512)
	row.URL = url
	row.Icon = in.Icon
	row.GroupID = in.GroupID
	row.LinkPwd = in.Pwd
	if err := s.store.DB().Save(&row).Error; err != nil {
		return row, err
	}
	return row, nil
}

// BatchDelete 批量删除；空 ids 返回 ErrInvalidInput
func (s *LinkService) BatchDelete(ids []int) error {
	if len(ids) == 0 {
		return ErrInvalidInput
	}
	return s.store.DB().Where("id IN ?", ids).Delete(&models.Link{}).Error
}

// BatchSetStatus 批量设置启用状态（1 启用 / 0 停用）；空 ids 返回 ErrInvalidInput
func (s *LinkService) BatchSetStatus(ids []int, status int) error {
	if len(ids) == 0 {
		return ErrInvalidInput
	}
	return s.store.DB().Model(&models.Link{}).
		Where("id IN ?", ids).Update("link_status", status).Error
}

// BatchMoveGroup 批量移动分组；空 ids 返回 ErrInvalidInput
func (s *LinkService) BatchMoveGroup(ids []int, groupID int) error {
	if len(ids) == 0 {
		return ErrInvalidInput
	}
	return s.store.DB().Model(&models.Link{}).
		Where("id IN ?", ids).Update("group_id", groupID).Error
}

// BatchSetPwd 批量设置访问密码组；空 ids 返回 ErrInvalidInput
func (s *LinkService) BatchSetPwd(ids []int, pwdID int) error {
	if len(ids) == 0 {
		return ErrInvalidInput
	}
	return s.store.DB().Model(&models.Link{}).
		Where("id IN ?", ids).Update("link_pwd", pwdID).Error
}

// Sort 按传入数组下标写 link_order；无法解析为整数的元素跳过；
// 单个更新失败不阻断后续（由上层决定是否记录日志）。
func (s *LinkService) Sort(ordered []string) error {
	var errs []error
	for i, raw := range ordered {
		id, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil {
			continue
		}
		if err := s.store.DB().Model(&models.Link{}).
			Where("id = ?", id).Update("link_order", i).Error; err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// FetchHead 抓取目标站点 head：调用 fetch.GetHead（保持 SSRF 防护与缓存目录约定）。
// title/icon 皆空时返回 ErrLinkHeadUnreachable（由 admin 映射为原 code=400 语义）。
func (s *LinkService) FetchHead(rawURL string) (map[string]any, error) {
	head := fetch.GetHead(rawURL, s.cfg.Path("cache"), true)
	title, _ := head["title"].(string)
	icon, _ := head["icon"].(string)
	if title == "" && icon == "" {
		return head, ErrLinkHeadUnreachable
	}
	return head, nil
}

// truncateRunes 按字符数截断并追加省略号（等价原版 truncate_text）
func truncateRunes(s string, max int) string {
	if max < 1 {
		max = 1
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-3]) + "..."
}

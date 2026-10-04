package service

import (
	"errors"
	"regexp"
	"strconv"

	"lylme_spage/internal/models"
	"lylme_spage/internal/site"
)

// ApplyCreateInput 前台收录申请落库入参（原始值；必填/长度/URL 校验由调用侧完成）
type ApplyCreateInput struct {
	Name    string
	URL     string
	GroupID int
	Icon    string
	Desc    string
}

// ApplyUpdateInput 收录申请修改入参（对应 admin edit_submit）
type ApplyUpdateInput struct {
	Name    string
	URL     string
	GroupID int
	Icon    string
}

// ApplyListResult 收录列表结果：申请行、分组与 GroupID→Name 映射
type ApplyListResult struct {
	Rows     []models.Apply
	Groups   []models.Group
	GroupMap map[int]string
}

// ApplySettings 收录设置：Open 为 apply 键（0 开启需审核 / 1 开启免审核 / 2 关闭），GG 为公告
type ApplySettings struct {
	Open int
	GG   string
}

// ApplyService 收录申请域业务规则：列表、设置、读取、提交、修改、审核、删除
type ApplyService struct {
	store *site.Store
}

// NewApplyService 创建收录申请服务
func NewApplyService(store *site.Store) *ApplyService {
	return &ApplyService{store: store}
}

// List 返回申请列表：status<0 时返回全部，否则按 apply_status 过滤；
// 一律按 apply_time DESC 排序，同时返回分组（group_order ASC）与 GroupID→Name 映射。
func (s *ApplyService) List(status int) (ApplyListResult, error) {
	res := ApplyListResult{GroupMap: map[int]string{}}
	if err := s.store.DB().Order("group_order ASC").Find(&res.Groups).Error; err != nil {
		return res, err
	}
	for _, g := range res.Groups {
		res.GroupMap[g.GroupID] = g.GroupName
	}
	q := s.store.DB().Order("apply_time DESC")
	if status >= 0 {
		q = q.Where("apply_status = ?", status)
	}
	if err := q.Find(&res.Rows).Error; err != nil {
		return res, err
	}
	return res, nil
}

// Settings 读取收录开关与公告；apply 缺失/为空时按 0（开启-需要审核）处理
func (s *ApplyService) Settings() ApplySettings {
	open, _ := strconv.Atoi(s.store.Get("apply", "0"))
	return ApplySettings{Open: open, GG: s.store.Get("apply_gg", "")}
}

// UpdateSettings 保存收录开关与公告（写配置键 apply / apply_gg）；
// 两个键都会尝试写入，错误合并返回，由调用侧决定如何提示。
func (s *ApplyService) UpdateSettings(open int, gg string) error {
	err1 := s.store.Set("apply", strconv.Itoa(open), "")
	err2 := s.store.Set("apply_gg", gg, "")
	return errors.Join(err1, err2)
}

// Get 按主键读取单个申请；不存在返回 ErrApplyNotFound
func (s *ApplyService) Get(id int) (models.Apply, error) {
	var row models.Apply
	if err := s.store.DB().Where("apply_id = ?", id).First(&row).Error; err != nil {
		return row, ErrApplyNotFound
	}
	return row, nil
}

// Create 写入一条收录申请：apply_status=0、apply_time 自动生成；
// 字段校验由调用侧（前台 pages）完成，service 只承载落库规则。
func (s *ApplyService) Create(in ApplyCreateInput) (models.Apply, error) {
	rec := models.Apply{
		ApplyName:   in.Name,
		ApplyURL:    in.URL,
		ApplyGroup:  in.GroupID,
		ApplyIcon:   in.Icon,
		ApplyDesc:   in.Desc,
		ApplyStatus: 0,
	}
	if err := s.store.DB().Create(&rec).Error; err != nil {
		return rec, err
	}
	return rec, nil
}

// Update 修改申请（等价 admin 按 WHERE 更新）：name 为空 → ErrInvalidInput；
// 不预先校验记录存在性，影响 0 行也不报错（与原版一致）。
func (s *ApplyService) Update(id int, in ApplyUpdateInput) error {
	if in.Name == "" {
		return ErrInvalidInput
	}
	updates := map[string]any{
		"apply_name":  in.Name,
		"apply_url":   in.URL,
		"apply_group": in.GroupID,
		"apply_icon":  in.Icon,
	}
	return s.store.DB().Model(&models.Apply{}).Where("apply_id = ?", id).Updates(updates).Error
}

// SetStatus 批量审核：status 仅接受 1（通过）/2（拒绝），否则 ErrInvalidInput。
// 逐条处理且仅对 apply_status=0 的记录生效——通过时先写链接表再置 1，
// 拒绝仅置 2；已处理记录保持幂等（不重复写链接、不重复改状态）。
func (s *ApplyService) SetStatus(ids []int, status int) error {
	if status != 1 && status != 2 {
		return ErrInvalidInput
	}
	var firstErr error
	for _, id := range ids {
		if status == 1 {
			var row models.Apply
			if err := s.store.DB().Where("apply_status = ? AND apply_id = ?", 0, id).First(&row).Error; err == nil {
				if err := s.insertLink(&row); err != nil && firstErr == nil {
					firstErr = err
				}
			}
		}
		if err := s.store.DB().Model(&models.Apply{}).
			Where("apply_status = ? AND apply_id = ?", 0, id).Update("apply_status", status).Error; err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// BatchDelete 批量删除申请；空切片直接返回
func (s *ApplyService) BatchDelete(ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return s.store.DB().Where("apply_id IN ?", ids).Delete(&models.Apply{}).Error
}

// insertLink 审核通过时将申请写入链接表：名称/URL 去除标签、icon 原样、
// link_order=COALESCE(MAX(id),0)+1、link_status=1
func (s *ApplyService) insertLink(row *models.Apply) error {
	var maxID int
	if err := s.store.DB().Model(&models.Link{}).Select("COALESCE(MAX(id),0)").Scan(&maxID).Error; err != nil {
		return err
	}
	rec := models.Link{
		Name:       applyStripTags(row.ApplyName),
		GroupID:    row.ApplyGroup,
		URL:        applyStripTags(row.ApplyURL),
		Icon:       row.ApplyIcon,
		LinkDesc:   "",
		LinkOrder:  maxID + 1,
		LinkStatus: 1,
	}
	return s.store.DB().Create(&rec).Error
}

var applyTagRe = regexp.MustCompile(`<[^>]*>`)

// applyStripTags 去除 HTML 标签（等价 PHP strip_tags）
func applyStripTags(s string) string { return applyTagRe.ReplaceAllString(s, "") }

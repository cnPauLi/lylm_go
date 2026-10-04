package service

import (
	"errors"
	"strings"

	"lylme_spage/internal/models"
	"lylme_spage/internal/site"
)

// GroupInput 分组新增/修改入参（强类型，原始值；trim 与规范化由 service 完成）
type GroupInput struct {
	Name string
	Icon string
	Pwd  int
}

// GroupList 分组列表数据：供后台 group.php 页面与 table_group.php 片段共用
type GroupList struct {
	Total     int64
	Groups    []models.Group
	Pwds      []models.Pwd
	PwdMap    map[int]string
	LinkCount map[int]int
}

// GroupService 分组域业务规则：列表、新增、修改、删除、启停用、排序
type GroupService struct {
	store *site.Store
}

// NewGroupService 创建分组服务
func NewGroupService(store *site.Store) *GroupService {
	return &GroupService{store: store}
}

// List 返回全部分组（group_order ASC）、加密组及其映射、各组链接数与总数。
// 各查询相互独立：分组查询失败时返回该错误，加密组/计数查询失败不阻断其余数据
// （与原 table_group.php 行为一致）。
func (s *GroupService) List() (*GroupList, error) {
	db := s.store.DB()
	out := &GroupList{PwdMap: map[int]string{}, LinkCount: map[int]int{}}

	var listErr error
	if err := db.Order("group_order ASC").Find(&out.Groups).Error; err != nil {
		listErr = err
	}
	out.Total = int64(len(out.Groups))

	var pwds []models.Pwd
	if err := db.Find(&pwds).Error; err == nil {
		out.Pwds = pwds
		for _, p := range pwds {
			out.PwdMap[p.PwdID] = p.PwdKey
		}
	}

	type cntRow struct {
		GroupID int
		Cnt     int
	}
	var rows []cntRow
	if err := db.Model(&models.Link{}).
		Select("group_id, COUNT(id) AS cnt").Group("group_id").Scan(&rows).Error; err == nil {
		for _, r := range rows {
			out.LinkCount[r.GroupID] = r.Cnt
		}
	}

	return out, listErr
}

// Get 按主键读取单个分组；不存在返回 ErrGroupNotFound
func (s *GroupService) Get(id int) (models.Group, error) {
	var row models.Group
	if err := s.store.DB().Where("group_id = ?", id).First(&row).Error; err != nil {
		return row, ErrGroupNotFound
	}
	return row, nil
}

// Create 新增分组：name 必填；纯 URL 图标自动包裹 img 标签；
// group_order = COALESCE(MAX(group_order),0)+1；默认启用。
func (s *GroupService) Create(in GroupInput) (models.Group, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return models.Group{}, ErrGroupNameEmpty
	}
	rec := models.Group{
		GroupName:   name,
		GroupIcon:   NormalizeGroupIcon(strings.TrimSpace(in.Icon)),
		GroupOrder:  s.nextOrder(),
		GroupPwd:    in.Pwd,
		GroupStatus: 1,
	}
	if err := s.store.DB().Create(&rec).Error; err != nil {
		return rec, err
	}
	return rec, nil
}

// Update 修改分组：不存在返回 ErrGroupNotFound；name 必填；图标同样规范化。
func (s *GroupService) Update(id int, in GroupInput) (models.Group, error) {
	row, err := s.Get(id)
	if err != nil {
		return row, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return row, ErrGroupNameEmpty
	}
	row.GroupName = name
	row.GroupIcon = NormalizeGroupIcon(strings.TrimSpace(in.Icon))
	row.GroupPwd = in.Pwd
	if err := s.store.DB().Save(&row).Error; err != nil {
		return row, err
	}
	return row, nil
}

// Delete 删除分组：先删该分组全部链接，再删分组本身；两步均会执行。
func (s *GroupService) Delete(id int) error {
	var errs []error
	if err := s.store.DB().Where("group_id = ?", id).Delete(&models.Link{}).Error; err != nil {
		errs = append(errs, errors.New("删除分组链接失败: "+err.Error()))
	}
	if err := s.store.DB().Where("group_id = ?", id).Delete(&models.Group{}).Error; err != nil {
		errs = append(errs, errors.New("删除分组失败: "+err.Error()))
	}
	return errors.Join(errs...)
}

// SetStatus 更新分组启用状态（1 启用 / 0 停用）
func (s *GroupService) SetStatus(id int, status int) error {
	return s.store.DB().Model(&models.Group{}).
		Where("group_id = ?", id).Update("group_status", status).Error
}

// Sort 按传入 id 的顺序依次写入 group_order 下标；单个失败不阻断后续更新
func (s *GroupService) Sort(ids []int) error {
	var errs []error
	for i, id := range ids {
		if err := s.store.DB().Model(&models.Group{}).
			Where("group_id = ?", id).Update("group_order", i).Error; err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// nextOrder 返回 COALESCE(MAX(group_order),0)+1
func (s *GroupService) nextOrder() int {
	var maxOrder int
	_ = s.store.DB().Model(&models.Group{}).
		Select("COALESCE(MAX(group_order),0)").Scan(&maxOrder).Error
	return maxOrder + 1
}

// NormalizeGroupIcon 纯 URL（不以 "<" 开头）自动包裹 img 标签；
// 空串与 HTML/SVG 片段原样返回
func NormalizeGroupIcon(icon string) string {
	if icon != "" && !strings.HasPrefix(icon, "<") {
		return `<img src="` + icon + `" class="lylme-group-img" />`
	}
	return icon
}

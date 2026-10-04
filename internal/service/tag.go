package service

import (
	"strings"

	"lylme_spage/internal/models"
	"lylme_spage/internal/site"
)

// TagInput 导航菜单新增/修改入参（强类型、原始值；trim 与默认值由 service 完成）。
// Target 语义由处理器映射：仅提交值为 "1" 时传 1，其余一律 0。
type TagInput struct {
	Name   string
	Link   string
	Sort   int
	Target int
}

// TagService 导航菜单域业务规则：列表、新增、修改、删除
type TagService struct {
	store *site.Store
}

// NewTagService 创建导航菜单服务
func NewTagService(store *site.Store) *TagService {
	return &TagService{store: store}
}

// List 返回全部导航菜单（sort ASC）
func (s *TagService) List() ([]models.Tag, error) {
	var rows []models.Tag
	if err := s.store.DB().Order("sort ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// Get 按主键读取单条导航菜单；不存在返回 ErrTagNotFound
func (s *TagService) Get(id int) (models.Tag, error) {
	var row models.Tag
	if err := s.store.DB().Where("tag_id = ?", id).First(&row).Error; err != nil {
		return row, ErrTagNotFound
	}
	return row, nil
}

// Create 新增导航菜单：name/link 必填（trim 后为空 → ErrInvalidInput）；
// sort=0 时取默认值 10；target 按入参原样落库。
func (s *TagService) Create(in TagInput) (models.Tag, error) {
	name := strings.TrimSpace(in.Name)
	link := strings.TrimSpace(in.Link)
	if name == "" || link == "" {
		return models.Tag{}, ErrInvalidInput
	}
	rec := models.Tag{
		TagName:   name,
		TagLink:   link,
		TagTarget: in.Target,
		Sort:      defaultTagSort(in.Sort),
	}
	if err := s.store.DB().Create(&rec).Error; err != nil {
		return rec, err
	}
	return rec, nil
}

// Update 修改导航菜单：先查存在（ErrTagNotFound），再做必填校验（ErrInvalidInput），
// 顺序与原版一致；sort=0 时同样取默认值 10。
func (s *TagService) Update(id int, in TagInput) (models.Tag, error) {
	row, err := s.Get(id)
	if err != nil {
		return row, err
	}
	name := strings.TrimSpace(in.Name)
	link := strings.TrimSpace(in.Link)
	if name == "" || link == "" {
		return row, ErrInvalidInput
	}
	row.TagName = name
	row.TagLink = link
	row.TagTarget = in.Target
	row.Sort = defaultTagSort(in.Sort)
	if err := s.store.DB().Save(&row).Error; err != nil {
		return row, err
	}
	return row, nil
}

// Delete 删除导航菜单；记录不存在时 GORM 影响 0 行且不报错（与原版一致）
func (s *TagService) Delete(id int) error {
	return s.store.DB().Where("tag_id = ?", id).Delete(&models.Tag{}).Error
}

// defaultTagSort sort=0 时返回默认值 10
func defaultTagSort(sort int) int {
	if sort == 0 {
		return 10
	}
	return sort
}

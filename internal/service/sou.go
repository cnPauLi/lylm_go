package service

import (
	"strings"

	"lylme_spage/internal/models"
	"lylme_spage/internal/site"
)

// SouInput 搜索引擎新增/修改入参（强类型、原始值）。
// St 布尔语义由处理器映射（"1"/"true"/"on"/"yes" → 1）；Order 仅修改时使用。
type SouInput struct {
	Alias   string
	Name    string
	Hint    string
	Color   string
	Link    string
	Waplink string
	Icon    string
	St      int
	Order   int
}

// SouService 搜索引擎域业务规则：列表、新增、修改、删除
type SouService struct {
	store *site.Store
}

// NewSouService 创建搜索引擎服务
func NewSouService(store *site.Store) *SouService {
	return &SouService{store: store}
}

// List 返回全部搜索引擎（sou_order ASC）
func (s *SouService) List() ([]models.Sou, error) {
	var rows []models.Sou
	if err := s.store.DB().Order("sou_order ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// Get 按主键读取单个搜索引擎；不存在返回 ErrSouNotFound
func (s *SouService) Get(id int) (models.Sou, error) {
	var row models.Sou
	if err := s.store.DB().Where("sou_id = ?", id).First(&row).Error; err != nil {
		return row, ErrSouNotFound
	}
	return row, nil
}

// Create 新增搜索引擎：原版「全部字段皆空才报错」——name/alias/hint/link/color/icon
// 全部 trim 后为空时返回 ErrInvalidInput（waplink 不参与该校验，与原版一致）；
// sou_order = 当前记录数 + 1；其余字段按原始提交值落库。
func (s *SouService) Create(in SouInput) (models.Sou, error) {
	if souAllEmpty(in) {
		return models.Sou{}, ErrInvalidInput
	}
	var total int64
	_ = s.store.DB().Model(&models.Sou{}).Count(&total).Error
	rec := models.Sou{
		SouAlias:   in.Alias,
		SouName:    in.Name,
		SouHint:    in.Hint,
		SouColor:   in.Color,
		SouLink:    in.Link,
		SouWaplink: in.Waplink,
		SouIcon:    in.Icon,
		SouSt:      in.St,
		SouOrder:   int(total) + 1,
	}
	if err := s.store.DB().Create(&rec).Error; err != nil {
		return rec, err
	}
	return rec, nil
}

// Update 修改搜索引擎：仅校验存在性（ErrSouNotFound），不做必填校验
//（原版 ajax_edit_sou 保存任意提交内容）；全部字段含 sou_order 原样写入。
func (s *SouService) Update(id int, in SouInput) (models.Sou, error) {
	row, err := s.Get(id)
	if err != nil {
		return row, err
	}
	row.SouAlias = in.Alias
	row.SouName = in.Name
	row.SouHint = in.Hint
	row.SouColor = in.Color
	row.SouLink = in.Link
	row.SouWaplink = in.Waplink
	row.SouIcon = in.Icon
	row.SouSt = in.St
	row.SouOrder = in.Order
	if err := s.store.DB().Save(&row).Error; err != nil {
		return row, err
	}
	return row, nil
}

// Delete 删除搜索引擎；记录不存在时影响 0 行且不报错（与原版一致）
func (s *SouService) Delete(id int) error {
	return s.store.DB().Where("sou_id = ?", id).Delete(&models.Sou{}).Error
}

// souAllEmpty 判断参与校验的六个字段是否全部 trim 后为空
func souAllEmpty(in SouInput) bool {
	return strings.TrimSpace(in.Name) == "" &&
		strings.TrimSpace(in.Alias) == "" &&
		strings.TrimSpace(in.Hint) == "" &&
		strings.TrimSpace(in.Link) == "" &&
		strings.TrimSpace(in.Color) == "" &&
		strings.TrimSpace(in.Icon) == ""
}

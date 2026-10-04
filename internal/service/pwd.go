package service

import (
	"strings"

	"lylme_spage/internal/models"
	"lylme_spage/internal/site"
)

// PwdInput 加密组新增/修改入参（强类型、原始值；name/key 的 trim 由 service 完成）
type PwdInput struct {
	Name string
	Key  string
	Ps   string
}

// PwdService 加密组域业务规则：列表、新增、修改、删除
type PwdService struct {
	store *site.Store
}

// NewPwdService 创建加密组服务
func NewPwdService(store *site.Store) *PwdService {
	return &PwdService{store: store}
}

// List 返回全部加密组
func (s *PwdService) List() ([]models.Pwd, error) {
	var rows []models.Pwd
	if err := s.store.DB().Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// Get 按主键读取单个加密组；不存在返回 ErrPwdNotFound
func (s *PwdService) Get(id int) (models.Pwd, error) {
	var row models.Pwd
	if err := s.store.DB().Where("pwd_id = ?", id).First(&row).Error; err != nil {
		return row, ErrPwdNotFound
	}
	return row, nil
}

// Create 新增加密组：name/key 必填（trim 后为空 → ErrInvalidInput）；
// ps 按原始提交值落库。
func (s *PwdService) Create(in PwdInput) (models.Pwd, error) {
	name := strings.TrimSpace(in.Name)
	key := strings.TrimSpace(in.Key)
	if name == "" || key == "" {
		return models.Pwd{}, ErrInvalidInput
	}
	rec := models.Pwd{PwdName: name, PwdKey: key, PwdPs: in.Ps}
	if err := s.store.DB().Create(&rec).Error; err != nil {
		return rec, err
	}
	return rec, nil
}

// Update 修改加密组：先查存在（ErrPwdNotFound），再做必填校验（ErrInvalidInput）；
// ps 按原始提交值写入。
func (s *PwdService) Update(id int, in PwdInput) (models.Pwd, error) {
	row, err := s.Get(id)
	if err != nil {
		return row, err
	}
	name := strings.TrimSpace(in.Name)
	key := strings.TrimSpace(in.Key)
	if name == "" || key == "" {
		return row, ErrInvalidInput
	}
	row.PwdName = name
	row.PwdKey = key
	row.PwdPs = in.Ps
	if err := s.store.DB().Save(&row).Error; err != nil {
		return row, err
	}
	return row, nil
}

// Delete 删除加密组；记录不存在时影响 0 行且不报错（与原版一致）
func (s *PwdService) Delete(id int) error {
	return s.store.DB().Where("pwd_id = ?", id).Delete(&models.Pwd{}).Error
}

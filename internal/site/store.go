package site

import (
	"errors"
	"sync"

	"gorm.io/gorm"

	"lylme_spage/internal/models"
	"lylme_spage/internal/theme"
)

// GroupView 分组及其链接
type GroupView struct {
	models.Group
	Links []models.Link
}

// Store 数据访问层：网站配置缓存 + 前台数据查询
type Store struct {
	db    *gorm.DB
	theme *theme.Manager

	mu   sync.RWMutex
	conf map[string]string
}

// NewStore 创建数据访问层并加载配置
func NewStore(db *gorm.DB, root string) (*Store, error) {
	s := &Store{db: db, theme: theme.NewManager(root)}
	if err := s.Reload(); err != nil {
		return nil, err
	}
	return s, nil
}

// DB 返回底层数据库句柄
func (s *Store) DB() *gorm.DB { return s.db }

// Theme 返回主题管理器
func (s *Store) Theme() *theme.Manager { return s.theme }

// Reload 重新加载网站配置
func (s *Store) Reload() error {
	var rows []models.Config
	if err := s.db.Find(&rows).Error; err != nil {
		return err
	}
	m := make(map[string]string, len(rows))
	for _, r := range rows {
		m[r.K] = r.V
	}
	s.mu.Lock()
	s.conf = m
	s.mu.Unlock()
	return nil
}

// Conf 返回配置副本
func (s *Store) Conf() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.conf))
	for k, v := range s.conf {
		out[k] = v
	}
	return out
}

// Get 读取配置项
func (s *Store) Get(key, def string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := s.conf[key]; ok && v != "" {
		return v
	}
	return def
}

// Set 写入/更新配置项（upsert）并刷新缓存
func (s *Store) Set(key, value, desc string) error {
	var c models.Config
	err := s.db.Where("k = ?", key).First(&c).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		c = models.Config{K: key, V: value, Description: desc}
		if err := s.db.Create(&c).Error; err != nil {
			return err
		}
	case err != nil:
		return err
	default:
		c.V = value
		if desc != "" {
			c.Description = desc
		}
		if err := s.db.Save(&c).Error; err != nil {
			return err
		}
	}
	return s.Reload()
}

// Template 当前启用的主题目录名
func (s *Store) Template() string {
	t := s.Get("template", "default")
	if !s.theme.Exists(t) {
		return "default"
	}
	return t
}

// Tags 顶部导航菜单
func (s *Store) Tags() ([]models.Tag, error) {
	var rows []models.Tag
	err := s.db.Order("sort ASC").Find(&rows).Error
	return rows, err
}

// SouEnabled 启用的搜索引擎
func (s *Store) SouEnabled() ([]models.Sou, error) {
	var rows []models.Sou
	err := s.db.Where("sou_st = ?", 1).Order("sou_order ASC").Find(&rows).Error
	return rows, err
}

// Groups 返回前台可见分组及其链接（按 group_order / link_order 排序，处理加密可见性）
func (s *Store) Groups(unlocked []int) ([]GroupView, error) {
	var groups []models.Group
	if err := s.db.Order("group_order ASC").Find(&groups).Error; err != nil {
		return nil, err
	}

	unlockSet := make(map[int]bool, len(unlocked))
	for _, id := range unlocked {
		unlockSet[id] = true
	}

	var out []GroupView
	for _, g := range groups {
		if g.GroupStatus == 0 {
			continue
		}
		if g.GroupPwd != 0 && !unlockSet[g.GroupPwd] {
			continue
		}
		var links []models.Link
		if err := s.db.Where("group_id = ?", g.GroupID).Order("link_order ASC").Find(&links).Error; err != nil {
			return nil, err
		}
		filtered := make([]models.Link, 0, len(links))
		for _, l := range links {
			if l.LinkStatus == 0 {
				continue
			}
			if g.GroupPwd == 0 && l.LinkPwd != 0 && !unlockSet[l.LinkPwd] {
				continue
			}
			filtered = append(filtered, l)
		}
		out = append(out, GroupView{Group: g, Links: filtered})
	}
	return out, nil
}

// UnlockedPwdIDs 读取会话中已解锁的加密组 ID
func UnlockedPwdIDs(vals []int) []int { return vals }

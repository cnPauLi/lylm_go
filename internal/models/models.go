package models

import "time"

// Config 网站配置（键值对）
type Config struct {
	ID          int    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	K           string `gorm:"column:k;size:30;uniqueIndex" json:"k"`
	V           string `gorm:"column:v;type:text" json:"v"`
	Description string `gorm:"column:description;size:30" json:"description"`
}

// TableName 指定表名
func (Config) TableName() string { return "lylme_config" }

// Link 导航链接
type Link struct {
	ID           int    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name         string `gorm:"column:name;size:255" json:"name"`
	GroupID      int    `gorm:"column:group_id;index" json:"group_id"`
	URL          string `gorm:"column:url;size:255" json:"url"`
	Icon         string `gorm:"column:icon;type:text" json:"icon"`
	LinkDesc     string `gorm:"column:link_desc;size:255" json:"link_desc"`
	LinkKeywords string `gorm:"column:link_keywords;size:512" json:"link_keywords"`
	LinkOrder    int    `gorm:"column:link_order;default:10" json:"link_order"`
	LinkStatus   int    `gorm:"column:link_status;default:1" json:"link_status"`
	LinkPwd      int    `gorm:"column:link_pwd;default:0;index" json:"link_pwd"`
}

// TableName 指定表名
func (Link) TableName() string { return "lylme_links" }

// Group 链接分组
type Group struct {
	GroupID     int    `gorm:"column:group_id;primaryKey;autoIncrement" json:"group_id"`
	GroupName   string `gorm:"column:group_name;size:30" json:"group_name"`
	GroupIcon   string `gorm:"column:group_icon;type:text" json:"group_icon"`
	GroupOrder  int    `gorm:"column:group_order;default:5" json:"group_order"`
	GroupStatus int    `gorm:"column:group_status;default:1" json:"group_status"`
	GroupPwd    int    `gorm:"column:group_pwd;default:0;index" json:"group_pwd"`
}

// TableName 指定表名
func (Group) TableName() string { return "lylme_groups" }

// Apply 收录申请
type Apply struct {
	ApplyID     int       `gorm:"column:apply_id;primaryKey;autoIncrement" json:"apply_id"`
	ApplyName   string    `gorm:"column:apply_name;size:60" json:"apply_name"`
	ApplyURL    string    `gorm:"column:apply_url;size:255" json:"apply_url"`
	ApplyGroup  int       `gorm:"column:apply_group;index" json:"apply_group"`
	ApplyIcon   string    `gorm:"column:apply_icon;type:text" json:"apply_icon"`
	ApplyDesc   string    `gorm:"column:apply_desc;size:255" json:"apply_desc"`
	ApplyTime   time.Time `gorm:"column:apply_time;autoCreateTime" json:"apply_time"`
	ApplyStatus int       `gorm:"column:apply_status;default:0;index" json:"apply_status"`
}

// TableName 指定表名
func (Apply) TableName() string { return "lylme_apply" }

// Pwd 加密访问密码组
type Pwd struct {
	PwdID   int    `gorm:"column:pwd_id;primaryKey;autoIncrement" json:"pwd_id"`
	PwdName string `gorm:"column:pwd_name;size:30" json:"pwd_name"`
	PwdKey  string `gorm:"column:pwd_key;size:60" json:"pwd_key"`
	PwdPs   string `gorm:"column:pwd_ps;size:60" json:"pwd_ps"`
}

// TableName 指定表名
func (Pwd) TableName() string { return "lylme_pwd" }

// Sou 搜索引擎
type Sou struct {
	SouID      int    `gorm:"column:sou_id;primaryKey;autoIncrement" json:"sou_id"`
	SouAlias   string `gorm:"column:sou_alias;size:30" json:"sou_alias"`
	SouName    string `gorm:"column:sou_name;size:30" json:"sou_name"`
	SouHint    string `gorm:"column:sou_hint;size:60;default:请输入搜索关键词" json:"sou_hint"`
	SouColor   string `gorm:"column:sou_color;size:20;default:#696a6d" json:"sou_color"`
	SouLink    string `gorm:"column:sou_link;size:255" json:"sou_link"`
	SouWaplink string `gorm:"column:sou_waplink;size:255" json:"sou_waplink"`
	SouIcon    string `gorm:"column:sou_icon;type:text" json:"sou_icon"`
	SouSt      int    `gorm:"column:sou_st;default:1;index" json:"sou_st"`
	SouOrder   int    `gorm:"column:sou_order;index" json:"sou_order"`
}

// TableName 指定表名
func (Sou) TableName() string { return "lylme_sou" }

// Tag 顶部导航菜单
type Tag struct {
	TagID     int    `gorm:"column:tag_id;primaryKey;autoIncrement" json:"tag_id"`
	TagName   string `gorm:"column:tag_name;size:30" json:"tag_name"`
	TagLink   string `gorm:"column:tag_link;size:255" json:"tag_link"`
	TagTarget int    `gorm:"column:tag_target;default:1" json:"tag_target"`
	Sort      int    `gorm:"column:sort;default:10;index" json:"sort"`
}

// TableName 指定表名
func (Tag) TableName() string { return "lylme_tags" }

// AllModels 返回所有需要迁移的模型
func AllModels() []any {
	return []any{
		&Config{}, &Link{}, &Group{}, &Apply{}, &Pwd{}, &Sou{}, &Tag{},
	}
}

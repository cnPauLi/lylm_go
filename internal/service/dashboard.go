package service

import (
	"lylme_spage/internal/models"
	"lylme_spage/internal/site"
)

// DashboardSummary 仪表盘汇总：计数 + 版本信息。
// 不含访问统计（stats.Summary）：访问统计由处理器从 stats.Store 读取后合并到渲染上下文，
// service 不依赖 stats 包。
type DashboardSummary struct {
	Links      int64  // 链接总数
	Groups     int64  // 分组总数
	ApplyRows  int64  // 待审申请数（apply_status=0）
	AppVersion string // 程序版本
	DBVersion  string // 数据库版本（配置键 version）
	Build      string // 建站时间（配置键 build）
}

// DashboardService 仪表盘业务规则：链接/分组/待审申请计数与版本信息组装
type DashboardService struct {
	store *site.Store
}

// NewDashboardService 创建仪表盘服务
func NewDashboardService(store *site.Store) *DashboardService {
	return &DashboardService{store: store}
}

// Summary 组装仪表盘汇总。appVersion 为程序版本（由处理器传入，service 不持有版本常量）。
// 各计数查询相互独立：单项失败时对应字段保持零值（与原 admin.Dashboard 一致），
// version/build 取自配置缓存。
func (s *DashboardService) Summary(appVersion string) DashboardSummary {
	db := s.store.DB()
	out := DashboardSummary{
		AppVersion: appVersion,
		DBVersion:  s.store.Get("version", ""),
		Build:      s.store.Get("build", ""),
	}
	_ = db.Model(&models.Link{}).Count(&out.Links).Error
	_ = db.Model(&models.Group{}).Count(&out.Groups).Error
	_ = db.Model(&models.Apply{}).Where("apply_status = ?", 0).Count(&out.ApplyRows).Error
	return out
}

package api

import (
	"lylme_spage/internal/config"
	"lylme_spage/internal/service"
	"lylme_spage/internal/site"
)

// Deps 汇总 REST 层所需的全部 service 依赖。处理器只依赖 service，
// 不在此层重复任何业务规则。
type Deps struct {
	Keys      *service.APIKeyService
	Group     *service.GroupService
	Link      *service.LinkService
	Tag       *service.TagService
	Sou       *service.SouService
	Pwd       *service.PwdService
	Config    *service.ConfigService
	Apply     *service.ApplyService
	Theme     *service.ThemeService
	Dashboard *service.DashboardService
	Files     *service.FileCleanupService

	Cfg        *config.Config
	AppVersion string
}

// NewDeps 基于共享数据层构建全部 service 依赖。
func NewDeps(store *site.Store, cfg *config.Config, appVersion string) *Deps {
	return &Deps{
		Keys:       service.NewAPIKeyService(store),
		Group:      service.NewGroupService(store),
		Link:       service.NewLinkService(store, cfg),
		Tag:        service.NewTagService(store),
		Sou:        service.NewSouService(store),
		Pwd:        service.NewPwdService(store),
		Config:     service.NewConfigService(store),
		Apply:      service.NewApplyService(store),
		Theme:      service.NewThemeService(store),
		Dashboard:  service.NewDashboardService(store),
		Files:      service.NewFileCleanupService(store, cfg),
		Cfg:        cfg,
		AppVersion: appVersion,
	}
}

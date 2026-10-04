package api

import (
	"time"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/service"
)

// sensitiveConfigKeys GET /settings 敏感键显式黑名单（键名以 lylme_config.k 实际值为准）。
// 出参只做黑名单剔除，其余配置项（含 wxplus 推送密钥、license 授权码）照常返回。
var sensitiveConfigKeys = map[string]struct{}{
	"admin_pwd": {}, // 管理员密码哈希
	"api_key":   {}, // API/MCP 接口密钥明文
}

// maskedKeyHead sk- 前缀后保留的明文 hex 位数，其余以 … 代替
const maskedKeyHead = 4

// bindSettingsJSON 绑定并校验 JSON 请求体；失败时直接写入 400 响应。
func bindSettingsJSON(c *gin.Context, obj any) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		FailBadRequest(c, "请求体格式错误或参数不合法: "+err.Error())
		return false
	}
	return true
}

// maskAPIKey 返回密钥掩码（形如 sk-1a2b…），不暴露完整明文。
func maskAPIKey(key string) string {
	if len(key) < len("sk-")+maskedKeyHead {
		return "sk-…"
	}
	return key[:len("sk-")+maskedKeyHead] + "…"
}

// formatCreatedAt 将时间格式化为 RFC3339；零值返回空串。
func formatCreatedAt(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// RegisterSettingsRoutes 在 v1 组（/api/v1，调用方已挂 Bearer 密钥鉴权中间件）下
// 注册网站设置、关于页、管理员账号、系统安全、微信推送、授权码、API 密钥相关端点。
func RegisterSettingsRoutes(g gin.IRouter, d *Deps) {
	g.GET("/settings", d.settingsIndex)
	g.PUT("/settings", d.updateSettings)

	g.GET("/settings/about", d.settingsAboutShow)
	g.PUT("/settings/about", d.settingsAboutUpdate)
	g.POST("/settings/about/reset", d.settingsAboutReset)

	g.PUT("/settings/account", d.settingsAccountUpdate)

	g.GET("/settings/system", d.settingsSystemShow)
	g.PUT("/settings/system", d.settingsSystemUpdate)

	g.GET("/wxplus", d.wxplusShow)
	g.PUT("/wxplus", d.wxplusUpdate)

	g.GET("/license", d.licenseShow)
	g.PUT("/license", d.licenseUpdate)

	g.GET("/apikey", d.apikeyShow)
	g.POST("/apikey/regenerate", d.apikeyRegenerate)
}

// settingsIndex 读取全部配置项（敏感键黑名单剔除）
func (d *Deps) settingsIndex(c *gin.Context) {
	conf := d.Config.AllConfig()
	for k := range sensitiveConfigKeys {
		delete(conf, k)
	}
	OK(c, conf)
}

// siteSettingsRequest 网站基本设置批量更新入参（空串为合法值，表示清空该项）
type siteSettingsRequest struct {
	Title         string `json:"title"`
	Logo          string `json:"logo"`
	Background    string `json:"background"`
	WapBackground string `json:"wap_background"`
	Keywords      string `json:"keywords"`
	Description   string `json:"description"`
	Mode          string `json:"mode"`
	Snapshot      string `json:"snapshot"`
	Copyright     string `json:"copyright"`
	ICP           string `json:"icp"`
	WzTJ          string `json:"wztj"`
	CDNPublic     string `json:"cdnpublic"`
	// Yan 仅恰好为 "true" 时落 "true"，其余一律落 "false"（归一逻辑在 service）
	Yan string `json:"yan"`
}

// updateSettings 全字段批量更新网站设置
func (d *Deps) updateSettings(c *gin.Context) {
	var req siteSettingsRequest
	if !bindSettingsJSON(c, &req) {
		return
	}
	if err := d.Config.UpdateSiteSettings(service.SiteSettingsInput{
		Title:         req.Title,
		Logo:          req.Logo,
		Background:    req.Background,
		WapBackground: req.WapBackground,
		Keywords:      req.Keywords,
		Description:   req.Description,
		Mode:          req.Mode,
		Snapshot:      req.Snapshot,
		Copyright:     req.Copyright,
		ICP:           req.ICP,
		WzTJ:          req.WzTJ,
		CDNPublic:     req.CDNPublic,
		Yan:           req.Yan,
	}); err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"updated": true})
}

// settingsAboutShow 读取关于页内容（未配置时为内置默认内容）
func (d *Deps) settingsAboutShow(c *gin.Context) {
	OK(c, gin.H{"content": d.Config.AboutContent()})
}

// aboutUpdateRequest 关于页更新入参
type aboutUpdateRequest struct {
	Content string `json:"content" binding:"required"`
}

// settingsAboutUpdate 更新关于页内容
func (d *Deps) settingsAboutUpdate(c *gin.Context) {
	var req aboutUpdateRequest
	if !bindSettingsJSON(c, &req) {
		return
	}
	if err := d.Config.UpdateAboutContent(req.Content); err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"updated": true})
}

// settingsAboutReset 恢复关于页默认内容，并返回恢复后的内容
func (d *Deps) settingsAboutReset(c *gin.Context) {
	if err := d.Config.ResetAboutContent(); err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"content": service.DefaultAboutContent})
}

// accountUpdateRequest 管理员账号修改入参。
// username 可空（表示不改用户名）；new_password/confirm_password 可空（表示不改密码）；
// 具体业务分支（无更改/不一致/旧密码错误）由 service 判定。
type accountUpdateRequest struct {
	Username        string `json:"username"`
	OldPassword     string `json:"old_password" binding:"required"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

// settingsAccountUpdate 修改管理员用户名/密码
func (d *Deps) settingsAccountUpdate(c *gin.Context) {
	var req accountUpdateRequest
	if !bindSettingsJSON(c, &req) {
		return
	}
	res, err := d.Config.UpdateAccount(service.AccountInput{
		Username:        req.Username,
		OldPassword:     req.OldPassword,
		NewPassword:     req.NewPassword,
		ConfirmPassword: req.ConfirmPassword,
	})
	if err != nil {
		WriteServiceError(c, err)
		return
	}
	// 出于安全考虑不回显新密码明文
	OK(c, gin.H{
		"changed_user": res.ChangedUser,
		"changed_pwd":  res.ChangedPwd,
		"username":     res.Username,
	})
}

// settingsSystemShow 读取系统安全设置（未配置时 admin_path 取应用配置默认值，debug 默认 "0"）
func (d *Deps) settingsSystemShow(c *gin.Context) {
	conf := d.Config.AllConfig()
	adminPath := conf["admin_path"]
	if adminPath == "" {
		adminPath = d.Cfg.AdminPath
		if adminPath == "" {
			adminPath = "admin"
		}
	}
	debug := conf["debug"]
	if debug != "1" {
		debug = "0"
	}
	OK(c, gin.H{"admin_path": adminPath, "debug": debug})
}

// systemUpdateRequest 系统安全设置更新入参
type systemUpdateRequest struct {
	AdminPath string `json:"admin_path" binding:"required"`
	Debug     string `json:"debug" binding:"required,oneof=0 1"`
}

// settingsSystemUpdate 更新后台目录与调试模式
func (d *Deps) settingsSystemUpdate(c *gin.Context) {
	var req systemUpdateRequest
	if !bindSettingsJSON(c, &req) {
		return
	}
	if err := d.Config.UpdateSystem(service.SystemInput{
		AdminPath: req.AdminPath,
		Debug:     req.Debug,
	}); err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"updated": true})
}

// wxplusShow 读取微信推送配置
func (d *Deps) wxplusShow(c *gin.Context) {
	cfg := d.Config.WxPlus()
	OK(c, gin.H{"key": cfg.Key, "time": cfg.Time})
}

// wxplusUpdateRequest 微信推送配置更新入参（空 key 表示关闭推送）
type wxplusUpdateRequest struct {
	Key  string `json:"key"`
	Time string `json:"time"`
}

// wxplusUpdate 更新微信推送密钥与每日推送时间
func (d *Deps) wxplusUpdate(c *gin.Context) {
	var req wxplusUpdateRequest
	if !bindSettingsJSON(c, &req) {
		return
	}
	if err := d.Config.UpdateWxPlus(req.Key, req.Time); err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"updated": true})
}

// licenseShow 读取网站授权配置（hostmode 仅 "1"/"2"，缺省归一为 "2"）
func (d *Deps) licenseShow(c *gin.Context) {
	cfg := d.Config.License()
	OK(c, gin.H{"code": cfg.Code, "hostmode": cfg.HostMode})
}

// licenseUpdateRequest 授权配置更新入参
type licenseUpdateRequest struct {
	Code     string `json:"code"`
	HostMode string `json:"hostmode" binding:"required,oneof=1 2"`
}

// licenseUpdate 更新授权码与域名获取方式
func (d *Deps) licenseUpdate(c *gin.Context) {
	var req licenseUpdateRequest
	if !bindSettingsJSON(c, &req) {
		return
	}
	if err := d.Config.UpdateLicense(req.Code, req.HostMode); err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{"updated": true})
}

// apikeyShow 仅返回密钥是否存在、创建时间与掩码，不回读完整明文
func (d *Deps) apikeyShow(c *gin.Context) {
	info := d.Keys.Get()
	masked := ""
	if info.Exists {
		masked = maskAPIKey(info.Key)
	}
	OK(c, gin.H{
		"exists":     info.Exists,
		"created_at": formatCreatedAt(info.CreatedAt),
		"masked":     masked,
	})
}

// apikeyRegenerate 重新生成密钥。
// 注意：旧密钥在重新生成成功的瞬间立即失效，所有使用旧密钥的客户端必须更换新密钥。
// 本端点是唯一会返回完整明文密钥的时机，响应内容不会再次出现于其他接口。
func (d *Deps) apikeyRegenerate(c *gin.Context) {
	key, err := d.Keys.Regenerate()
	if err != nil {
		WriteServiceError(c, err)
		return
	}
	OK(c, gin.H{
		"key":        key,
		"created_at": formatCreatedAt(d.Keys.Get().CreatedAt),
	})
}

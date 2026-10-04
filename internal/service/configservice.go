package service

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"

	"lylme_spage/internal/site"
)

// adminPasswordSalt 管理员密码盐值（与原版一致：md5('lylme'.明文)）
const adminPasswordSalt = "lylme"

// DefaultAboutContent 关于页面默认内容（原 admin.defaultAboutContent）
const DefaultAboutContent = "<h3>关于本站</h3>\r\n<p>欢迎访问本站，这是一个开源的网址导航与搜索入口项目，旨在提供简洁、轻量的上网起始页体验。</p>\r\n<p>如果您喜欢本站，可将本页添加到收藏夹（快捷键 <code>Ctrl+D</code>）方便下次访问；也可将其设为浏览器主页。感谢您的支持。</p>\r\n<hr>\r\n<h3>隐私说明</h3>\r\n<p>本项目为开源程序，默认仅提供网址导航与跳转功能，链接直接指向目标地址，不对访问链接做二次中转。</p>\r\n<p>程序本身不强制收集用户隐私信息（如点击记录、访问记录、搜索记录等）。但您所访问的具体实例由部署者自行搭建与维护，其实际的数据收集与处理方式以该实例部署者的隐私政策为准。建议您在使用前了解所在实例的相关说明，并注意保护个人信息。</p>\r\n<hr>\r\n<h3>申请收录</h3>\r\n<p>如需将您的网站加入导航，请点<a href=\"../apply\" target=\"_blank\">这里</a>提交申请。</p>\r\n<hr>\r\n<h3>联系我们</h3>\r\n<p>若您在使用本站时遇到了以下问题，欢迎与我们联系：</p>\r\n<ul>\r\n<li>图标缺失</li>\r\n<li>目标网站无法打开</li>\r\n<li>描述错误</li>\r\n<li>网站违规</li>\r\n<li>收录加急处理</li>\r\n<li>链接删除</li>\r\n</ul>\r\n<h5>联系方式</h5>\r\n<ul>\r\n<li>邮箱：<a href=\"mailto:#\">未配置</a></li>\r\n</ul>\r\n<h5>联系说明</h5>\r\n<p>为了您的问题能快速被处理，建议在邮件主题中添加【反馈】【投诉】【建议】【友链】等标识。</p>"

// 配置域哨兵错误。service 只返回哨兵错误与强类型结果，不渲染任何 HTML。
var (
	// ErrWrongPassword 旧密码校验失败（认证类错误，不归属 ErrInvalidInput）
	ErrWrongPassword = errors.New("config: 当前密码错误")

	// ErrPasswordMismatch 两次新密码不一致；归属于 ErrInvalidInput
	ErrPasswordMismatch = errors.Join(ErrInvalidInput, errors.New("config: 两次新密码不一致"))

	// ErrNoChange 用户名与新密码均为空，未做出更改
	ErrNoChange = errors.New("config: 未做出更改")

	// ErrAdminPathInvalid 后台目录不合法；归属于 ErrInvalidInput
	ErrAdminPathInvalid = errors.Join(ErrInvalidInput, errors.New("config: 后台目录不合法"))

	// ErrDebugInvalid 调试模式取值非法；归属于 ErrInvalidInput
	ErrDebugInvalid = errors.Join(ErrInvalidInput, errors.New("config: 调试模式参数无效"))
)

// adminPathRe 后台目录合法字符（与原版一致：字母、数字、下划线、中划线）
var adminPathRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// AdminPasswordHash 管理员密码哈希：md5("lylme"+明文)。
// 全项目唯一的管理员密码哈希实现，登录与改密共用。
func AdminPasswordHash(plain string) string {
	sum := md5.Sum([]byte(adminPasswordSalt + plain))
	return hex.EncodeToString(sum[:])
}

// SiteSettingsInput 网站基本设置入参（字段名为 service 契约，键名映射在 service 内定义）
type SiteSettingsInput struct {
	Title         string
	Logo          string
	Background    string
	WapBackground string
	Keywords      string
	Description   string
	Mode          string
	Snapshot      string
	Copyright     string
	ICP           string
	WzTJ          string
	CDNPublic     string
	// Yan 仅当提交值恰好为 "true" 时落 "true"，其余一律落 "false"
	Yan string
}

// AccountInput 管理员账号修改入参（Username 的 trim 由 service 完成；密码按原始值）
type AccountInput struct {
	Username        string
	OldPassword     string
	NewPassword     string
	ConfirmPassword string
}

// AccountResult 账号修改结果
type AccountResult struct {
	ChangedUser bool
	ChangedPwd  bool
	// Username/Password 落库后的用户名/新密码明文，供处理器回显（与原版 alert 一致）
	Username string
	Password string
}

// SystemInput 系统安全设置入参（AdminPath 的 trim 由 service 完成）
type SystemInput struct {
	AdminPath string
	Debug     string
}

// ConfigService 配置域业务规则：网站设置、关于页、管理员账号、系统安全
type ConfigService struct {
	store *site.Store
}

// NewConfigService 创建配置服务
func NewConfigService(store *site.Store) *ConfigService {
	return &ConfigService{store: store}
}

// siteSettingDef 网站设置项定义：配置键名、配置描述与入参取值
type siteSettingDef struct {
	key  string
	desc string
	val  string
}

// siteSettings 将入参映射为原版配置键（wap_background ← 表单 wapbackground）
func siteSettings(in SiteSettingsInput) []siteSettingDef {
	return []siteSettingDef{
		{"title", "网站名称", in.Title},
		{"logo", "网站LOGO", in.Logo},
		{"background", "背景图片", in.Background},
		{"wap_background", "手机背景图片", in.WapBackground},
		{"keywords", "网站关键词", in.Keywords},
		{"description", "网站描述", in.Description},
		{"mode", "网站运行模式", in.Mode},
		{"snapshot", "详情页快照API", in.Snapshot},
		{"copyright", "底部版权", in.Copyright},
		{"icp", "ICP备案号", in.ICP},
		{"wztj", "自定义footer", in.WzTJ},
		{"cdnpublic", "CDN地址", in.CDNPublic},
	}
}

// UpdateSiteSettings 批量写入原版 12 个网站设置项，以及：
// yan（仅提交 "true" 落 "true"，否则 "false"）、tq（固定 "false"）。
// 单项写入失败不阻断其余项（与原版一致：错误仅记录、最终仍提示成功），
// 全部失败经 errors.Join 聚合返回；无论是否有失败项，最后均重载配置缓存。
func (s *ConfigService) UpdateSiteSettings(in SiteSettingsInput) error {
	var errs []error
	for _, it := range siteSettings(in) {
		if err := s.store.Set(it.key, it.val, it.desc); err != nil {
			errs = append(errs, err)
		}
	}
	yan := "false"
	if in.Yan == "true" {
		yan = "true"
	}
	if err := s.store.Set("yan", yan, ""); err != nil {
		errs = append(errs, err)
	}
	if err := s.store.Set("tq", "false", ""); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// AboutContent 读取关于页内容；未配置时返回默认内容
func (s *ConfigService) AboutContent() string {
	return s.store.Get("about_content", DefaultAboutContent)
}

// UpdateAboutContent 更新关于页内容（按原始提交值落库）
func (s *ConfigService) UpdateAboutContent(html string) error {
	return s.store.Set("about_content", html, "")
}

// ResetAboutContent 恢复关于页默认内容
func (s *ConfigService) ResetAboutContent() error {
	return s.store.Set("about_content", DefaultAboutContent, "")
}

// UpdateAccount 修改管理员账号，保持原版四个分支语义（旧密码校验在最前）：
//   - 新密码为空：用户名为空 → ErrNoChange；仅改名 → ChangedUser；
//   - 用户名为空且两次密码一致：仅改密 → ChangedPwd；
//   - 两次密码一致（用户名非空）：两者都改；
//   - 其余：ErrPasswordMismatch。
func (s *ConfigService) UpdateAccount(in AccountInput) (AccountResult, error) {
	if AdminPasswordHash(in.OldPassword) != s.store.Get("admin_pwd", "") {
		return AccountResult{}, ErrWrongPassword
	}
	user := strings.TrimSpace(in.Username)
	switch {
	case in.NewPassword == "":
		if user == "" {
			return AccountResult{}, ErrNoChange
		}
		if err := s.store.Set("admin_user", user, ""); err != nil {
			return AccountResult{}, err
		}
		return AccountResult{ChangedUser: true, Username: user}, nil
	case user == "" && in.NewPassword == in.ConfirmPassword:
		if err := s.store.Set("admin_pwd", AdminPasswordHash(in.NewPassword), ""); err != nil {
			return AccountResult{}, err
		}
		return AccountResult{ChangedPwd: true, Password: in.NewPassword}, nil
	case in.NewPassword == in.ConfirmPassword:
		if err := s.store.Set("admin_user", user, ""); err != nil {
			return AccountResult{}, err
		}
		if err := s.store.Set("admin_pwd", AdminPasswordHash(in.NewPassword), ""); err != nil {
			return AccountResult{}, err
		}
		return AccountResult{ChangedUser: true, ChangedPwd: true, Username: user, Password: in.NewPassword}, nil
	default:
		return AccountResult{}, ErrPasswordMismatch
	}
}

// UpdateSystem 修改系统安全设置：
// admin_path 非空且匹配 ^[A-Za-z0-9_-]+$；debug 仅接受 "0"/"1"。
// 校验全部通过后才落库（任一项非法则不写任何配置）。
func (s *ConfigService) UpdateSystem(in SystemInput) error {
	adminPath := strings.TrimSpace(in.AdminPath)
	if adminPath == "" || !adminPathRe.MatchString(adminPath) {
		return ErrAdminPathInvalid
	}
	if in.Debug != "0" && in.Debug != "1" {
		return ErrDebugInvalid
	}
	if err := s.store.Set("admin_path", adminPath, "后台目录"); err != nil {
		return err
	}
	if err := s.store.Set("debug", in.Debug, "调试模式"); err != nil {
		return err
	}
	return nil
}

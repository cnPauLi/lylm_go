package admin

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

const defaultAboutContent = "<h3>关于本站</h3>\r\n<p>欢迎访问本站，这是一个开源的网址导航与搜索入口项目，旨在提供简洁、轻量的上网起始页体验。</p>\r\n<p>如果您喜欢本站，可将本页添加到收藏夹（快捷键 <code>Ctrl+D</code>）方便下次访问；也可将其设为浏览器主页。感谢您的支持。</p>\r\n<hr>\r\n<h3>隐私说明</h3>\r\n<p>本项目为开源程序，默认仅提供网址导航与跳转功能，链接直接指向目标地址，不对访问链接做二次中转。</p>\r\n<p>程序本身不强制收集用户隐私信息（如点击记录、访问记录、搜索记录等）。但您所访问的具体实例由部署者自行搭建与维护，其实际的数据收集与处理方式以该实例部署者的隐私政策为准。建议您在使用前了解所在实例的相关说明，并注意保护个人信息。</p>\r\n<hr>\r\n<h3>申请收录</h3>\r\n<p>如需将您的网站加入导航，请点<a href=\"../apply\" target=\"_blank\">这里</a>提交申请。</p>\r\n<hr>\r\n<h3>联系我们</h3>\r\n<p>若您在使用本站时遇到了以下问题，欢迎与我们联系：</p>\r\n<ul>\r\n<li>图标缺失</li>\r\n<li>目标网站无法打开</li>\r\n<li>描述错误</li>\r\n<li>网站违规</li>\r\n<li>收录加急处理</li>\r\n<li>链接删除</li>\r\n</ul>\r\n<h5>联系方式</h5>\r\n<ul>\r\n<li>邮箱：<a href=\"mailto:#\">未配置</a></li>\r\n</ul>\r\n<h5>联系说明</h5>\r\n<p>为了您的问题能快速被处理，建议在邮件主题中添加【反馈】【投诉】【建议】【友链】等标识。</p>"

// setPage 网站基本设置模板数据
type setPage struct {
	Mode2 bool
}

// SetPage 处理 set.php（GET 渲染 / POST ?set=save 保存）
func (h *Handler) SetPage(c *gin.Context) {
	if c.Request.Method == http.MethodPost && c.Query("set") == "save" {
		h.saveSettings(c)
		return
	}
	data := h.newData(c, "网站设置", "set.php", "config")
	data.Page = &setPage{Mode2: h.Store.Get("mode", "1") == "2"}
	h.render(c, "set.html", data)
}

func (h *Handler) saveSettings(c *gin.Context) {
	type item struct{ key, desc string }
	items := []item{
		{"title", "网站名称"},
		{"logo", "网站LOGO"},
		{"background", "背景图片"},
		{"wap_background", "手机背景图片"},
		{"keywords", "网站关键词"},
		{"description", "网站描述"},
		{"mode", "网站运行模式"},
		{"snapshot", "详情页快照API"},
		{"copyright", "底部版权"},
		{"icp", "ICP备案号"},
		{"wztj", "自定义footer"},
		{"cdnpublic", "CDN地址"},
	}
	formKey := map[string]string{
		"title": "title", "logo": "logo", "background": "background",
		"wap_background": "wapbackground", "keywords": "keywords",
		"description": "description", "mode": "mode", "snapshot": "snapshot",
		"copyright": "copyright", "icp": "icp", "wztj": "wztj", "cdnpublic": "cdnpublic",
	}
	for _, it := range items {
		if err := h.Store.Set(it.key, c.PostForm(formKey[it.key]), it.desc); err != nil {
			h.Log.Errorf("保存配置 %s 失败: %v", it.key, err)
		}
	}
	yan := "false"
	if c.PostForm("yan") == "true" {
		yan = "true"
	}
	if err := h.Store.Set("yan", yan, ""); err != nil {
		h.Log.Errorf("保存配置 yan 失败: %v", err)
	}
	if err := h.Store.Set("tq", "false", ""); err != nil {
		h.Log.Errorf("保存配置 tq 失败: %v", err)
	}
	h.Store.Reload()
	h.renderHTML(c, alert("成功", "网站设置修改成功！", "./set.php"))
}

// AboutPage 处理 about.php
func (h *Handler) AboutPage(c *gin.Context) {
	switch {
	case c.Request.Method == http.MethodPost && c.Query("set") == "conf_submit":
		if err := h.Store.Set("about_content", c.PostForm("about"), ""); err != nil {
			h.Log.Errorf("保存关于页失败: %v", err)
		}
		h.renderHTML(c, alert("成功", "修改成功！", "./about.php"))
		return
	case c.Query("set") == "default":
		if err := h.Store.Set("about_content", defaultAboutContent, ""); err != nil {
			h.Log.Errorf("恢复关于页默认失败: %v", err)
		}
		h.renderHTML(c, alert("成功", "恢复默认成功！", "./about.php"))
		return
	}
	data := h.newData(c, "关于页面设置", "about.php", "config")
	h.render(c, "about.html", data)
}

// userPage 账户/系统安全模板数据
type userPage struct {
	CurrentDir string
	DebugOn    bool
}

// UserPage 处理 user.php
func (h *Handler) UserPage(c *gin.Context) {
	set := c.Query("set")
	if c.Request.Method == http.MethodPost {
		switch set {
		case "syssave":
			h.userSysSave(c)
			return
		case "save":
			h.userSave(c)
			return
		}
	}
	data := h.newData(c, "系统安全", "user.php", "config")
	data.Page = &userPage{CurrentDir: h.AdminPath(), DebugOn: h.Store.Get("debug", "0") == "1"}
	h.render(c, "user.html", data)
}

func (h *Handler) userSave(c *gin.Context) {
	user := strings.TrimSpace(c.PostForm("new-usernameuser"))
	oldpwd := c.PostForm("oldpwd")
	newpwd := c.PostForm("newpwd")
	repwd := c.PostForm("confirmpwd")

	if md5hex("lylme"+oldpwd) != h.Store.Get("admin_pwd", "") {
		h.renderHTML(c, alert("错误", "当前密码错误！", ""))
		return
	}
	switch {
	case newpwd == "":
		if user == "" {
			h.renderHTML(c, alert("提示", "未做出更改", ""))
			return
		}
		if err := h.Store.Set("admin_user", user, ""); err != nil {
			h.Log.Errorf("修改用户名失败: %v", err)
		}
		h.renderHTML(c, alert("成功", "用户名修改成功！<br>新用户名："+user+"<br>请牢记，将重新登录！", "./"))
	case user == "" && newpwd == repwd:
		if err := h.Store.Set("admin_pwd", md5hex("lylme"+newpwd), ""); err != nil {
			h.Log.Errorf("修改密码失败: %v", err)
		}
		h.renderHTML(c, alert("成功", "密码修改成功！<br>新密码："+newpwd+"<br>请牢记，将重新登录！", "./"))
	case newpwd == repwd:
		if err := h.Store.Set("admin_user", user, ""); err != nil {
			h.Log.Errorf("修改用户名失败: %v", err)
		}
		if err := h.Store.Set("admin_pwd", md5hex("lylme"+newpwd), ""); err != nil {
			h.Log.Errorf("修改密码失败: %v", err)
		}
		h.renderHTML(c, alert("成功", "修改成功！<br>新用户名："+user+"<br>新密码："+newpwd+"<br>请牢记，将重新登录！", "./"))
	default:
		h.renderHTML(c, alert("错误", "两次新密码不一致！", ""))
	}
}

var adminPathRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func (h *Handler) userSysSave(c *gin.Context) {
	newAdminPath := strings.TrimSpace(c.PostForm("admin_path"))
	newDebug := c.PostForm("debug")
	if newAdminPath == "" || !adminPathRe.MatchString(newAdminPath) {
		h.renderHTML(c, alert("错误", "后台目录不合法！只能使用字母、数字、下划线、中划线", ""))
		return
	}
	if newDebug != "0" && newDebug != "1" {
		h.renderHTML(c, alert("错误", "调试模式参数无效！", ""))
		return
	}
	// 说明：原版通过改写 include/common.php 中的 ADMIN_PATH / DEBUG 常量实现，
	// Go 版无法改写 PHP 源码，改为持久化为数据库配置项 admin_path / debug，
	// 于下次启动注册路由时生效（见 Handler.AdminPath）。
	if err := h.Store.Set("admin_path", newAdminPath, "后台目录"); err != nil {
		h.Log.Errorf("保存 admin_path 失败: %v", err)
	}
	if err := h.Store.Set("debug", newDebug, "调试模式"); err != nil {
		h.Log.Errorf("保存 debug 失败: %v", err)
	}
	msg := "系统设置保存成功！"
	if newAdminPath != h.AdminPath() {
		msg += "\\n后台目录已修改为 " + newAdminPath + "，重启服务后请通过新地址访问后台（" + newAdminPath + "/）"
	}
	h.renderHTML(c, alert("成功", msg, "./user.php"))
}

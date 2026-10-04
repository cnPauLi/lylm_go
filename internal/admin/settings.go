package admin

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/service"
)

// setPage 网站基本设置模板数据
type setPage struct {
	Mode2 bool
}

// SetPage 处理 set（GET 渲染 / POST ?set=save 保存）
func (h *Handler) SetPage(c *gin.Context) {
	if c.Request.Method == http.MethodPost && c.Query("set") == "save" {
		in := service.SiteSettingsInput{
			Title:         c.PostForm("title"),
			Logo:          c.PostForm("logo"),
			Background:    c.PostForm("background"),
			WapBackground: c.PostForm("wapbackground"),
			Keywords:      c.PostForm("keywords"),
			Description:   c.PostForm("description"),
			Mode:          c.PostForm("mode"),
			Snapshot:      c.PostForm("snapshot"),
			Copyright:     c.PostForm("copyright"),
			ICP:           c.PostForm("icp"),
			WzTJ:          c.PostForm("wztj"),
			CDNPublic:     c.PostForm("cdnpublic"),
			Yan:           c.PostForm("yan"),
		}
		// 单项失败不影响成功提示（与原版一致，错误仅记录日志）
		if err := h.ConfigSvc.UpdateSiteSettings(in); err != nil {
			h.Log.Errorf("保存网站设置失败: %v", err)
		}
		h.renderHTML(c, alert("成功", "网站设置修改成功！", "./set"))
		return
	}
	data := h.newData(c, "网站设置", "set", "config")
	data.Page = &setPage{Mode2: h.Store.Get("mode", "1") == "2"}
	h.render(c, "set.html", data)
}

// AboutPage 处理 about
func (h *Handler) AboutPage(c *gin.Context) {
	switch {
	case c.Request.Method == http.MethodPost && c.Query("set") == "conf_submit":
		if err := h.ConfigSvc.UpdateAboutContent(c.PostForm("about")); err != nil {
			h.Log.Errorf("保存关于页失败: %v", err)
		}
		h.renderHTML(c, alert("成功", "修改成功！", "./about"))
		return
	case c.Query("set") == "default":
		if err := h.ConfigSvc.ResetAboutContent(); err != nil {
			h.Log.Errorf("恢复关于页默认失败: %v", err)
		}
		h.renderHTML(c, alert("成功", "恢复默认成功！", "./about"))
		return
	}
	data := h.newData(c, "关于页面设置", "about", "config")
	h.render(c, "about.html", data)
}

// userPage 账户/系统安全模板数据
type userPage struct {
	CurrentDir string
	DebugOn    bool
}

// UserPage 处理 user
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
	data := h.newData(c, "系统安全", "user", "config")
	data.Page = &userPage{CurrentDir: h.AdminPath(), DebugOn: h.Store.Get("debug", "0") == "1"}
	h.render(c, "user.html", data)
}

func (h *Handler) userSave(c *gin.Context) {
	res, err := h.ConfigSvc.UpdateAccount(service.AccountInput{
		Username:        c.PostForm("new-usernameuser"),
		OldPassword:     c.PostForm("oldpwd"),
		NewPassword:     c.PostForm("newpwd"),
		ConfirmPassword: c.PostForm("confirmpwd"),
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrWrongPassword):
			h.renderHTML(c, alert("错误", "当前密码错误！", ""))
			return
		case errors.Is(err, service.ErrPasswordMismatch):
			h.renderHTML(c, alert("错误", "两次新密码不一致！", ""))
			return
		case errors.Is(err, service.ErrNoChange):
			h.renderHTML(c, alert("提示", "未做出更改", ""))
			return
		default:
			h.Log.Errorf("修改管理员账号失败: %v", err)
			c.String(http.StatusInternalServerError, "保存失败: %v", err)
			return
		}
	}
	switch {
	case res.ChangedUser && res.ChangedPwd:
		h.renderHTML(c, alert("成功", "修改成功！<br>新用户名："+res.Username+"<br>新密码："+res.Password+"<br>请牢记，将重新登录！", "./"))
	case res.ChangedUser:
		h.renderHTML(c, alert("成功", "用户名修改成功！<br>新用户名："+res.Username+"<br>请牢记，将重新登录！", "./"))
	default:
		h.renderHTML(c, alert("成功", "密码修改成功！<br>新密码："+res.Password+"<br>请牢记，将重新登录！", "./"))
	}
}

func (h *Handler) userSysSave(c *gin.Context) {
	newAdminPath := strings.TrimSpace(c.PostForm("admin_path"))
	newDebug := c.PostForm("debug")
	err := h.ConfigSvc.UpdateSystem(service.SystemInput{AdminPath: newAdminPath, Debug: newDebug})
	switch {
	case errors.Is(err, service.ErrAdminPathInvalid):
		h.renderHTML(c, alert("错误", "后台目录不合法！只能使用字母、数字、下划线、中划线", ""))
		return
	case errors.Is(err, service.ErrDebugInvalid):
		h.renderHTML(c, alert("错误", "调试模式参数无效！", ""))
		return
	case err != nil:
		h.Log.Errorf("保存系统设置失败: %v", err)
		c.String(http.StatusInternalServerError, "保存失败: %v", err)
		return
	}
	// 说明：原版通过改写 include/common 中的 ADMIN_PATH / DEBUG 常量实现，
	// Go 版无法改写 PHP 源码，改为持久化为数据库配置项 admin_path / debug，
	// 于下次启动注册路由时生效（见 Handler.AdminPath）。
	msg := "系统设置保存成功！"
	if newAdminPath != h.AdminPath() {
		msg += "\\n后台目录已修改为 " + newAdminPath + "，重启服务后请通过新地址访问后台（" + newAdminPath + "/）"
	}
	h.renderHTML(c, alert("成功", msg, "./user"))
}

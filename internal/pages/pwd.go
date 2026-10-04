package pages

import (
	"html"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/fetch"
	"lylme_spage/internal/logger"
	"lylme_spage/internal/models"
	"lylme_spage/internal/site"
	"lylme_spage/internal/webutil"
)

// PasswordHandler 实现 /include/go.php：跳转、访问密码登录与注销
func PasswordHandler(store *site.Store, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		referer := c.Request.Referer()
		if referer == "" {
			referer = "/"
		}

		// 外链跳转：?url=xxx
		if target := c.Query("url"); target != "" {
			if err := fetch.ValidateURL(target); err == nil {
				c.Redirect(http.StatusFound, target)
				return
			}
			c.Redirect(http.StatusFound, "/")
			return
		}

		if c.Request.Method == http.MethodPost {
			// 注销登录
			if c.PostForm("exit") == "exit" {
				if !webutil.CSRFVerify(c) {
					writeAlert(c, referer, "安全验证失败，请重试")
					return
				}
				webutil.ClearUnlocked(c)
				webutil.SetPwdLogged(c, false)
				c.Redirect(http.StatusFound, referer)
				return
			}

			// 提交访问密码
			if pass := strings.TrimSpace(c.PostForm("pass")); pass != "" {
				if !webutil.CSRFVerify(c) {
					writeAlert(c, referer, "安全验证失败，请重试")
					return
				}
				var pwds []models.Pwd
				if err := store.DB().Where("pwd_key = ?", pass).Find(&pwds).Error; err != nil {
					log.Errorf("查询访问密码失败: %v", err)
				}
				ids := make([]int, 0, len(pwds))
				for _, p := range pwds {
					ids = append(ids, p.PwdID)
				}
				if len(ids) == 0 {
					writeAlert(c, referer, "密码错误！")
					return
				}
				webutil.SetUnlockedIDs(c, ids)
				webutil.SetPwdLogged(c, true)
				c.Redirect(http.StatusFound, "/")
				return
			}
		}

		c.Redirect(http.StatusFound, "/")
	}
}

// pwdPage 访问密码页渲染数据
type pwdPage struct {
	PwdEnabled bool
	IsLoggedIn bool
	Unlocked   []string
}

// Pwd 访问密码页 /pwd/ 与 /pwd
func Pwd(store *site.Store, engine *Engine, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var count int64
		if err := store.DB().Model(&models.Pwd{}).Count(&count).Error; err != nil {
			log.Errorf("查询访问密码失败: %v", err)
		}

		page := &pwdPage{
			PwdEnabled: count != 0,
			IsLoggedIn: webutil.IsPwdLogged(c),
		}
		if page.IsLoggedIn {
			for _, id := range webutil.UnlockedIDs(c) {
				page.Unlocked = append(page.Unlocked, strconv.Itoa(id))
			}
		}

		data := engine.DataFor("", page.IsLoggedIn, webutil.IsMobile(c.Request))
		data.CSRFToken = webutil.CSRFToken(c)
		data.BackgroundImg = relativeBackground(data.BackgroundImg)
		data.Page = page

		c.Header("Content-Type", "text/html; charset=utf-8")
		if err := engine.Render(c.Writer, "pwd.html", data); err != nil {
			log.Errorf("访问密码页渲染失败: %v", err)
			c.String(http.StatusInternalServerError, "页面渲染失败")
		}
	}
}

// writeAlert 输出与原始 PHP 一致的 alert 跳转脚本
func writeAlert(c *gin.Context, referer, msg string) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	_, _ = c.Writer.WriteString(`<script>alert("` + msg + `");window.location.href="` + html.EscapeString(referer) + `"</script>`)
}

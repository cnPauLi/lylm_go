package pages

import (
	"html"
	"net/http"
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

// writeAlert 输出与原始 PHP 一致的 alert 跳转脚本
func writeAlert(c *gin.Context, referer, msg string) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	_, _ = c.Writer.WriteString(`<script>alert("` + msg + `");window.location.href="` + html.EscapeString(referer) + `"</script>`)
}

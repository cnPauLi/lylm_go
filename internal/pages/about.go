package pages

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/logger"
	"lylme_spage/internal/site"
	"lylme_spage/internal/webutil"
)

// About 关于页 /about/ 与 /about
func About(store *site.Store, engine *Engine, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		data := engine.DataFor("", false, webutil.IsMobile(c.Request))
		data.CSRFToken = webutil.CSRFToken(c)

		tags, err := store.Tags()
		if err != nil {
			log.Errorf("查询导航菜单失败: %v", err)
		}
		data.Tags = tags

		c.Header("Content-Type", "text/html; charset=utf-8")
		if err := engine.Render(c.Writer, "about.html", data); err != nil {
			log.Errorf("关于页渲染失败: %v", err)
			c.String(http.StatusInternalServerError, "页面渲染失败")
		}
	}
}

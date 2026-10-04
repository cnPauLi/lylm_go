package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// indexPage 仪表盘模板数据
type indexPage struct {
	Links      int64
	Groups     int64
	ApplyRows  int64
	AppVersion string
	DBVersion  string
	Build      string
}

// Dashboard 后台首页
func (h *Handler) Dashboard(c *gin.Context) {
	// AJAX 检查更新（原版 index.php?check=update），此处为桩实现，始终返回最新
	if c.Query("check") == "update" {
		c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "当前已是最新版本"})
		return
	}

	sum := h.DashboardSvc.Summary(AppVersion)
	page := &indexPage{
		Links:      sum.Links,
		Groups:     sum.Groups,
		ApplyRows:  sum.ApplyRows,
		AppVersion: sum.AppVersion,
		DBVersion:  sum.DBVersion,
		Build:      sum.Build,
	}

	data := h.newData(c, "后台管理", "index.php", "home")
	data.Page = page
	if h.Stats != nil {
		data.Stats = h.Stats.Summary(7)
	}
	h.render(c, "index.html", data)
}

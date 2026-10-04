package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/models"
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

	page := &indexPage{
		AppVersion: AppVersion,
		DBVersion:  h.Store.Get("version", ""),
		Build:      h.Store.Get("build", ""),
	}
	db := h.Store.DB()
	if err := db.Model(&models.Link{}).Count(&page.Links).Error; err != nil {
		h.Log.Errorf("统计链接数失败: %v", err)
	}
	if err := db.Model(&models.Group{}).Count(&page.Groups).Error; err != nil {
		h.Log.Errorf("统计分组数失败: %v", err)
	}
	if err := db.Model(&models.Apply{}).Where("apply_status = ?", 0).Count(&page.ApplyRows).Error; err != nil {
		h.Log.Errorf("统计待审申请失败: %v", err)
	}

	data := h.newData(c, "后台管理", "index.php", "home")
	data.Page = page
	if h.Stats != nil {
		data.Stats = h.Stats.Summary(7)
	}
	h.render(c, "index.html", data)
}

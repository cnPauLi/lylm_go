package pages

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/config"
	"lylme_spage/internal/fetch"
	"lylme_spage/internal/logger"
	"lylme_spage/internal/models"
	"lylme_spage/internal/site"
	"lylme_spage/internal/webutil"
)

// sitePage 详情页渲染数据
type sitePage struct {
	GroupName      string
	GroupIcon      string
	GroupID        int
	URLID          int
	URLName        string
	URLHref        string
	URLIcon        string
	URLTitle       string
	URLKeywords    string
	URLDescription string
	Snapshot       string
}

// SiteDetail 详情页 /site-{id}.html 与 /site?id=
func SiteDetail(cfg *config.Config, store *site.Store, engine *Engine, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := strings.TrimSuffix(c.Param("id"), ".html")
		if idStr == "" {
			idStr = c.Query("id")
		}
		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			renderNotFound(c, store, engine, log)
			return
		}

		unlocked := webutil.UnlockedIDs(c)
		allowed := append([]int{0}, unlocked...)

		var link models.Link
		if err := store.DB().Where("id = ?", id).Where("link_pwd IN ?", allowed).First(&link).Error; err != nil {
			renderNotFound(c, store, engine, log)
			return
		}

		var group models.Group
		if err := store.DB().Where("group_id = ?", link.GroupID).First(&group).Error; err != nil {
			renderNotFound(c, store, engine, log)
			return
		}
		if group.GroupPwd != 0 && !containsInt(unlocked, group.GroupPwd) {
			renderNotFound(c, store, engine, log)
			return
		}

		info := collectInfo(cfg, store, log, &link)

		icon := link.Icon
		alt := stripTags(link.Name)
		switch {
		case icon == "":
			icon = `<img src="/assets/img/default-icon.png" alt="` + alt + `" />`
		case strings.HasPrefix(icon, "<svg"):
			// 保留原始 SVG 图标
		default:
			icon = `<img src="` + icon + `" alt="` + alt + `" />`
		}

		urlKeywords := link.LinkKeywords
		if urlKeywords == "" {
			urlKeywords = anyString(info["keywords"])
		}
		urlDescription := link.LinkDesc
		if urlDescription == "" {
			urlDescription = anyString(info["description"])
		}
		if urlDescription == "" {
			urlDescription = "暂无网站描述"
		}

		data := engine.DataFor("", false, webutil.IsMobile(c.Request))
		data.Page = &sitePage{
			GroupName:      group.GroupName,
			GroupIcon:      group.GroupIcon,
			GroupID:        group.GroupID,
			URLID:          link.ID,
			URLName:        stripTags(link.Name),
			URLHref:        link.URL,
			URLIcon:        icon,
			URLTitle:       stripTags(anyString(info["title"])),
			URLKeywords:    urlKeywords,
			URLDescription: urlDescription,
			Snapshot:       store.Get("snapshot", ""),
		}

		tags, err := store.Tags()
		if err != nil {
			log.Errorf("查询导航菜单失败: %v", err)
		}
		data.Tags = tags

		c.Header("Content-Type", "text/html; charset=utf-8")
		if err := engine.Render(c.Writer, "site.html", data); err != nil {
			log.Errorf("详情页渲染失败: %v", err)
			c.String(http.StatusInternalServerError, "页面渲染失败")
		}
	}
}

// collectInfo 复刻原版 site/common 的描述/关键词采集与回写逻辑
func collectInfo(cfg *config.Config, store *site.Store, log *logger.Logger, link *models.Link) map[string]any {
	if link.LinkDesc != "" && link.LinkKeywords != "" {
		return map[string]any{
			"title":       link.Name,
			"charset":     "UTF-8",
			"icon":        "",
			"description": link.LinkDesc,
			"keywords":    link.LinkKeywords,
			"url":         link.URL,
		}
	}

	info := fetch.GetHead(link.URL, cfg.Path("cache"), true)
	if info == nil {
		info = map[string]any{}
	}

	saveDesc := "无"
	if d := strings.TrimSpace(stripTags(anyString(info["description"]))); d != "" {
		saveDesc = d
	}
	saveKw := "无"
	if k := strings.TrimSpace(stripTags(anyString(info["keywords"]))); k != "" {
		saveKw = k
	}
	saveDesc = truncateRunes(saveDesc, 255)
	saveKw = collapseCommas(strings.NewReplacer("、", ",", "，", ",", " ", ",").Replace(truncateRunes(saveKw, 512)))

	updates := map[string]any{}
	if link.LinkDesc == "" {
		updates["link_desc"] = saveDesc
		link.LinkDesc = saveDesc
	}
	if link.LinkKeywords == "" {
		updates["link_keywords"] = saveKw
		link.LinkKeywords = saveKw
	}
	if len(updates) > 0 {
		if err := store.DB().Model(&models.Link{}).Where("id = ?", link.ID).Updates(updates).Error; err != nil {
			log.Errorf("回写链接采集信息失败: %v", err)
		}
	}
	return info
}

// collapseCommas 折叠连续逗号并去除首尾逗号（等价 trim(preg_replace('/,+/', ',', ...))）
func collapseCommas(s string) string {
	var b strings.Builder
	prevComma := false
	for _, r := range s {
		if r == ',' {
			if prevComma {
				continue
			}
			prevComma = true
		} else {
			prevComma = false
		}
		b.WriteRune(r)
	}
	return strings.Trim(b.String(), ",")
}

// anyString 安全地取出 map 中的字符串值
func anyString(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// renderNotFound 渲染详情页 404
func renderNotFound(c *gin.Context, store *site.Store, engine *Engine, log *logger.Logger) {
	data := engine.DataFor("", false, webutil.IsMobile(c.Request))
	data.CSRFToken = webutil.CSRFToken(c)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusNotFound)
	if err := engine.Render(c.Writer, "404.html", data); err != nil {
		log.Errorf("404页渲染失败: %v", err)
		_, _ = c.Writer.WriteString("404 page not found")
	}
}

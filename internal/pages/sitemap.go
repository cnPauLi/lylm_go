package pages

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/config"
	"lylme_spage/internal/logger"
	"lylme_spage/internal/models"
	"lylme_spage/internal/site"
)

// Sitemap 站点地图 /sitemap.xml
func Sitemap(cfg *config.Config, store *site.Store, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		base := siteURL(c)
		day := time.Now().Format("2006-01-02")
		month := time.Now().Format("2006-01") + "-01"

		var b strings.Builder
		b.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n" + "<urlset>\n")
		b.WriteString("    <url>\n        <loc>" + base + "</loc>\n        <lastmod>" + day + "</lastmod>\n        <changefreq>daily</changefreq>\n        <priority>1.0</priority>\n    </url>\n")
		b.WriteString("    <url>\n        <loc>" + base + "/apply</loc>\n        <lastmod>" + month + "</lastmod>\n        <changefreq>monthly</changefreq>\n        <priority>0.9</priority>\n    </url>\n")
		b.WriteString("    <url>\n        <loc>" + base + "/about</loc>\n        <lastmod>" + month + "</lastmod>\n        <changefreq>monthly</changefreq>\n        <priority>0.9</priority>\n    </url>\n")

		var ids []int
		if err := store.DB().Model(&models.Link{}).Where("link_pwd = 0").Pluck("id", &ids).Error; err != nil {
			log.Errorf("查询站点地图链接失败: %v", err)
		}
		for _, id := range ids {
			b.WriteString("    <url>\n        <loc>" + base + "/site-" + strconv.Itoa(id) + ".html</loc>\n        <lastmod>" + month + "</lastmod>\n        <changefreq>monthly</changefreq>\n        <priority>0.8</priority>\n    </url>\n")
		}
		b.WriteString("</urlset>")

		c.Header("Content-Type", "text/xml")
		c.Status(http.StatusOK)
		_, _ = c.Writer.WriteString(b.String())
	}
}

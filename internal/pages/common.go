package pages

import (
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

var pageTagRe = regexp.MustCompile(`<[^>]*>`)

// stripTags 去除 HTML 标签（等价 PHP strip_tags）
func stripTags(s string) string { return pageTagRe.ReplaceAllString(s, "") }

// truncateRunes 按字符数截断（等价 mb_substr）
func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

// relativeBackground 将背景地址中的 "./" 替换为 "../"（供 /apply、/pwd 等子目录页面使用）
func relativeBackground(bg string) string {
	return strings.ReplaceAll(bg, "./", "../")
}

// containsInt 判断切片是否包含指定值
func containsInt(list []int, v int) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// siteURL 依据请求推导站点根地址 scheme://host（优先 X-Forwarded-Proto）
func siteURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if p := c.GetHeader("X-Forwarded-Proto"); p != "" {
		if i := strings.Index(p, ","); i >= 0 {
			p = p[:i]
		}
		if v := strings.ToLower(strings.TrimSpace(p)); v != "" {
			scheme = v
		}
	}
	host := strings.TrimRight(c.Request.Host, "/")
	return scheme + "://" + host
}

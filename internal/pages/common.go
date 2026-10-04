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

// siteURL 依据请求推导站点根地址 scheme://host。
// 反向代理场景优先使用 X-Forwarded-Proto / X-Forwarded-Host，确保使用当前访问域名。
func siteURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if p := firstForwarded(c.GetHeader("X-Forwarded-Proto")); p != "" {
		scheme = p
	}
	host := strings.TrimRight(firstForwarded(c.GetHeader("X-Forwarded-Host")), "/")
	if host == "" {
		host = strings.TrimRight(c.Request.Host, "/")
	}
	return scheme + "://" + host
}

// firstForwarded 取转发头的首个值（逗号分隔链中的最靠近客户端者）并规范化空白
func firstForwarded(v string) string {
	if i := strings.Index(v, ","); i >= 0 {
		v = v[:i]
	}
	return strings.ToLower(strings.TrimSpace(v))
}

package webutil

import (
	"net/http"
	"strings"
)

var mobileAgents = []string{
	"android", "midp", "nokia", "mobile", "iphone", "ipod", "blackberry", "windows phone",
}

// IsMobile 判断是否移动端（复刻原版 checkmobile()）
func IsMobile(r *http.Request) bool {
	ua := strings.ToLower(r.UserAgent())
	accept := strings.ToLower(r.Header.Get("Accept"))
	via := strings.ToLower(r.Header.Get("Via"))

	for _, m := range mobileAgents {
		if strings.Contains(ua, m) {
			return true
		}
	}
	if strings.Contains(accept, "vnd.wap") || strings.Contains(via, "wap") {
		return true
	}
	if r.Header.Get("X-Wap-Profile") != "" || r.Header.Get("Profile") != "" {
		return true
	}
	if strings.Contains(accept, "text/vnd.wap.wml") || strings.Contains(accept, "application/vnd.wap.xhtml+xml") {
		return true
	}
	return false
}

// RealIP 获取真实客户端 IP
func RealIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		return strings.TrimSpace(xr)
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i >= 0 {
		host = host[:i]
	}
	if host == "" {
		return "0.0.0.0"
	}
	return host
}

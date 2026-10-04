package webutil

import (
	"net/http"
	"strings"
	"sync"
	"time"
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

// rateEntry 单个限流键的计数与窗口重置时间
type rateEntry struct {
	count int
	reset time.Time
}

var (
	rateMu  sync.Mutex
	rateMap = map[string]*rateEntry{}
)

// RateLimit 简单固定窗口限流：key 在 window 内最多允许 max 次请求。
// 返回 true 表示放行，false 表示超限。key 不存在时创建新窗口。
func RateLimit(key string, max int, window time.Duration) bool {
	if max <= 0 || window <= 0 {
		return true
	}
	now := time.Now()
	rateMu.Lock()
	defer rateMu.Unlock()

	// 地图过大时顺带清理过期键，避免无限增长
	if len(rateMap) > 10000 {
		for k, e := range rateMap {
			if now.After(e.reset) {
				delete(rateMap, k)
			}
		}
	}

	e, ok := rateMap[key]
	if !ok || now.After(e.reset) {
		rateMap[key] = &rateEntry{count: 1, reset: now.Add(window)}
		return true
	}
	if e.count >= max {
		return false
	}
	e.count++
	return true
}

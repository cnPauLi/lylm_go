package fetch

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

const (
	maxBodyGet   = 2 << 20 // 常规抓取最多读取 2MB
	maxRedirects = 3
	userAgent    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.54 Safari/537.36"
	cacheTTL     = 7 * 24 * time.Hour
)

// 抓取错误哨兵
var (
	// ErrNotFound 目标返回 404
	ErrNotFound = errors.New("资源不存在")
	// ErrTooLarge 响应体超过限制
	ErrTooLarge = errors.New("响应体过大")
)

var httpClient = newClient()

func newClient() *http.Client {
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
		// Control 在建立连接前对实际解析出的 IP 做二次校验，可防御 DNS rebinding
		Control: func(network, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip == nil {
				return fmt.Errorf("无法解析目标地址: %s", host)
			}
			if isDisallowedIP(ip) {
				return fmt.Errorf("禁止访问内网或保留地址: %s", ip)
			}
			return nil
		},
	}
	tr := &http.Transport{
		DialContext:     dialer.DialContext,
		MaxIdleConns:    10,
		IdleConnTimeout: 30 * time.Second,
	}
	return &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// 禁用自动重定向，改为手动逐跳校验
			return http.ErrUseLastResponse
		},
	}
}

// isDisallowedIP 判断 IP 是否属于禁止访问的地址段
func isDisallowedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	return false
}

// Get 抓取指定 URL 的内容（最多 2MB）
func Get(rawURL string) ([]byte, error) {
	return doGet(rawURL, "", maxBodyGet)
}

// GetLimited 抓取内容，可指定 Referer 与最大字节数（供图片下载等场景使用）
func GetLimited(rawURL, referer string, maxBytes int64) ([]byte, error) {
	return doGet(rawURL, referer, maxBytes)
}

// ValidateURL SSRF 预校验：仅允许 http/https 且主机解析结果均为公网地址
func ValidateURL(rawURL string) error {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return fmt.Errorf("URL 格式错误")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("仅支持 http/https 协议")
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("缺少主机名")
	}
	if ip := net.ParseIP(host); ip != nil {
		if isDisallowedIP(ip) {
			return fmt.Errorf("禁止访问内网或保留地址")
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return fmt.Errorf("域名无法解析")
	}
	for _, ip := range ips {
		if isDisallowedIP(ip) {
			return fmt.Errorf("禁止访问内网或保留地址")
		}
	}
	return nil
}

func doGet(rawURL, referer string, maxBytes int64) ([]byte, error) {
	current := strings.TrimSpace(rawURL)
	if current == "" {
		return nil, fmt.Errorf("空 URL")
	}
	for i := 0; i <= maxRedirects; i++ {
		u, err := url.Parse(current)
		if err != nil {
			return nil, err
		}
		switch strings.ToLower(u.Scheme) {
		case "http", "https":
		default:
			return nil, fmt.Errorf("仅支持 http/https 协议")
		}

		req, err := http.NewRequest(http.MethodGet, current, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/*,*/*;q=0.8")
		req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
		if referer != "" {
			req.Header.Set("Referer", referer)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if isRedirectStatus(resp.StatusCode) && i < maxRedirects {
			loc := resp.Header.Get("Location")
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))
			_ = resp.Body.Close()
			if loc == "" {
				return nil, fmt.Errorf("重定向缺少 Location")
			}
			current = GetURLPath(loc, current)
			continue
		}

		defer resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("请求失败，状态码 %d", resp.StatusCode)
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
		if err != nil {
			return nil, err
		}
		if int64(len(data)) > maxBytes {
			return nil, ErrTooLarge
		}
		return data, nil
	}
	return nil, fmt.Errorf("重定向次数过多")
}

func isRedirectStatus(code int) bool {
	switch code {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	}
	return false
}

var (
	titleRe = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	linkRe  = regexp.MustCompile(`(?is)<link\b[^>]*>`)
	relRe   = regexp.MustCompile(`(?is)\brel\s*=\s*["']?([^"'>]+)`)
	hrefRe  = regexp.MustCompile(`(?is)\bhref\s*=\s*["']?([^"'>\s]+)`)
)

// GetHead 抓取站点 head 信息，等价于原 PHP get_head()
func GetHead(rawURL, cacheDir string, useCache bool) map[string]any {
	result := map[string]any{
		"code":        403,
		"msg":         "请求失败",
		"title":       "",
		"charset":     "UTF-8",
		"icon":        "",
		"description": "",
		"keywords":    "",
		"url":         rawURL,
	}

	if !isHTTPURL(rawURL) {
		return result
	}

	if useCache {
		if cached, ok := readCache(cacheDir, rawURL); ok {
			return cached
		}
	}

	data, err := Get(rawURL)
	if err != nil || len(data) == 0 {
		result["code"] = 404
		return result
	}

	body := string(data)
	title := ""
	if m := titleRe.FindStringSubmatch(body); m != nil {
		title = strings.TrimSpace(m[1])
	}

	icon := extractIcon(body, rawURL)
	description := extractMeta(body, "description")
	keywords := extractMeta(body, "keywords")

	result["code"] = 200
	result["msg"] = "请求成功"
	result["title"] = title
	result["charset"] = "UTF-8"
	result["icon"] = strings.TrimSpace(icon)
	result["description"] = description
	result["keywords"] = keywords
	result["url"] = rawURL

	if useCache {
		writeCache(cacheDir, rawURL, result)
	}
	return result
}

func isHTTPURL(rawURL string) bool {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	return (scheme == "http" || scheme == "https") && u.Host != ""
}

// extractIcon 提取站点图标，缺失时回退到 favicon.ico（需可达）
func extractIcon(body, pageURL string) string {
	for _, tag := range linkRe.FindAllString(body, -1) {
		rel := ""
		if m := relRe.FindStringSubmatch(tag); m != nil {
			rel = strings.ToLower(m[1])
		}
		if !strings.Contains(rel, "icon") {
			continue
		}
		if m := hrefRe.FindStringSubmatch(tag); m != nil {
			href := strings.TrimSpace(m[1])
			if href == "" {
				continue
			}
			if resolved := GetURLPath(href, pageURL); resolved != "" {
				return resolved
			}
		}
	}

	u, err := url.Parse(pageURL)
	if err != nil || u.Host == "" {
		return ""
	}
	favicon := u.Scheme + "://" + u.Host + "/favicon.ico"
	if _, err := Get(favicon); err == nil {
		return favicon
	}
	return ""
}

func extractMeta(body, name string) string {
	quoted := regexp.QuoteMeta(name)
	re1 := regexp.MustCompile(`(?is)<meta\b[^>]*\bname\s*=\s*["']?` + quoted + `["']?[^>]*\bcontent\s*=\s*["']?([^"'>]+)`)
	if m := re1.FindStringSubmatch(body); m != nil {
		return strings.TrimSpace(m[1])
	}
	re2 := regexp.MustCompile(`(?is)<meta\b[^>]*\bcontent\s*=\s*["']?([^"'>]+)["']?[^>]*\bname\s*=\s*["']?` + quoted + `["']?`)
	if m := re2.FindStringSubmatch(body); m != nil {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// GetURLPath 将可能为相对路径的 ref 基于 base 解析为完整 URL
func GetURLPath(ref, base string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	ru, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	if ru.IsAbs() {
		return ref
	}
	bu, err := url.Parse(base)
	if err != nil || bu.Host == "" {
		return ref
	}
	return bu.ResolveReference(ru).String()
}

func cacheFile(cacheDir, rawURL string) string {
	sum := md5.Sum([]byte(rawURL))
	return filepath.Join(cacheDir, fmt.Sprintf("%x.txt", sum))
}

func readCache(cacheDir, rawURL string) (map[string]any, bool) {
	if cacheDir == "" {
		return nil, false
	}
	path := cacheFile(cacheDir, rawURL)
	fi, err := os.Stat(path)
	if err != nil || fi.IsDir() {
		return nil, false
	}
	if time.Since(fi.ModTime()) >= cacheTTL {
		return nil, false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil || len(data) == 0 {
		return nil, false
	}
	if v, ok := data["code"].(float64); ok {
		data["code"] = int(v)
	}
	return data, true
}

func writeCache(cacheDir, rawURL string, data map[string]any) {
	if cacheDir == "" {
		return
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	_ = os.WriteFile(cacheFile(cacheDir, rawURL), raw, 0o644)
}

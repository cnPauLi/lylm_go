package upload

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/config"
	"lylme_spage/internal/fetch"
	"lylme_spage/internal/logger"
	"lylme_spage/internal/site"
	"lylme_spage/internal/webutil"
)

// 允许的图片后缀与内容类型
var allowedExts = map[string]bool{
	"jpeg": true, "jpg": true, "png": true, "gif": true, "webp": true, "ico": true,
}

const maxSize = 5 << 20 // 5MB

// result 上传接口统一响应
type result struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	URL  string `json:"url,omitempty"`
}

// Handler 实现 /include/file（图片上传 / 远程抓取）
func Handler(cfg *config.Config, store *site.Store, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawURL := c.PostForm("url")
		if rawURL == "" {
			rawURL = c.Query("url")
		}
		referer := c.PostForm("referer")
		if referer == "" {
			referer = c.Query("referer")
		}

		if rawURL != "" {
			handleDownload(c, cfg, log, rawURL, referer)
			return
		}

		if fh, err := c.FormFile("file"); err == nil && fh != nil {
			handleUpload(c, cfg, log, fh)
			return
		}

		writeJSON(c, 0, "error", "")
	}
}

// handleDownload 处理远程图片抓取
func handleDownload(c *gin.Context, cfg *config.Config, log *logger.Logger, rawURL, referer string) {
	ip := webutil.RealIP(c.Request)
	if !webutil.RateLimit("download_img_"+ip, 3, time.Minute) {
		writeJSON(c, -8, "抓取请求过于频繁，请稍后再试", "")
		return
	}
	if err := fetch.ValidateURL(rawURL); err != nil {
		writeJSON(c, -6, "不安全的下载地址", "")
		return
	}

	ext := extFromURL(rawURL)
	if !allowedExts[ext] {
		writeJSON(c, -4, "抓取的图片类型 ."+ext+" 不支持", "")
		return
	}

	data, err := fetch.GetLimited(rawURL, referer, maxSize)
	if errors.Is(err, fetch.ErrTooLarge) {
		writeJSON(c, -1, "抓取的图片超过5M", "")
		return
	}
	if err != nil || len(data) == 0 {
		if err != nil {
			log.Warnf("远程抓图失败: %s: %v", rawURL, err)
		}
		writeJSON(c, -1, "抓取图片失败", "")
		return
	}
	if !validImage(data) {
		writeJSON(c, -4, "抓取的文件不是有效的图片", "")
		return
	}

	name, ok := buildName(ext)
	if !ok {
		writeJSON(c, -5, "文件后缀不合法", "")
		return
	}
	dir := cfg.Path("files", "download")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Errorf("创建目录失败 %s: %v", dir, err)
		writeJSON(c, -1, "抓取图片失败", "")
		return
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		log.Errorf("写入抓取图片失败: %v", err)
		writeJSON(c, -1, "抓取图片失败", "")
		return
	}
	writeJSON(c, 200, "抓取图片成功", "/files/download/"+name)
}

// handleUpload 处理本地文件上传
func handleUpload(c *gin.Context, cfg *config.Config, log *logger.Logger, fh *multipart.FileHeader) {
	if fh.Size > maxSize {
		writeJSON(c, -1, "图片不能超过5M", "")
		return
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fh.Filename), "."))
	if !allowedExts[ext] {
		writeJSON(c, -4, "上传的图片类型 ."+ext+" 不支持", "")
		return
	}

	f, err := fh.Open()
	if err != nil {
		writeJSON(c, -3, "非法上传请求", "")
		return
	}
	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(io.LimitReader(f, maxSize+1))
	if err != nil {
		writeJSON(c, -1, "上传失败", "")
		return
	}
	if int64(len(data)) > maxSize {
		writeJSON(c, -1, "图片不能超过5M", "")
		return
	}
	if !validImage(data) {
		writeJSON(c, -4, "上传的文件不是有效的图片", "")
		return
	}

	name, ok := buildName(ext)
	if !ok {
		writeJSON(c, -5, "文件后缀不合法", "")
		return
	}
	dir := cfg.Path("files", "upload")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Errorf("创建目录失败 %s: %v", dir, err)
		writeJSON(c, -1, "上传失败", "")
		return
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		log.Errorf("写入上传图片失败: %v", err)
		writeJSON(c, -1, "上传失败", "")
		return
	}
	writeJSON(c, 200, "上传成功", "/files/upload/"+name)
}

// validImage 依据内容类型判断是否为受支持的图片
func validImage(data []byte) bool {
	ct := http.DetectContentType(data)
	switch ct {
	case "image/jpeg", "image/png", "image/gif", "image/webp", "image/x-icon":
	default:
		return false
	}
	// JPEG/PNG/GIF 额外做一次解码校验，确保内容真实可解析
	switch ct {
	case "image/jpeg", "image/png", "image/gif":
		if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
			return false
		}
	}
	return true
}

// buildName 生成随机文件名并校验后缀合法性
func buildName(ext string) (string, bool) {
	ext = strings.ToLower(ext)
	if !allowedExts[ext] {
		return "", false
	}
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		return "", false
	}
	name := "img_" + hex.EncodeToString(b) + "." + ext
	if !validName(name) {
		return "", false
	}
	return name, true
}

// validName 仅允许字母、数字、下划线与点，防止路径穿越
func validName(name string) bool {
	if name == "" || strings.Contains(name, "..") {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '_' || r == '-' || r == '.':
		default:
			return false
		}
	}
	return true
}

// extFromURL 从 URL 路径提取小写后缀（忽略 query）
func extFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimPrefix(filepath.Ext(u.Path), "."))
}

func writeJSON(c *gin.Context, code int, msg, u string) {
	resp := result{Code: code, Msg: msg}
	if u != "" {
		resp.URL = u
	}
	c.JSON(http.StatusOK, resp)
}

package pages

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"

	"lylme_spage/internal/config"
	"lylme_spage/internal/fetch"
	"lylme_spage/internal/logger"
	"lylme_spage/internal/site"
)

var (
	bingURLRe   = regexp.MustCompile(`(?is)<url>(.+?)</url>`)
	bingStartRe = regexp.MustCompile(`(?is)<fullstartdate>(.+?)</fullstartdate>`)
)

// Qrcode 二维码 /include/qrcode?text=
func Qrcode(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		text := c.Query("text")
		if text == "" {
			c.String(http.StatusBadRequest, "text 参数不能为空")
			return
		}
		if len(text) > 1024 {
			c.String(http.StatusBadRequest, "text 长度不能超过 1024")
			return
		}
		png, err := qrcode.Encode(text, qrcode.Medium, 256)
		if err != nil {
			log.Errorf("生成二维码失败: %v", err)
			c.String(http.StatusInternalServerError, "生成二维码失败")
			return
		}
		c.Header("Content-Type", "image/png")
		c.Header("Cache-Control", "max-age=300")
		_, _ = c.Writer.Write(png)
	}
}

// BingImage Bing 每日壁纸 /assets/img/bing
func BingImage(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := fetch.Get("http://cn.bing.com/HPImageArchive.aspx?idx=0&n=1")
		if err != nil {
			writeBingError(c)
			return
		}
		m := bingURLRe.FindSubmatch(body)
		if m == nil {
			writeBingError(c)
			return
		}
		imgURL := "http://cn.bing.com" + string(m[1])
		img, err := fetch.GetLimited(imgURL, "", 10<<20)
		if err != nil || len(img) == 0 {
			log.Warnf("Bing 壁纸下载失败: %v", err)
			writeBingError(c)
			return
		}
		c.Header("Content-Type", "image/jpeg")
		c.Header("Cache-Control", "public, max-age=86400")
		_, _ = c.Writer.Write(img)
	}
}

func writeBingError(c *gin.Context) {
	c.String(http.StatusNotFound, "Error: Invalid image URL.")
}

// CronImage 缓存 Bing 壁纸 /assets/img/cron?key=
func CronImage(cfg *config.Config, store *site.Store, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")

		key := c.Query("key")
		if key == "" {
			_, _ = c.Writer.WriteString("错误：密钥为空，请传入包含参数key的GET请求<br>\n    请求示例：<b>http://" + c.Request.Host + "/assets/img/cron?key=秘钥</b>")
			return
		}
		pass := store.Get("cron_key", "")
		if pass == "" {
			_, _ = c.Writer.WriteString("<b>错误：</b>未配置CRON密钥，请在后台配置密钥")
			return
		}
		if pass != key {
			_, _ = c.Writer.WriteString("错误：传入参数key与密钥不匹配")
			return
		}

		idx := c.DefaultQuery("idx", "0")
		body, err := fetch.Get("http://cn.bing.com/HPImageArchive.aspx?idx=" + idx + "&n=1")
		if err != nil {
			_, _ = c.Writer.WriteString("错误：获取 Bing 壁纸失败")
			return
		}
		m := bingURLRe.FindSubmatch(body)
		if m == nil {
			_, _ = c.Writer.WriteString("错误：获取 Bing 壁纸失败")
			return
		}
		imgURL := "http://cn.bing.com" + string(m[1])

		var crdate string
		if cm := bingStartRe.FindSubmatch(body); cm != nil {
			if t, err := time.ParseInLocation("200601021504", string(cm[1]), time.Local); err == nil {
				crdate = t.Format("2006年01月02日 15:04")
			}
		}
		_, _ = c.Writer.WriteString("壁纸地址：" + imgURL + "<br>")
		if crdate != "" {
			_, _ = c.Writer.WriteString("发布时间：" + crdate + "<br>")
		}

		img, err := fetch.GetLimited(imgURL, "", 20<<20)
		if err != nil || len(img) == 0 {
			log.Warnf("Bing 壁纸下载失败: %v", err)
			_, _ = c.Writer.WriteString("<p><font color='red'>错误： 保存壁纸失败</font></p>")
			return
		}

		dir := cfg.Path("assets", "img")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Errorf("创建壁纸目录失败: %v", err)
		}
		target := filepath.Join(dir, "background.jpg")
		if err := os.WriteFile(target, img, 0o644); err != nil {
			log.Errorf("保存壁纸失败: %v", err)
			_, _ = c.Writer.WriteString("<p><font color='red'>错误： 保存文件 <b>" + target + "</b> 失败，请检查目录权限</font></p>")
			return
		}
		_, _ = c.Writer.WriteString("壁纸大小：" + strconv.Itoa(len(img)/1024) + "KB<br>")
		_, _ = c.Writer.WriteString("<p><font color='green'>成功：当前壁纸已与Bing同步！</font></p>")
	}
}

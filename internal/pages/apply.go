package pages

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/config"
	"lylme_spage/internal/fetch"
	"lylme_spage/internal/logger"
	"lylme_spage/internal/models"
	"lylme_spage/internal/service"
	"lylme_spage/internal/site"
	"lylme_spage/internal/webutil"
)

// Apply 收录申请页：/apply/、/apply、/apply/index.php（GET/POST）
func Apply(cfg *config.Config, store *site.Store, engine *Engine, log *logger.Logger) gin.HandlerFunc {
	applySvc := service.NewApplyService(store)
	return func(c *gin.Context) {
		// 1) 抓取目标站点 head 信息（AJAX: index.php?url=）
		if rawURL := c.Query("url"); rawURL != "" {
			c.Header("Content-Type", "application/json")
			ip := webutil.RealIP(c.Request)
			if !webutil.RateLimit("apply_geturl_"+ip, 5, 60*time.Second) {
				_, _ = c.Writer.WriteString(`{"code":"-8","msg":"获取请求频繁，请稍后再试"}`)
				return
			}
			if err := fetch.ValidateURL(rawURL); err != nil {
				_, _ = c.Writer.WriteString(`{"code":"-9","msg":"链接地址不合法"}`)
				return
			}
			head := fetch.GetHead(rawURL, cfg.Path("cache"), false)
			b, err := json.Marshal(head)
			if err != nil {
				_, _ = c.Writer.WriteString(`{"code":"-9","msg":"链接地址不合法"}`)
				return
			}
			_, _ = c.Writer.Write(b)
			return
		}

		// 2) 提交收录申请（AJAX: index.php?submit=post）
		if c.Query("submit") == "post" || c.PostForm("submit") == "post" {
			c.Header("Content-Type", "application/json; charset=utf-8")
			if !webutil.VerifyCaptcha(c, c.PostForm("authcode")) {
				_, _ = c.Writer.WriteString(`{"code": "-6", "msg": "验证码错误"}`)
				return
			}
			if store.Get("apply", "0") == "2" {
				_, _ = c.Writer.WriteString(`{"code": "400", "msg": "网站已关闭收录"}`)
				return
			}
			_, _ = c.Writer.WriteString(insertApply(applySvc, store, log, c))
			return
		}

		// 3) 渲染申请页面
		data := engine.DataFor("", false, webutil.IsMobile(c.Request))
		data.CSRFToken = webutil.CSRFToken(c)
		data.BackgroundImg = relativeBackground(data.BackgroundImg)

		groups, err := store.Groups(webutil.UnlockedIDs(c))
		if err != nil {
			log.Errorf("查询分组失败: %v", err)
		}
		data.Groups = groups

		c.Header("Content-Type", "text/html; charset=utf-8")
		if err := engine.Render(c.Writer, "apply.html", data); err != nil {
			log.Errorf("申请页渲染失败: %v", err)
			c.String(http.StatusInternalServerError, "页面渲染失败")
		}
	}
}

// insertApply 校验并写入收录申请，返回与原始 PHP apply() 一致的 JSON 字符串
func insertApply(applySvc *service.ApplyService, store *site.Store, log *logger.Logger, c *gin.Context) string {
	name := stripTags(strings.TrimSpace(c.PostForm("name")))
	rawURL := stripTags(strings.TrimSpace(c.PostForm("url")))
	icon := stripTags(strings.TrimSpace(c.PostForm("icon")))
	groupID, _ := strconv.Atoi(strings.TrimSpace(c.PostForm("group_id")))
	status, _ := strconv.Atoi(store.Get("apply", "0"))
	userIP := webutil.RealIP(c.Request)

	if name == "" || rawURL == "" || groupID == 0 {
		return `{"code": "-1", "msg": "必填项不能为空"}`
	}
	if !validApplyURL(rawURL) {
		return `{"code": "-2", "msg": "链接不符合要求"}`
	}
	// apply_name 为 varchar(20)，超长截断并追加省略号
	if utf8.RuneCountInString(name) > 20 {
		name = truncateRunes(name, 17) + "..."
	}
	if len(name) > 255 || len(rawURL) > 255 || len(icon) > 255 || len(userIP) > 255 {
		return `{"code": "500", "msg": "参数过长"}`
	}

	var exists int64
	if err := store.DB().Model(&models.Apply{}).Where("apply_url = ?", rawURL).Count(&exists).Error; err != nil {
		log.Errorf("查询重复申请失败: %v", err)
		return `{"code": "-4", "msg": "数据库操作失败，请联系网站管理员"}`
	}
	if exists > 0 {
		return `{"code": "-3", "msg": "链接已存在，请勿重复提交"}`
	}

	rec, err := applySvc.Create(service.ApplyCreateInput{
		Name: name, URL: rawURL, GroupID: groupID, Icon: icon, Desc: userIP,
	})
	if err != nil {
		log.Errorf("收录申请写入失败: %v", err)
		return `{"code": "-4", "msg": "数据库操作失败，请联系网站管理员"}`
	}

	switch status {
	case 0:
		return `{"code": "200", "msg": "请等待管理员审核"}`
	case 1:
		// 免审核：复用审核通过规则（写链接 + 状态置 1），以链接表实际增量判定结果
		var before, after int64
		_ = store.DB().Model(&models.Link{}).Count(&before).Error
		_ = applySvc.SetStatus([]int{rec.ApplyID}, 1)
		_ = store.DB().Model(&models.Link{}).Count(&after).Error
		if after == before+1 {
			return `{"code": "200", "msg": "网站已收录"}`
		}
		return `{"code": "-5", "msg": "收录失败，请联系网站管理员"}`
	default:
		return `{"code": "-4", "msg": "未知状态码"}`
	}
}

// validApplyURL 等价 PHP filter_var($url, FILTER_VALIDATE_URL)
func validApplyURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return false
	}
	return u.Host != ""
}

// WxPlus 微信推送 /apply/wxplus.php
func WxPlus(cfg *config.Config, store *site.Store, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.TrimSpace(store.Get("wxplus", "")) == "" {
			c.Header("Content-Type", "text/plain; charset=utf-8")
			_, _ = c.Writer.WriteString("微信推送未开启")
			return
		}
		ip := webutil.RealIP(c.Request)
		if !webutil.RateLimit("wxplus_"+ip, 5, 60*time.Second) {
			c.Header("Content-Type", "text/plain; charset=utf-8")
			_, _ = c.Writer.WriteString("请求频繁，请稍后再试")
			return
		}
		// 原版 wxPlus() 的推送逻辑位于混淆代码中（include/include.php），
		// 按 AGENTS.md 安全规范不予移植；此处仅保留鉴权与限流并返回成功。
		c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "ok", "data": "ok"})
	}
}

package webutil

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// 会话键名
const (
	sessKeyUnlocked = "unlocked"
	sessKeyPwd      = "pass"
	sessKeyCSRF     = "csrf_token"
	sessKeyCaptcha  = "authcode"
)

// UnlockedIDs 读取会话中已解锁的加密密码组 ID 列表
func UnlockedIDs(c *gin.Context) []int {
	s := sessions.Default(c)
	if v, ok := s.Get(sessKeyUnlocked).([]int); ok {
		return v
	}
	return nil
}

// SetUnlockedIDs 写入已解锁的加密密码组 ID 列表
func SetUnlockedIDs(c *gin.Context, ids []int) {
	s := sessions.Default(c)
	s.Set(sessKeyUnlocked, ids)
	_ = s.Save()
}

// ClearUnlocked 清空已解锁的加密密码组
func ClearUnlocked(c *gin.Context) {
	s := sessions.Default(c)
	s.Delete(sessKeyUnlocked)
	_ = s.Save()
}

// IsPwdLogged 是否已通过访问密码登录
func IsPwdLogged(c *gin.Context) bool {
	s := sessions.Default(c)
	v, _ := s.Get(sessKeyPwd).(bool)
	return v
}

// SetPwdLogged 设置访问密码登录状态
func SetPwdLogged(c *gin.Context, v bool) {
	s := sessions.Default(c)
	s.Set(sessKeyPwd, v)
	_ = s.Save()
}

// CSRFToken 返回当前会话的 CSRF Token，不存在时生成 64 位十六进制随机串
func CSRFToken(c *gin.Context) string {
	s := sessions.Default(c)
	if v, ok := s.Get(sessKeyCSRF).(string); ok && v != "" {
		return v
	}
	tok := randomHex(32)
	s.Set(sessKeyCSRF, tok)
	_ = s.Save()
	return tok
}

// CSRFVerifyToken 校验给定的 CSRF Token
func CSRFVerifyToken(c *gin.Context, token string) bool {
	if token == "" {
		return false
	}
	s := sessions.Default(c)
	want, _ := s.Get(sessKeyCSRF).(string)
	if want == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(want), []byte(token)) == 1
}

// CSRFVerify 读取表单字段 csrf_token 并校验
func CSRFVerify(c *gin.Context) bool {
	return CSRFVerifyToken(c, c.PostForm("csrf_token"))
}

// SetCaptchaCode 保存验证码到会话
func SetCaptchaCode(c *gin.Context, code string) {
	s := sessions.Default(c)
	s.Set(sessKeyCaptcha, code)
	_ = s.Save()
}

// VerifyCaptcha 校验验证码（忽略大小写），无论是否通过都会清除已保存的验证码
func VerifyCaptcha(c *gin.Context, code string) bool {
	s := sessions.Default(c)
	stored, _ := s.Get(sessKeyCaptcha).(string)
	s.Delete(sessKeyCaptcha)
	_ = s.Save()
	if stored == "" {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(stored), strings.TrimSpace(code))
}

// randomHex 生成 n 字节随机数据的十六进制字符串（长度为 2n）
func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return strings.Repeat("0", n*2)
	}
	return hex.EncodeToString(b)
}

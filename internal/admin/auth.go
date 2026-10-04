package admin

import (
	"crypto/subtle"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/service"
	"lylme_spage/internal/webutil"
)

// 登录安全策略常量（与原版登录页一致）
const (
	loginMaxFailures   = 5
	loginLockSeconds   = 60
	loginLockMaxSecond = 1800
)

// loginPage 登录页模板数据
type loginPage struct {
	NeedCaptcha bool
	IsLocked    bool
	LockMinutes int
	LoginError  string
	LoginUser   string
	Background  string
}

// clientIP 仅信任 RemoteAddr，避免 X-Forwarded-For 伪造
func clientIP(c *gin.Context) string {
	host := c.Request.RemoteAddr
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	if host == "" {
		return "unknown"
	}
	return host
}

// lockState 返回当前 IP 的失败状态（自动清理过期锁定）
func (h *Handler) lockState(ip string) *failState {
	st, ok := h.fails[ip]
	if !ok {
		st = &failState{}
		h.fails[ip] = st
	}
	// 锁定到期后自动解除并清空计数
	if st.lockUntil > 0 && time.Now().Unix() > st.lockUntil {
		st.count = 0
		st.lockUntil = 0
	}
	return st
}

// recordFailure 记录一次登录失败，达到阈值则按轮次递增锁定
func (h *Handler) recordFailure(ip string) *failState {
	h.mu.Lock()
	defer h.mu.Unlock()
	st := h.lockState(ip)
	st.count++
	if st.count >= loginMaxFailures {
		st.level++
		secs := loginLockSeconds << (st.level - 1)
		if secs > loginLockMaxSecond || secs <= 0 {
			secs = loginLockMaxSecond
		}
		st.lockUntil = time.Now().Unix() + int64(secs)
		st.count = 0
	}
	return st
}

// resetFailure 登录成功后清除当前 IP 失败记录
func (h *Handler) resetFailure(ip string) {
	h.mu.Lock()
	delete(h.fails, ip)
	h.mu.Unlock()
}

// LoginPage 渲染登录页 / 处理登出
func (h *Handler) LoginPage(c *gin.Context) {
	if _, ok := c.GetQuery("logout"); ok {
		webutil.ClearAdmin(c)
		c.Redirect(http.StatusFound, h.Base()+"/login")
		return
	}
	if authed(c, h.Store) {
		c.Redirect(http.StatusFound, h.Base()+"/")
		return
	}

	ip := clientIP(c)
	h.mu.Lock()
	st := h.lockState(ip)
	count, lockUntil := st.count, st.lockUntil
	h.mu.Unlock()

	now := time.Now().Unix()
	page := &loginPage{
		NeedCaptcha: count >= 1,
		IsLocked:    lockUntil > now,
		LoginError:  flashGet(c, sessLoginError),
		LoginUser:   flashGet(c, sessLoginUser),
	}
	if page.IsLocked {
		page.LockMinutes = int((lockUntil - now + 59) / 60)
	}
	page.Background = strings.ReplaceAll(h.Store.Get("background", ""), "./", "../")

	data := h.newData(c, "后台登录", "login", "")
	data.Page = page
	h.render(c, "login.html", data)
}

// LoginSubmit 处理登录提交
func (h *Handler) LoginSubmit(c *gin.Context) {
	user := strings.TrimSpace(c.PostForm("user"))
	pass := c.PostForm("pass")
	if user == "" && pass == "" {
		h.LoginPage(c)
		return
	}

	ip := clientIP(c)
	h.mu.Lock()
	st := h.lockState(ip)
	count, lockUntil := st.count, st.lockUntil
	h.mu.Unlock()

	now := time.Now().Unix()
	if lockUntil > now {
		remain := int((lockUntil - now + 59) / 60)
		flashSet(c, sessLoginError, "登录尝试次数过多，请 "+itoa(remain)+" 分钟后再试")
		flashSet(c, sessLoginUser, user)
		c.Redirect(http.StatusFound, h.Base()+"/login")
		return
	}

	// 需要验证码时先校验（一次性使用）
	if count >= 1 {
		if !webutil.VerifyCaptcha(c, c.PostForm("authcode")) {
			h.recordFailure(ip)
			flashSet(c, sessLoginError, "验证码错误，请重新输入")
			flashSet(c, sessLoginUser, user)
			c.Redirect(http.StatusFound, h.Base()+"/login")
			return
		}
	}

	wantUser := h.Store.Get("admin_user", "admin")
	wantPwd := h.Store.Get("admin_pwd", "")
	gotPwd := service.AdminPasswordHash(pass)
	if user != "" &&
		subtle.ConstantTimeCompare([]byte(wantUser), []byte(user)) == 1 &&
		subtle.ConstantTimeCompare([]byte(wantPwd), []byte(gotPwd)) == 1 {
		h.resetFailure(ip)
		webutil.SetAdmin(c, adminSign(h.Store))
		c.Redirect(http.StatusFound, h.Base()+"/")
		return
	}

	h.recordFailure(ip)
	flashSet(c, sessLoginError, "用户名或密码不正确")
	flashSet(c, sessLoginUser, user)
	c.Redirect(http.StatusFound, h.Base()+"/login")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

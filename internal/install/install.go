// Package install 实现 /install/ 安装向导：环境检测、数据库配置、写入 .env、
// 建表导入种子数据、创建管理员账号，并生成 install/install.lock。
package install

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// LockRel 安装锁文件相对项目根目录的路径
const LockRel = "install/install.lock"

// Installed 判断系统是否已安装（存在安装锁文件即为已安装）
func Installed(root string) bool {
	fi, err := os.Stat(filepath.Join(root, LockRel))
	return err == nil && !fi.IsDir()
}

// Guard 未安装时的访问守卫：非白名单请求 302 重定向到 /install/。
// 白名单：/install、/assets、/template、/site/static、/files、/healthz、/include。
// AJAX/JSON 请求直接返回 503 JSON，避免破坏前端对 JSON 的解析。
func Guard(root string) gin.HandlerFunc {
	allowed := []string{
		"/install", "/assets", "/template", "/site/static",
		"/files", "/healthz", "/include",
	}
	return func(c *gin.Context) {
		if Installed(root) {
			c.Next()
			return
		}
		path := c.Request.URL.Path
		for _, p := range allowed {
			if path == p || strings.HasPrefix(path, p+"/") {
				c.Next()
				return
			}
		}
		if wantsJSON(c) {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"code": 503, "msg": "系统尚未安装"})
			return
		}
		c.Redirect(http.StatusFound, "/install/")
		c.Abort()
	}
}

// wantsJSON 判断请求是否期望 JSON 响应
func wantsJSON(c *gin.Context) bool {
	if c.GetHeader("X-Requested-With") == "XMLHttpRequest" {
		return true
	}
	if strings.Contains(strings.ToLower(c.GetHeader("Accept")), "application/json") {
		return true
	}
	if strings.Contains(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		return true
	}
	return strings.HasSuffix(strings.ToLower(c.Request.URL.Path), ".json")
}

// envKeys 需要写入 .env 的键顺序
var envKeys = []string{
	"APP_ENV", "SERVER_HOST", "SERVER_PORT", "SERVER_MODE", "ADMIN_PATH",
	"DB_DRIVER", "SQLITE_PATH", "DB_DSN",
	"MYSQL_HOST", "MYSQL_PORT", "MYSQL_USER", "MYSQL_PASSWORD", "MYSQL_DATABASE", "MYSQL_CHARSET",
	"LOG_DIR", "LOG_LEVEL", "SESSION_SECRET", "CDN_PUBLIC",
}

// writeEnv 将 values 写入 <root>/.env：保留已有文件中的其他键，覆盖同名键；
// SESSION_SECRET 未显式提供时沿用已有值，否则生成随机 32 字节 hex。
// 采用临时文件 + rename 原子写入，权限 0600。
func writeEnv(root string, values map[string]string) error {
	path := filepath.Join(root, ".env")
	existing := readEnvFile(path)

	if v := values["SESSION_SECRET"]; v == "" {
		if s := existing["SESSION_SECRET"]; s != "" && s != "lylme_spage_session_secret_change_me" {
			values["SESSION_SECRET"] = s
		} else {
			values["SESSION_SECRET"] = randomHex(32)
		}
	}

	written := make(map[string]bool, len(values))
	var lines []string
	if data, err := os.ReadFile(path); err == nil {
		text := strings.ReplaceAll(string(data), "\r\n", "\n")
		for _, line := range strings.Split(text, "\n") {
			if key, ok := parseEnvKey(line); ok {
				if v, hit := values[key]; hit {
					lines = append(lines, key+"="+v)
					written[key] = true
					continue
				}
			}
			lines = append(lines, line)
		}
	}
	for _, k := range envKeys {
		if v, ok := values[k]; ok && !written[k] {
			lines = append(lines, k+"="+v)
			written[k] = true
		}
	}

	content := strings.Join(lines, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return atomicWrite(path, []byte(content), 0o600)
}

// readEnvFile 读取已有 .env 中的键值对（忽略注释与空行）
func readEnvFile(path string) map[string]string {
	m := map[string]string{}
	data, err := os.ReadFile(path)
	if err != nil {
		return m
	}
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		if key, ok := parseEnvKey(line); ok {
			if eq := strings.IndexByte(strings.TrimSpace(line), '='); eq > 0 {
				m[key] = strings.TrimSpace(strings.TrimSpace(line)[eq+1:])
			}
		}
	}
	return m
}

// parseEnvKey 解析形如 KEY=VALUE 的行，返回键名
func parseEnvKey(line string) (string, bool) {
	t := strings.TrimSpace(line)
	if t == "" || strings.HasPrefix(t, "#") {
		return "", false
	}
	eq := strings.IndexByte(t, '=')
	if eq <= 0 {
		return "", false
	}
	key := strings.TrimSpace(t[:eq])
	if !validEnvKey(key) {
		return "", false
	}
	return key, true
}

func validEnvKey(key string) bool {
	for i := 0; i < len(key); i++ {
		c := key[i]
		if !(c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

// randomHex 生成 n 字节随机数的十六进制字符串
func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000000")))
	}
	return hex.EncodeToString(b)
}

// atomicWrite 原子写入文件（同目录临时文件 + rename）
func atomicWrite(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".env-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

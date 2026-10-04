package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// MySQLConfig MySQL 连接配置
type MySQLConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	Charset  string
}

// Config 应用配置，全部来源于 .env / 环境变量
type Config struct {
	AppEnv   string
	RootDir  string
	Host     string
	Port     int
	Mode     string
	AdminPath string

	DBDriver   string
	SQLitePath string
	DBDSN      string
	MySQL      MySQLConfig

	LogDir   string
	LogLevel string

	SessionSecret string
	CDNPublic     string
}

// Load 读取 .env 与环境变量并构建配置
func Load() (*Config, error) {
	loadDotEnv()

	c := &Config{
		AppEnv:        getEnv("APP_ENV", "production"),
		RootDir:       resolveRootDir(),
		Host:          getEnv("SERVER_HOST", "0.0.0.0"),
		Port:          getEnvInt("SERVER_PORT", 8080),
		Mode:          getEnv("SERVER_MODE", "release"),
		AdminPath:     strings.Trim(getEnv("ADMIN_PATH", "admin"), "/"),
		DBDriver:      strings.ToLower(getEnv("DB_DRIVER", "sqlite")),
		SQLitePath:    getEnv("SQLITE_PATH", "data/lylme.db"),
		DBDSN:         getEnv("DB_DSN", ""),
		LogDir:        getEnv("LOG_DIR", "logs"),
		LogLevel:      strings.ToLower(getEnv("LOG_LEVEL", "info")),
		SessionSecret: getEnv("SESSION_SECRET", "lylme_spage_session_secret_change_me"),
		CDNPublic:     getEnv("CDN_PUBLIC", ""),
		MySQL: MySQLConfig{
			Host:     getEnv("MYSQL_HOST", "127.0.0.1"),
			Port:     getEnvInt("MYSQL_PORT", 3306),
			User:     getEnv("MYSQL_USER", "root"),
			Password: getEnv("MYSQL_PASSWORD", ""),
			Database: getEnv("MYSQL_DATABASE", "lylme"),
			Charset:  getEnv("MYSQL_CHARSET", "utf8mb4"),
		},
	}

	if c.AdminPath == "" {
		c.AdminPath = "admin"
	}
	return c, nil
}

// Addr 返回监听地址
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Path 返回项目根目录下的路径
func (c *Config) Path(elem ...string) string {
	return filepath.Join(append([]string{c.RootDir}, elem...)...)
}

// LogPath 日志目录绝对路径
func (c *Config) LogPath() string {
	return c.Path(c.LogDir)
}

// SQLiteFullPath SQLite 数据库文件绝对路径
func (c *Config) SQLiteFullPath() string {
	if filepath.IsAbs(c.SQLitePath) {
		return c.SQLitePath
	}
	return c.Path(c.SQLitePath)
}

// DSN 返回最终使用的数据库连接串
func (c *Config) DSN() string {
	if c.DBDSN != "" {
		return c.DBDSN
	}
	if c.DBDriver == "mysql" {
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
			c.MySQL.User, c.MySQL.Password, c.MySQL.Host, c.MySQL.Port, c.MySQL.Database, c.MySQL.Charset)
	}
	return c.SQLiteFullPath()
}

// loadDotEnv 依次尝试从当前目录和可执行文件目录加载 .env
func loadDotEnv() {
	var files []string
	if wd, err := os.Getwd(); err == nil {
		files = append(files, filepath.Join(wd, ".env"))
	}
	if exe, err := os.Executable(); err == nil {
		files = append(files, filepath.Join(filepath.Dir(exe), ".env"))
	}
	for _, f := range files {
		if _, err := os.Stat(f); err == nil {
			_ = godotenv.Load(f)
		}
	}
}

// resolveRootDir 解析项目根目录：
// 1. APP_ROOT 指定优先
// 2. 否则取可执行文件所在目录（go run 的临时目录则回退到当前工作目录）
func resolveRootDir() string {
	if v := os.Getenv("APP_ROOT"); v != "" {
		if abs, err := filepath.Abs(v); err == nil {
			return abs
		}
		return v
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if strings.Contains(dir, "go-build") || strings.Contains(dir, "Temporary") {
			if wd, err := os.Getwd(); err == nil {
				return wd
			}
		}
		return dir
	}
	wd, _ := os.Getwd()
	return wd
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

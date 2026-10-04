package install

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	htmltmpl "html/template"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"lylme_spage/internal/config"
	"lylme_spage/internal/database"
	"lylme_spage/internal/logger"
	"lylme_spage/internal/models"
)

// Handler 安装向导处理器
type Handler struct {
	cfg  *config.Config
	log  *logger.Logger
	tmpl *htmltmpl.Template
}

// pageData 安装向导页面上下文
type pageData struct {
	Copyright string
	Step      int
	Err       string

	OS              string
	GoVersion       string
	RootWritable    bool
	InstallWritable bool

	DBType     string
	SQLitePath string
	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPwd      string

	AdminUser string
	AdminPwd  string

	SiteURL   string
	AdminURL  string
	AdminPath string
}

// New 创建安装向导处理器并解析 web/install 下的页面模板
func New(cfg *config.Config, log *logger.Logger) (*Handler, error) {
	tmpl, err := htmltmpl.ParseGlob(cfg.Path("web", "install", "*.html"))
	if err != nil {
		return nil, fmt.Errorf("解析安装向导模板失败: %w", err)
	}
	return &Handler{cfg: cfg, log: log, tmpl: tmpl}, nil
}

// Register 注册安装向导路由
func (h *Handler) Register(r *gin.Engine, basePath string) {
	base := strings.TrimRight(basePath, "/")
	if base == "" {
		base = "/install"
	}
	r.GET(base, func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, base+"/") })
	r.GET(base+"/", h.index)
	r.GET(base+"/index.php", h.index)
	r.POST(base+"/index.php", h.index)
	r.POST(base+"/save", h.index)
	r.GET(base+"/templates/*filepath", h.staticHandler())
}

// index 安装向导入口：根据 step 参数分发
func (h *Handler) index(c *gin.Context) {
	// 已安装则一律展示"已安装"页，禁止重复安装
	if Installed(h.cfg.RootDir) {
		h.render(c, "installed.html", h.data(c))
		return
	}

	isSave := strings.HasSuffix(strings.TrimRight(c.Request.URL.Path, "/"), "/save")
	step := h.stepParam(c)
	if isSave {
		step = 3
	}

	if c.Request.Method == http.MethodPost && step == 3 {
		h.execute(c)
		return
	}

	switch step {
	case 1:
		h.render(c, "step_1.html", h.data(c))
	case 2:
		h.render(c, "step_2.html", h.data(c))
	default:
		h.render(c, "step_0.html", h.data(c))
	}
}

// stepParam 读取当前步骤（兼容 step 与 s 两种参数名）
func (h *Handler) stepParam(c *gin.Context) int {
	for _, key := range []string{"step", "s"} {
		if v := strings.TrimSpace(c.Query(key)); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				return n
			}
		}
	}
	for _, key := range []string{"step", "s"} {
		if v := strings.TrimSpace(c.PostForm(key)); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				return n
			}
		}
	}
	return 0
}

// data 构建页面上下文（表单回填优先取提交值，其次取当前配置）
func (h *Handler) data(c *gin.Context) *pageData {
	d := &pageData{
		Copyright: "© 2022-" + strconv.Itoa(time.Now().Year()) + " LYLME",
		Step:      h.stepParam(c),
		OS:        runtime.GOOS,
		GoVersion: runtime.Version(),
		SiteURL:   siteURL(c),
		AdminPath: h.cfg.AdminPath,
	}
	d.RootWritable = writable(h.cfg.RootDir)
	d.InstallWritable = writable(h.cfg.Path("install"))

	d.DBType = firstNonEmpty(c.PostForm("db_type"), h.cfg.DBDriver, "sqlite")
	d.SQLitePath = firstNonEmpty(c.PostForm("sqlite_path"), h.cfg.SQLitePath, "data/lylme.db")
	d.DBHost = firstNonEmpty(c.PostForm("dbhost"), h.cfg.MySQL.Host, "127.0.0.1")
	d.DBPort = firstNonEmpty(c.PostForm("dbport"), strconv.Itoa(h.cfg.MySQL.Port), "3306")
	d.DBName = firstNonEmpty(c.PostForm("dbname"), h.cfg.MySQL.Database)
	d.DBUser = firstNonEmpty(c.PostForm("dbuser"), h.cfg.MySQL.User)
	d.DBPwd = c.PostForm("dbpwd")

	d.AdminUser = firstNonEmpty(c.PostForm("admin_user"), "admin")
	d.AdminPwd = firstNonEmpty(c.PostForm("admin_pwd"), "123456")
	d.AdminURL = strings.TrimRight(d.SiteURL, "/") + "/" + h.cfg.AdminPath
	return d
}

// execute 执行安装：写入 .env → 建表导入种子 → 创建管理员 → 生成安装锁
func (h *Handler) execute(c *gin.Context) {
	dbType := strings.ToLower(firstNonEmpty(c.PostForm("db_type"), h.cfg.DBDriver, "sqlite"))
	adminUser := strings.TrimSpace(firstNonEmpty(c.PostForm("admin_user"), "admin"))
	adminPwd := firstNonEmpty(c.PostForm("admin_pwd"), "123456")

	if adminUser == "" {
		h.failStep(c, 2, "管理员账号不能为空！")
		return
	}
	if utf8.RuneCountInString(adminPwd) < 6 {
		h.failStep(c, 2, "管理员密码长度不能小于6位！")
		return
	}

	var driver, dsn string
	values := map[string]string{}

	sqlitePath := strings.TrimSpace(firstNonEmpty(c.PostForm("sqlite_path"), h.cfg.SQLitePath, "data/lylme.db"))
	switch dbType {
	case "mysql":
		host := strings.TrimSpace(c.PostForm("dbhost"))
		name := strings.TrimSpace(c.PostForm("dbname"))
		user := strings.TrimSpace(c.PostForm("dbuser"))
		pwd := c.PostForm("dbpwd")
		port := 3306
		if v, err := strconv.Atoi(strings.TrimSpace(c.PostForm("dbport"))); err == nil && v > 0 {
			port = v
		}
		if host == "" || name == "" || user == "" {
			h.failStep(c, 1, "请完整填写数据库服务器、数据库名与数据库用户名！")
			return
		}
		if !dbNameRe.MatchString(name) {
			h.failStep(c, 1, "数据库名称不合法，仅允许字母、数字、下划线！")
			return
		}
		charset := firstNonEmpty(h.cfg.MySQL.Charset, "utf8mb4")
		driver = "mysql"
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
			user, pwd, host, port, name, charset)
		values["DB_DRIVER"] = "mysql"
		values["MYSQL_HOST"] = host
		values["MYSQL_PORT"] = strconv.Itoa(port)
		values["MYSQL_USER"] = user
		values["MYSQL_PASSWORD"] = pwd
		values["MYSQL_DATABASE"] = name
		values["MYSQL_CHARSET"] = charset
		values["DB_DSN"] = ""
		// 尽力兼容原版行为：目标库不存在时先创建
		h.ensureMySQLDatabase(host, port, user, pwd, name, charset)
	default:
		if sqlitePath == "" {
			h.failStep(c, 1, "请填写 SQLite 数据库文件路径！")
			return
		}
		driver = "sqlite"
		abs := sqlitePath
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(h.cfg.RootDir, abs)
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			h.failStep(c, 1, "创建数据库目录失败："+err.Error())
			return
		}
		dsn = abs
		values["DB_DRIVER"] = "sqlite"
		values["SQLITE_PATH"] = sqlitePath
		values["DB_DSN"] = ""
	}

	// 应用级配置（沿用当前配置，保证其它键不丢失）
	values["APP_ENV"] = h.cfg.AppEnv
	values["SERVER_HOST"] = h.cfg.Host
	values["SERVER_PORT"] = strconv.Itoa(h.cfg.Port)
	values["SERVER_MODE"] = h.cfg.Mode
	values["ADMIN_PATH"] = h.cfg.AdminPath
	values["LOG_DIR"] = h.cfg.LogDir
	values["LOG_LEVEL"] = h.cfg.LogLevel
	values["CDN_PUBLIC"] = h.cfg.CDNPublic
	if dbType != "mysql" {
		values["MYSQL_HOST"] = h.cfg.MySQL.Host
		values["MYSQL_PORT"] = strconv.Itoa(h.cfg.MySQL.Port)
		values["MYSQL_USER"] = h.cfg.MySQL.User
		values["MYSQL_PASSWORD"] = h.cfg.MySQL.Password
		values["MYSQL_DATABASE"] = h.cfg.MySQL.Database
		values["MYSQL_CHARSET"] = firstNonEmpty(h.cfg.MySQL.Charset, "utf8mb4")
	}

	// 1) 先写 .env，保证安装完成后重启即可使用
	if err := writeEnv(h.cfg.RootDir, values); err != nil {
		h.log.Errorf("写入 .env 失败: %v", err)
		h.failStep(c, 1, "写入 .env 文件失败："+err.Error())
		return
	}

	// 2) 连接数据库并建表、导入种子数据
	db, err := database.OpenDSN(driver, dsn, h.log)
	if err != nil {
		h.log.Errorf("安装时数据库连接失败: %v", err)
		h.failStep(c, 1, "数据库连接失败："+err.Error())
		return
	}
	if err := database.MigrateAndSeed(db, h.log); err != nil {
		h.log.Errorf("安装时数据库初始化失败: %v", err)
		h.failStep(c, 1, "数据库初始化失败："+err.Error())
		return
	}

	// 3) 创建管理员账号（沿用 md5('lylme' + 明文) 存储）
	sum := md5.Sum([]byte("lylme" + adminPwd))
	if err := upsertConfig(db, "admin_user", adminUser, "管理员账号"); err != nil {
		h.failStep(c, 2, "设置管理员账号失败："+err.Error())
		return
	}
	if err := upsertConfig(db, "admin_pwd", hex.EncodeToString(sum[:]), "管理员密码"); err != nil {
		h.failStep(c, 2, "设置管理员密码失败："+err.Error())
		return
	}
	if err := upsertConfig(db, "template", currentTemplate(db), "网站主题"); err != nil {
		h.failStep(c, 2, "设置网站主题失败："+err.Error())
		return
	}

	// 4) 生成安装锁
	if err := writeLock(h.cfg.RootDir); err != nil {
		h.log.Errorf("写入安装锁失败: %v", err)
		h.failStep(c, 2, "写入 install/install.lock 失败："+err.Error())
		return
	}

	h.log.Infof("安装完成，管理员账号 %s，数据库驱动 %s", adminUser, driver)

	d := h.data(c)
	d.Step = 3
	d.AdminUser = adminUser
	d.AdminPwd = adminPwd
	d.AdminURL = strings.TrimRight(d.SiteURL, "/") + "/" + h.cfg.AdminPath
	h.render(c, "step_3.html", d)
}

// ensureMySQLDatabase 尽力在安装前创建目标数据库（失败不阻断，后续连接会再次报错）
func (h *Handler) ensureMySQLDatabase(host string, port int, user, pwd, name, charset string) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=%s&parseTime=True&loc=Local", user, pwd, host, port, charset)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		h.log.Warnf("MySQL 建库连接失败（忽略）：%v", err)
		return
	}
	if err := db.Exec("CREATE DATABASE IF NOT EXISTS `" + name + "` DEFAULT CHARACTER SET " + charset).Error; err != nil {
		h.log.Warnf("创建 MySQL 数据库失败（忽略）：%v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

// upsertConfig 写入/更新 lylme_config 中的配置项
func upsertConfig(db *gorm.DB, key, value, desc string) error {
	var row models.Config
	err := db.Where("k = ?", key).First(&row).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return db.Create(&models.Config{K: key, V: value, Description: desc}).Error
	case err != nil:
		return err
	default:
		row.V = value
		if desc != "" {
			row.Description = desc
		}
		return db.Save(&row).Error
	}
}

// currentTemplate 读取当前主题配置，缺省为 default
func currentTemplate(db *gorm.DB) string {
	var row models.Config
	if err := db.Where("k = ?", "template").First(&row).Error; err == nil && row.V != "" {
		return row.V
	}
	return "default"
}

// writeLock 生成 install/install.lock
func writeLock(root string) error {
	path := filepath.Join(root, LockRel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content := "程序已正确安装，重新安装请删除本文件\n安装时间: " + time.Now().Format("2006-01-02 15:04:05") + "\n"
	return os.WriteFile(path, []byte(content), 0o644)
}

// render 渲染安装向导模板
func (h *Handler) render(c *gin.Context, name string, d *pageData) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(c.Writer, name, d); err != nil {
		h.log.Errorf("安装向导渲染失败 %s: %v", name, err)
		c.String(http.StatusInternalServerError, "页面渲染失败: %v", err)
	}
}

// failStep 渲染指定步骤并附带错误提示
func (h *Handler) failStep(c *gin.Context, step int, msg string) {
	d := h.data(c)
	d.Step = step
	d.Err = msg
	switch step {
	case 1:
		h.render(c, "step_1.html", d)
	case 2:
		h.render(c, "step_2.html", d)
	default:
		h.render(c, "step_0.html", d)
	}
}

// staticHandler 提供 install/templates 下的静态资源，屏蔽 .php 源码
func (h *Handler) staticHandler() gin.HandlerFunc {
	root := h.cfg.Path("install", "templates")
	return func(c *gin.Context) {
		rel := strings.TrimPrefix(c.Param("filepath"), "/")
		if rel == "" || strings.Contains(rel, "..") {
			c.Status(http.StatusNotFound)
			return
		}
		if strings.HasSuffix(strings.ToLower(rel), ".php") {
			c.Status(http.StatusNotFound)
			return
		}
		full := filepath.Join(root, filepath.Clean("/"+rel))
		fi, err := os.Stat(full)
		if err != nil || fi.IsDir() {
			c.Status(http.StatusNotFound)
			return
		}
		http.ServeFile(c.Writer, c.Request, full)
	}
}

var dbNameRe = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// writable 判断目录（或不存在时其最近存在的父目录）是否可写
func writable(path string) bool {
	dir := path
	for {
		fi, err := os.Stat(dir)
		if err == nil {
			if !fi.IsDir() {
				return false
			}
			return unixWritable(fi)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

func unixWritable(fi os.FileInfo) bool {
	return fi.Mode().Perm()&0o200 != 0
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// siteURL 依据请求推导站点根地址
func siteURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if p := c.GetHeader("X-Forwarded-Proto"); p != "" {
		if i := strings.Index(p, ","); i >= 0 {
			p = p[:i]
		}
		if v := strings.ToLower(strings.TrimSpace(p)); v != "" {
			scheme = v
		}
	}
	host := strings.TrimRight(c.Request.Host, "/")
	if host == "" {
		host = "localhost"
	}
	return scheme + "://" + host
}

package database

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"lylme_spage/internal/config"
	"lylme_spage/internal/logger"
	"lylme_spage/internal/models"
)

// Open 根据配置打开数据库（默认 SQLite，可切换 MySQL），并自动迁移与初始化种子数据
func Open(cfg *config.Config, log *logger.Logger) (*gorm.DB, error) {
	gcfg := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	}

	var db *gorm.DB
	var err error

	switch cfg.DBDriver {
	case "mysql":
		db, err = gorm.Open(mysql.Open(cfg.DSN()), gcfg)
		if err != nil {
			return nil, fmt.Errorf("MySQL 连接失败: %w", err)
		}
		log.Infof("已连接 MySQL 数据库 %s@%s:%d/%s", cfg.MySQL.User, cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.Database)
	default:
		db, err = openSQLite(cfg, gcfg)
		if err != nil {
			return nil, err
		}
		log.Infof("已连接 SQLite 数据库 %s", cfg.SQLiteFullPath())
	}

	if err := MigrateAndSeed(db, log); err != nil {
		return nil, err
	}
	return db, nil
}

// OpenDSN 使用给定驱动与 DSN 打开数据库连接（仅建立连接，不迁移、不导入种子数据）。
// driver 取值 sqlite|mysql；sqlite 的 dsn 既可以是文件路径，也可以是完整连接串。
func OpenDSN(driver, dsn string, log *logger.Logger) (*gorm.DB, error) {
	gcfg := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	}
	if strings.ToLower(driver) == "mysql" {
		db, err := gorm.Open(mysql.Open(dsn), gcfg)
		if err != nil {
			return nil, fmt.Errorf("MySQL 连接失败: %w", err)
		}
		log.Infof("已连接 MySQL 数据库")
		return db, nil
	}
	db, err := openSQLiteDSN(dsn, gcfg)
	if err != nil {
		return nil, err
	}
	log.Infof("已连接 SQLite 数据库 %s", dsn)
	return db, nil
}

// MigrateAndSeed 执行表结构自动迁移并导入种子数据
func MigrateAndSeed(db *gorm.DB, log *logger.Logger) error {
	if err := db.AutoMigrate(models.AllModels()...); err != nil {
		return fmt.Errorf("数据库表结构迁移失败: %w", err)
	}
	if err := Seed(db, log); err != nil {
		return fmt.Errorf("数据库种子数据初始化失败: %w", err)
	}
	return nil
}

func openSQLite(cfg *config.Config, gcfg *gorm.Config) (*gorm.DB, error) {
	if cfg.DBDSN != "" {
		db, err := gorm.Open(sqlite.Open(cfg.DBDSN), gcfg)
		if err != nil {
			return nil, fmt.Errorf("SQLite 连接失败: %w", err)
		}
		return db, nil
	}
	return openSQLiteDSN(cfg.SQLiteFullPath(), gcfg)
}

// openSQLiteDSN 打开 SQLite：dsn 为文件路径时补全目录并追加兼容用的 PRAGMA 参数
func openSQLiteDSN(dsn string, gcfg *gorm.Config) (*gorm.DB, error) {
	if !strings.Contains(dsn, "?") {
		if err := os.MkdirAll(filepath.Dir(dsn), 0o755); err != nil {
			return nil, fmt.Errorf("创建数据库目录失败: %w", err)
		}
		dsn = dsn + "?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	}
	db, err := gorm.Open(sqlite.Open(dsn), gcfg)
	if err != nil {
		return nil, fmt.Errorf("SQLite 连接失败: %w", err)
	}
	return db, nil
}

package database

import (
	"fmt"
	"os"
	"path/filepath"

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

	if err := db.AutoMigrate(models.AllModels()...); err != nil {
		return nil, fmt.Errorf("数据库表结构迁移失败: %w", err)
	}
	if err := Seed(db, log); err != nil {
		return nil, fmt.Errorf("数据库种子数据初始化失败: %w", err)
	}
	return db, nil
}

func openSQLite(cfg *config.Config, gcfg *gorm.Config) (*gorm.DB, error) {
	dsn := cfg.DBDSN
	if dsn == "" {
		path := cfg.SQLiteFullPath()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("创建数据库目录失败: %w", err)
		}
		dsn = path + "?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	}
	db, err := gorm.Open(sqlite.Open(dsn), gcfg)
	if err != nil {
		return nil, fmt.Errorf("SQLite 连接失败: %w", err)
	}
	return db, nil
}

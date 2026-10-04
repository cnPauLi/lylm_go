package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Level 日志级别
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func parseLevel(s string) Level {
	switch s {
	case "debug":
		return LevelDebug
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

func (l Level) tag() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "INFO"
	}
}

// Logger 简单分级日志，输出到控制台与文件
type Logger struct {
	mu    sync.Mutex
	out   io.Writer
	level Level
	file  *os.File
}

// New 创建日志器，日志文件写入 dir/app-YYYYMMDD.log
func New(dir, level string) (*Logger, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %w", err)
	}
	name := filepath.Join(dir, "app-"+time.Now().Format("20060102")+".log")
	f, err := os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %w", err)
	}
	return &Logger{
		out:   io.MultiWriter(os.Stdout, f),
		level: parseLevel(level),
		file:  f,
	}, nil
}

// Close 关闭日志文件
func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		_ = l.file.Close()
	}
}

func (l *Logger) log(lv Level, format string, args ...any) {
	if lv < l.level {
		return
	}
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s [%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), lv.tag(), msg)
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = io.WriteString(l.out, line)
}

// Debugf 调试日志
func (l *Logger) Debugf(format string, args ...any) { l.log(LevelDebug, format, args...) }

// Infof 信息日志
func (l *Logger) Infof(format string, args ...any) { l.log(LevelInfo, format, args...) }

// Warnf 警告日志
func (l *Logger) Warnf(format string, args ...any) { l.log(LevelWarn, format, args...) }

// Errorf 错误日志
func (l *Logger) Errorf(format string, args ...any) { l.log(LevelError, format, args...) }

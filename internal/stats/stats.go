// Package stats 提供与原 lylme_spage/include/tj 等价的访问统计：
// 每日 PV / 独立 IP 日志与累计日志，以及后台仪表盘所需的汇总数据。
package stats

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"lylme_spage/internal/logger"
	"lylme_spage/internal/webutil"
)

// dayData 每日统计文件结构：<logDir>/<YYYYMMDD>.log
type dayData struct {
	PV  int            `json:"pv"`
	IPs map[string]int `json:"ips"`
}

// totalData 累计统计文件结构：<logDir>/total.log
type totalData struct {
	PV  int            `json:"pv"`
	IP  int            `json:"ip"`
	IPs map[string]int `json:"ips"`
}

// Summary 访问统计汇总
type Summary struct {
	Today     int      // 今日浏览量
	Yesterday int      // 昨日浏览量
	Total     int      // 累计浏览量
	TodayIP   int      // 今日独立 IP 数
	TotalIP   int      // 累计独立 IP 数
	Month     int      // 本月浏览量
	Labels    []string // 图表横轴标签（近 N 天）
	PV        []int    // 图表每日浏览量
	IP        []int    // 图表每日独立 IP 数
}

// Store 访问统计存储（并发安全）
type Store struct {
	logDir string
	db     *gorm.DB

	mu  sync.Mutex
	log *logger.Logger
}

// New 创建统计存储。logDir 为日志目录，db 预留（当前统计不依赖数据库）。
func New(logDir string, db *gorm.DB) *Store {
	_ = os.MkdirAll(logDir, 0o755)
	return &Store{logDir: logDir, db: db}
}

// SetLogger 设置日志器，用于记录统计写入异常
func (s *Store) SetLogger(l *logger.Logger) { s.log = l }

// Record 记录一次访问：分别累加每日文件与累计文件。
// 该方法是尽力而为的，写入失败不会向外抛出，仅记录日志。
func (s *Store) Record(ip string) {
	if s == nil || ip == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if err := s.recordDay(now.Format("20060102"), ip); err != nil {
		if s.log != nil {
			s.log.Warnf("写入每日访问统计失败: %v", err)
		}
	}
	if err := s.recordTotal(ip); err != nil {
		if s.log != nil {
			s.log.Warnf("写入累计访问统计失败: %v", err)
		}
	}
}

// Middleware 记录非后台页面请求的访问量（在处理器执行之后统计）。
func (s *Store) Middleware(adminPath string) gin.HandlerFunc {
	skipPrefixes := []string{"/assets", "/template", "/site/static", "/files", "/include", "/api", "/mcp"}
	return func(c *gin.Context) {
		c.Next()
		if s == nil {
			return
		}
		path := c.Request.URL.Path
		if adminPath != "" && strings.Contains(path, adminPath) {
			return
		}
		for _, p := range skipPrefixes {
			if strings.HasPrefix(path, p) {
				return
			}
		}
		if path == "/healthz" || path == "/sitemap.xml" {
			return
		}
		ip := webutil.RealIP(c.Request)
		if ip == "" {
			return
		}
		s.Record(ip)
	}
}

// Summary 读取统计汇总。days 为图表天数（近 N 天，含今天），不足 1 时取 7。
func (s *Store) Summary(days int) Summary {
	if days < 1 {
		days = 7
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	var out Summary

	today := s.readDay(now.Format("20060102"))
	out.Today = today.PV
	out.TodayIP = len(today.IPs)

	out.Yesterday = s.readDay(now.AddDate(0, 0, -1).Format("20060102")).PV

	total := s.readTotal()
	out.Total = total.PV
	out.TotalIP = total.IP
	if out.TotalIP == 0 {
		out.TotalIP = len(total.IPs)
	}

	monthPrefix := now.Format("200601")
	for d := 1; d <= now.Day(); d++ {
		out.Month += s.readDay(fmt.Sprintf("%s%02d", monthPrefix, d)).PV
	}

	out.Labels = make([]string, 0, days)
	out.PV = make([]int, 0, days)
	out.IP = make([]int, 0, days)
	for i := days - 1; i >= 0; i-- {
		t := now.AddDate(0, 0, -i)
		var label string
		switch i {
		case 0:
			label = "今天"
		case 1:
			label = "昨天"
		case 2:
			label = "前天"
		default:
			label = t.Format("01-02")
		}
		d := s.readDay(t.Format("20060102"))
		out.Labels = append(out.Labels, label)
		out.PV = append(out.PV, d.PV)
		out.IP = append(out.IP, len(d.IPs))
	}
	return out
}

// recordDay 累加某一天的 PV 与 IP 计数
func (s *Store) recordDay(key, ip string) error {
	file := filepath.Join(s.logDir, key+".log")
	d := s.readDay(key)
	d.PV++
	d.IPs[ip]++
	return saveJSON(file, d)
}

// recordTotal 累加累计 PV 与 IP 计数
func (s *Store) recordTotal(ip string) error {
	file := filepath.Join(s.logDir, "total.log")
	t := s.readTotal()
	t.PV++
	t.IPs[ip]++
	t.IP = len(t.IPs)
	return saveJSON(file, t)
}

// readDay 读取每日统计文件（缺失或损坏时返回空默认值）
func (s *Store) readDay(key string) dayData {
	d := dayData{IPs: map[string]int{}}
	loadJSON(filepath.Join(s.logDir, key+".log"), &d)
	if d.IPs == nil {
		d.IPs = map[string]int{}
	}
	return d
}

// readTotal 读取累计统计文件（缺失或损坏时返回空默认值）
func (s *Store) readTotal() totalData {
	t := totalData{IPs: map[string]int{}}
	loadJSON(filepath.Join(s.logDir, "total.log"), &t)
	if t.IPs == nil {
		t.IPs = map[string]int{}
	}
	return t
}

// loadJSON 读取 JSON 文件；失败时保留默认值
func loadJSON(file string, v any) {
	b, err := os.ReadFile(file)
	if err != nil {
		return
	}
	_ = json.Unmarshal(b, v)
}

// saveJSON 以「临时文件 + rename」原子写入 JSON 文件
func saveJSON(file string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tmp := file + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}

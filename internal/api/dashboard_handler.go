package api

import (
	"github.com/gin-gonic/gin"

	"lylme_spage/internal/service"
	"lylme_spage/internal/stats"
)

// visitsResp 近 7 天访问统计（图表结构，与 stats.Summary 对应）。
type visitsResp struct {
	Today     int      `json:"today"`     // 今日浏览量
	Yesterday int      `json:"yesterday"` // 昨日浏览量
	Total     int      `json:"total"`     // 累计浏览量
	TodayIP   int      `json:"today_ip"`  // 今日独立 IP 数
	TotalIP   int      `json:"total_ip"`  // 累计独立 IP 数
	Month     int      `json:"month"`     // 本月浏览量
	Labels    []string `json:"labels"`    // 近 7 天横轴标签
	PV        []int    `json:"pv"`        // 近 7 天每日浏览量
	IP        []int    `json:"ip"`        // 近 7 天每日独立 IP 数
}

func toVisitsResp(s stats.Summary) visitsResp {
	return visitsResp{
		Today:     s.Today,
		Yesterday: s.Yesterday,
		Total:     s.Total,
		TodayIP:   s.TodayIP,
		TotalIP:   s.TotalIP,
		Month:     s.Month,
		Labels:    nonNilStrings(s.Labels),
		PV:        nonNilInts(s.PV),
		IP:        nonNilInts(s.IP),
	}
}

func emptyVisitsResp() visitsResp {
	return visitsResp{Labels: []string{}, PV: []int{}, IP: []int{}}
}

func nonNilStrings(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

func nonNilInts(v []int) []int {
	if v == nil {
		return []int{}
	}
	return v
}

// dashboardSummaryResp GET /dashboard 出参：计数 + 版本信息。
// 字段由 service.DashboardSummary 映射，REST 层固定 snake_case 契约。
type dashboardSummaryResp struct {
	Links      int64       `json:"links"`       // 链接总数
	Groups     int64       `json:"groups"`      // 分组总数
	ApplyRows  int64       `json:"apply_rows"`  // 待审申请数（apply_status=0）
	AppVersion string      `json:"app_version"` // 程序版本
	DBVersion  string      `json:"db_version"`  // 数据库版本（配置键 version）
	Build      string      `json:"build"`       // 建站时间（配置键 build）
	Visits     visitsResp  `json:"visits"`      // 近 7 天访问统计
}

// cleanupFileResp 未使用文件单项
type cleanupFileResp struct {
	Rel     string `json:"rel"`      // 相对 files 的路径，如 download/a.png
	Name    string `json:"name"`     // 文件名
	Dir     string `json:"dir"`      // 所在目录 download/upload
	URL     string `json:"url"`      // 访问 URL，/files/<rel>
	Ext     string `json:"ext"`      // 小写扩展名（无点）
	SizeStr string `json:"size_str"` // 人类可读大小
	Size    int64  `json:"size"`     // 字节数
	IsImg   bool   `json:"is_img"`   // 是否为图片
}

// cleanupDirResp 单目录统计（Files 仅列出未使用文件）
type cleanupDirResp struct {
	Name   string            `json:"name"`   // download/upload
	Unused int               `json:"unused"` // 未使用文件数
	Files  []cleanupFileResp `json:"files"`  // 未使用文件列表
}

// cleanupReportResp GET /files/cleanup 扫描报告
type cleanupReportResp struct {
	Total        int              `json:"total"`          // 文件总数（含已使用）
	Used         int              `json:"used"`           // 被图标引用的文件数
	Unused       int              `json:"unused"`         // 未使用文件数
	TotalSize    int64            `json:"total_size"`     // 未使用文件总字节数
	TotalSizeStr string           `json:"total_size_str"` // 未使用文件总大小（人类可读）
	Dirs         []cleanupDirResp `json:"dirs"`           // 分目录报告（download/upload）
}

// cleanupDeleteReq POST /files/cleanup 入参：待删除文件的相对路径列表
type cleanupDeleteReq struct {
	// 每项须匹配 ^(download|upload)/[^/]+$；非法项计入 failed，不删除任何文件。
	Rels []string `json:"rels" binding:"required,min=1"`
}

// cleanupDeleteResp POST /files/cleanup 出参：删除计数
type cleanupDeleteResp struct {
	Deleted int `json:"deleted"` // 成功删除数
	Failed  int `json:"failed"`  // 失败数（含非法路径、不存在的文件）
}

// RegisterDashboardRoutes 在 /api/v1 下注册 dashboard / files 域端点：
//   - GET    /dashboard       仪表盘计数与版本信息
//   - GET    /files/cleanup   扫描未使用文件报告
//   - POST   /files/cleanup   按 rel 列表删除文件
//
// 鉴权与限流由挂载 g 的上层路由组统一处理。
func RegisterDashboardRoutes(g gin.IRouter, d *Deps) {
	g.GET("/dashboard", func(c *gin.Context) {
		sum := d.Dashboard.Summary(d.AppVersion)
		vis := emptyVisitsResp()
		if d.Stats != nil {
			vis = toVisitsResp(d.Stats.Summary(7))
		}
		OK(c, dashboardSummaryResp{
			Links:      sum.Links,
			Groups:     sum.Groups,
			ApplyRows:  sum.ApplyRows,
			AppVersion: sum.AppVersion,
			DBVersion:  sum.DBVersion,
			Build:      sum.Build,
			Visits:     vis,
		})
	})

	g.GET("/files/cleanup", func(c *gin.Context) {
		report, err := d.Files.Scan()
		if err != nil {
			WriteServiceError(c, err)
			return
		}
		OK(c, toCleanupReportResp(report))
	})

	g.POST("/files/cleanup", func(c *gin.Context) {
		var req cleanupDeleteReq
		if err := c.ShouldBindJSON(&req); err != nil {
			FailBadRequest(c, "请求体无效：rels 为必填项且至少包含 1 个路径")
			return
		}
		res := d.Files.Delete(req.Rels)
		OK(c, cleanupDeleteResp{Deleted: res.Deleted, Failed: res.Failed})
	})
}

// toCleanupReportResp 将 service 报告映射为 REST 出参，空列表归一为 [] 而非 null。
func toCleanupReportResp(r *service.CleanupReport) cleanupReportResp {
	out := cleanupReportResp{
		Total:        r.Total,
		Used:         r.Used,
		Unused:       r.Unused,
		TotalSize:    r.TotalSize,
		TotalSizeStr: r.TotalSizeStr,
		Dirs:         make([]cleanupDirResp, 0, len(r.Dirs)),
	}
	for _, d := range r.Dirs {
		dir := cleanupDirResp{
			Name:   d.Name,
			Unused: d.Unused,
			Files:  make([]cleanupFileResp, 0, len(d.Files)),
		}
		for _, f := range d.Files {
			dir.Files = append(dir.Files, cleanupFileResp{
				Rel:     f.Rel,
				Name:    f.Name,
				Dir:     f.Dir,
				URL:     f.URL,
				Ext:     f.Ext,
				SizeStr: f.SizeStr,
				Size:    f.Size,
				IsImg:   f.IsImg,
			})
		}
		out.Dirs = append(out.Dirs, dir)
	}
	return out
}

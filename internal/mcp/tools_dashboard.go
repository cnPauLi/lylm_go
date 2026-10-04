package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	"lylme_spage/internal/service"
)

// mcpVisits 近 7 天访问统计（结构与 REST visitsResp 一致）
type mcpVisits struct {
	Today     int      `json:"today"`
	Yesterday int      `json:"yesterday"`
	Total     int      `json:"total"`
	TodayIP   int      `json:"today_ip"`
	TotalIP   int      `json:"total_ip"`
	Month     int      `json:"month"`
	Labels    []string `json:"labels"`
	PV        []int    `json:"pv"`
	IP        []int    `json:"ip"`
}

// mcpDashboardSummary 仪表盘汇总（结构与 REST dashboardSummaryResp 一致）
type mcpDashboardSummary struct {
	Links      int64          `json:"links"`
	Groups     int64          `json:"groups"`
	ApplyRows  int64          `json:"apply_rows"`
	AppVersion string         `json:"app_version"`
	DBVersion  string         `json:"db_version"`
	Build      string         `json:"build"`
	Visits     mcpVisits      `json:"visits"`
}

// mcpCleanupFile 未使用文件单项（结构与 REST cleanupFileResp 一致）
type mcpCleanupFile struct {
	Rel     string `json:"rel"`
	Name    string `json:"name"`
	Dir     string `json:"dir"`
	URL     string `json:"url"`
	Ext     string `json:"ext"`
	SizeStr string `json:"size_str"`
	Size    int64  `json:"size"`
	IsImg   bool   `json:"is_img"`
}

// mcpCleanupDir 单目录统计
type mcpCleanupDir struct {
	Name   string           `json:"name"`
	Unused int              `json:"unused"`
	Files  []mcpCleanupFile `json:"files"`
}

// mcpCleanupReport 文件清理扫描报告
type mcpCleanupReport struct {
	Total        int             `json:"total"`
	Used         int             `json:"used"`
	Unused       int             `json:"unused"`
	TotalSize    int64           `json:"total_size"`
	TotalSizeStr string          `json:"total_size_str"`
	Dirs         []mcpCleanupDir `json:"dirs"`
}

// mcpCleanupDelete 删除结果
type mcpCleanupDelete struct {
	Deleted int `json:"deleted"`
	Failed  int `json:"failed"`
}

func toMcpCleanupReport(r *service.CleanupReport) mcpCleanupReport {
	out := mcpCleanupReport{
		Total:        r.Total,
		Used:         r.Used,
		Unused:       r.Unused,
		TotalSize:    r.TotalSize,
		TotalSizeStr: r.TotalSizeStr,
		Dirs:         make([]mcpCleanupDir, 0, len(r.Dirs)),
	}
	for _, d := range r.Dirs {
		dir := mcpCleanupDir{
			Name:   d.Name,
			Unused: d.Unused,
			Files:  make([]mcpCleanupFile, 0, len(d.Files)),
		}
		for _, f := range d.Files {
			dir.Files = append(dir.Files, mcpCleanupFile{
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

// registerDashboardTools 注册仪表盘与文件清理域工具，
// 与 GET /api/v1/dashboard、GET/POST /api/v1/files/cleanup 一一对应。
func (set *MCPSet) registerDashboardTools() {
	d := set.deps

	// dashboard_summary：GET /dashboard
	set.add(mcp.NewTool("dashboard_summary",
		mcp.WithDescription("查询仪表盘汇总：链接总数、分组总数、待审核申请数、程序版本、数据库版本、建站时间与近 7 天访问统计。"),
	), func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sum := d.Dashboard.Summary(d.AppVersion)
		vis := mcpVisits{Labels: []string{}, PV: []int{}, IP: []int{}}
		if d.Stats != nil {
			s := d.Stats.Summary(7)
			vis = mcpVisits{
				Today:     s.Today,
				Yesterday: s.Yesterday,
				Total:     s.Total,
				TodayIP:   s.TodayIP,
				TotalIP:   s.TotalIP,
				Month:     s.Month,
				Labels:    s.Labels,
				PV:        s.PV,
				IP:        s.IP,
			}
			if vis.Labels == nil {
				vis.Labels = []string{}
			}
			if vis.PV == nil {
				vis.PV = []int{}
			}
			if vis.IP == nil {
				vis.IP = []int{}
			}
		}
		return jsonResult(mcpDashboardSummary{
			Links:      sum.Links,
			Groups:     sum.Groups,
			ApplyRows:  sum.ApplyRows,
			AppVersion: sum.AppVersion,
			DBVersion:  sum.DBVersion,
			Build:      sum.Build,
			Visits:     vis,
		}), nil
	})

	// files_cleanup_scan：GET /files/cleanup
	set.add(mcp.NewTool("files_cleanup_scan",
		mcp.WithDescription("扫描 files/download 与 files/upload 目录，比对数据库中的图标引用，返回未使用文件报告（含分目录列表与总大小）。"),
	), func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		report, err := d.Files.Scan()
		if err != nil {
			return serviceErrorResult(err), nil
		}
		return jsonResult(toMcpCleanupReport(report)), nil
	})

	// files_cleanup_delete：POST /files/cleanup
	set.add(mcp.NewTool("files_cleanup_delete",
		mcp.WithDescription("按相对路径列表删除未使用文件。仅允许 download/upload 单层目录下的文件且拒绝目录穿越；非法路径计入失败、不删除任何文件。"),
		mcp.WithArray("rels",
			mcp.Required(),
			mcp.Items(map[string]any{"type": "string"}),
			mcp.MinItems(1),
			mcp.Description("待删除文件相对路径列表（至少 1 个），如 [\"upload/a.png\"]"),
		),
	), func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		rels, err := requiredStrings(args, "rels")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if len(rels) == 0 {
			return mcp.NewToolResultError("参数 \"rels\" 至少包含 1 个相对路径"), nil
		}
		res := d.Files.Delete(rels)
		return jsonResult(mcpCleanupDelete{Deleted: res.Deleted, Failed: res.Failed}), nil
	})
}

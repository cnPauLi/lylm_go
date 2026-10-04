package service

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"lylme_spage/internal/config"
	"lylme_spage/internal/models"
	"lylme_spage/internal/site"
)

// CleanupFile 未使用文件单项（字段与原 admin.cleanImgFile 等价）
type CleanupFile struct {
	Rel     string // 相对 files 的路径，如 download/a.png
	Name    string // 文件名
	Dir     string // 所在目录 download/upload
	URL     string // 访问 URL，/files/<rel>
	Ext     string // 小写扩展名（无点）
	SizeStr string // 人类可读大小
	Size    int64  // 字节数
	IsImg   bool   // 是否为图片
}

// CleanupDir 单目录统计（字段与原 admin.cleanImgDir 等价）
type CleanupDir struct {
	Name   string // download/upload
	Unused int    // 未使用文件数
	Files  []CleanupFile
}

// CleanupReport 扫描报告（字段与原 admin.cleanImgPage 等价）
type CleanupReport struct {
	Total        int // 文件总数
	Used         int // 已使用数
	Unused       int // 未使用数
	TotalSize    int64
	TotalSizeStr string
	Dirs         []CleanupDir
}

// DeleteResult 批量删除结果
type DeleteResult struct {
	Deleted int // 成功删除数
	Failed  int // 失败数（含非法路径）
}

// 允许删除的相对路径：仅 download/upload 单层目录下的单个文件；
// 另以 dotRe 拒绝目录穿越（download/.. 可通过 relRe，须二次拦截）。
var cleanupRelRe = regexp.MustCompile(`^(download|upload)/[^/]+$`)
var cleanupDotRe = regexp.MustCompile(`(^|/)\.\.(/|$)`)

// FileCleanupService 文件清理域业务规则：扫描未使用文件、安全删除
type FileCleanupService struct {
	store *site.Store
	cfg   *config.Config
}

// NewFileCleanupService 创建文件清理服务
func NewFileCleanupService(store *site.Store, cfg *config.Config) *FileCleanupService {
	return &FileCleanupService{store: store, cfg: cfg}
}

// Scan 扫描 files/download 与 files/upload：
// 图标引用取自 lylme_groups.group_icon 与 lylme_links.icon，
// 文件名（小写）出现在任一引用（小写）中即判定为已使用；仅未使用文件计入报告。
// 目录不存在时该目录按空统计；DB 引用查询失败时返回错误。
func (s *FileCleanupService) Scan() (*CleanupReport, error) {
	var icons []string
	if err := s.store.DB().Model(&models.Group{}).Where("group_icon <> ''").Pluck("group_icon", &icons).Error; err != nil {
		return nil, err
	}
	var linkIcons []string
	if err := s.store.DB().Model(&models.Link{}).Where("icon <> ''").Pluck("icon", &linkIcons).Error; err != nil {
		return nil, err
	}
	icons = append(icons, linkIcons...)

	report := &CleanupReport{}
	for _, d := range []string{"download", "upload"} {
		dir := CleanupDir{Name: d}
		entries, err := os.ReadDir(s.cfg.Path("files", d))
		if err != nil {
			report.Dirs = append(report.Dirs, dir)
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			report.Total++
			name := e.Name()
			ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
			lname := strings.ToLower(name)
			used := false
			for _, v := range icons {
				if strings.Contains(strings.ToLower(v), lname) {
					used = true
					break
				}
			}
			if used {
				report.Used++
				continue
			}
			report.Unused++
			report.TotalSize += info.Size()
			dir.Unused++
			dir.Files = append(dir.Files, CleanupFile{
				Rel: d + "/" + name, Name: name, Dir: d, URL: "/files/" + d + "/" + name,
				Ext: ext, Size: info.Size(), SizeStr: FormatSize(info.Size()), IsImg: isImageExt(ext),
			})
		}
		report.Dirs = append(report.Dirs, dir)
	}
	report.TotalSizeStr = FormatSize(report.TotalSize)
	return report, nil
}

// Delete 逐条校验并删除文件，返回成功/失败计数：
// rel 须匹配 ^(download|upload)/[^/]+$、不含目录穿越，且目标为已存在的普通文件。
// 非法路径一律不删除（不计入任何文件操作）；单条失败不阻断其余条目。
func (s *FileCleanupService) Delete(rels []string) DeleteResult {
	base := s.cfg.Path("files")
	res := DeleteResult{}
	for _, rel := range rels {
		rel = strings.ReplaceAll(strings.TrimSpace(rel), `\`, "/")
		if !cleanupRelRe.MatchString(rel) || cleanupDotRe.MatchString(rel) {
			res.Failed++
			continue
		}
		full := filepath.Join(base, filepath.FromSlash(rel))
		if fi, err := os.Stat(full); err == nil && !fi.IsDir() {
			if err := os.Remove(full); err == nil {
				res.Deleted++
				continue
			}
		}
		res.Failed++
	}
	return res
}

// FormatSize 人类可读文件大小（与原 admin.fmtSize 等价）
func FormatSize(b int64) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.2f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.2f KB", float64(b)/float64(1<<10))
	default:
		return strconv.FormatInt(b, 10) + " B"
	}
}

// isImageExt 是否为图片扩展名（与原 admin.isImageExt 等价）
func isImageExt(ext string) bool {
	switch ext {
	case "png", "jpg", "jpeg", "gif", "ico", "webp":
		return true
	}
	return false
}

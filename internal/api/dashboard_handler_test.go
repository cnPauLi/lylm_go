package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/service"
	"lylme_spage/internal/stats"
)

// apiReq 发起一次 /api/v1 请求；key 为空表示不带 Authorization；
// body 非空时按 JSON 发送。
func apiReq(t *testing.T, r *gin.Engine, method, path, key, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// mustOKData 断言 200 统一信封并把 data 反序列化到 dst。
func mustOKData(t *testing.T, w *httptest.ResponseRecorder, dst any) {
	t.Helper()
	var env struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("响应非统一信封 JSON: %v; body=%s", err, w.Body.String())
	}
	if w.Code != http.StatusOK || env.Code != CodeOK {
		t.Fatalf("状态码=%d code=%d, want 200/200; body=%s", w.Code, env.Code, w.Body.String())
	}
	if dst != nil {
		if err := json.Unmarshal(env.Data, dst); err != nil {
			t.Fatalf("data 反序列化失败: %v; data=%s", err, env.Data)
		}
	}
}

// writeTestFile 在 root/files 下写入测试文件。
func writeTestFile(t *testing.T, root, rel string, content []byte) {
	t.Helper()
	full := filepath.Join(root, "files", filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	if err := os.WriteFile(full, content, 0o644); err != nil {
		t.Fatalf("写入文件 %s 失败: %v", rel, err)
	}
}

func fileExists(root, rel string) bool {
	_, err := os.Stat(filepath.Join(root, "files", filepath.FromSlash(rel)))
	return err == nil
}

func TestGetDashboard(t *testing.T) {
	register := func(v1 *gin.RouterGroup, d *Deps) { RegisterDashboardRoutes(v1, d) }

	r, d, key := newTestV1(t, register)

	g, err := d.Group.Create(service.GroupInput{Name: "仪表盘分组"})
	if err != nil {
		t.Fatalf("创建分组失败: %v", err)
	}
	if _, err := d.Link.Create(service.LinkInput{
		Name: "仪表盘链接", URL: "https://example.com", GroupID: g.GroupID,
	}); err != nil {
		t.Fatalf("创建链接失败: %v", err)
	}
	if _, err := d.Apply.Create(service.ApplyCreateInput{
		Name: "待审站点", URL: "https://pending.com",
	}); err != nil {
		t.Fatalf("创建申请失败: %v", err)
	}

	w := apiReq(t, r, http.MethodGet, "/api/v1/dashboard", key, "")
	var sum dashboardSummaryResp
	mustOKData(t, w, &sum)

	if sum.Links != 1 {
		t.Fatalf("links = %d, want 1", sum.Links)
	}
	if sum.Groups != 1 {
		t.Fatalf("groups = %d, want 1", sum.Groups)
	}
	if sum.ApplyRows != 1 {
		t.Fatalf("apply_rows = %d, want 1", sum.ApplyRows)
	}
	if sum.AppVersion != "test" {
		t.Fatalf("app_version = %q, want %q", sum.AppVersion, "test")
	}
	if sum.DBVersion != "" {
		t.Fatalf("db_version = %q, want 空串", sum.DBVersion)
	}
	if sum.Build != "" {
		t.Fatalf("build = %q, want 空串", sum.Build)
	}
}

func TestGetFilesCleanupScan(t *testing.T) {
	r, d, key := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) { RegisterDashboardRoutes(v1, d) })
	root := d.Cfg.RootDir

	g, err := d.Group.Create(service.GroupInput{Name: "清理分组"})
	if err != nil {
		t.Fatalf("创建分组失败: %v", err)
	}
	if _, err := d.Link.Create(service.LinkInput{
		Name: "引用图标的链接", URL: "https://used.example.com", GroupID: g.GroupID,
		Icon: "/files/download/used.png",
	}); err != nil {
		t.Fatalf("创建链接失败: %v", err)
	}

	writeTestFile(t, root, "download/used.png", make([]byte, 100))
	writeTestFile(t, root, "download/unused1.png", []byte("x"))
	writeTestFile(t, root, "download/unused2.txt", []byte("hello"))
	writeTestFile(t, root, "upload/up.jpg", make([]byte, 1024))

	w := apiReq(t, r, http.MethodGet, "/api/v1/files/cleanup", key, "")
	var report cleanupReportResp
	mustOKData(t, w, &report)

	if report.Total != 4 {
		t.Fatalf("total = %d, want 4", report.Total)
	}
	if report.Used != 1 {
		t.Fatalf("used = %d, want 1", report.Used)
	}
	if report.Unused != 3 {
		t.Fatalf("unused = %d, want 3", report.Unused)
	}
	// 未使用文件：1 B + 5 B + 1024 B
	if report.TotalSize != 1030 {
		t.Fatalf("total_size = %d, want 1030", report.TotalSize)
	}
	if report.TotalSizeStr != "1.01 KB" {
		t.Fatalf("total_size_str = %q, want %q", report.TotalSizeStr, "1.01 KB")
	}
	if len(report.Dirs) != 2 {
		t.Fatalf("dirs 数量 = %d, want 2", len(report.Dirs))
	}

	dl := report.Dirs[0]
	if dl.Name != "download" || dl.Unused != 2 {
		t.Fatalf("download 目录统计异常: %+v", dl)
	}
	dlRels := map[string]cleanupFileResp{}
	for _, f := range dl.Files {
		dlRels[f.Rel] = f
	}
	if _, ok := dlRels["download/used.png"]; ok {
		t.Fatal("被引用的 used.png 不应出现在未使用列表")
	}
	u1, ok := dlRels["download/unused1.png"]
	if !ok {
		t.Fatal("未使用文件 unused1.png 未被识别")
	}
	if u1.Ext != "png" || !u1.IsImg || u1.Size != 1 || u1.URL != "/files/download/unused1.png" {
		t.Fatalf("unused1.png 元数据异常: %+v", u1)
	}
	if _, ok := dlRels["download/unused2.txt"]; !ok {
		t.Fatal("未使用文件 unused2.txt 未被识别")
	}

	up := report.Dirs[1]
	if up.Name != "upload" || up.Unused != 1 || len(up.Files) != 1 {
		t.Fatalf("upload 目录统计异常: %+v", up)
	}
	if up.Files[0].Rel != "upload/up.jpg" || !up.Files[0].IsImg {
		t.Fatalf("up.jpg 元数据异常: %+v", up.Files[0])
	}
}

func TestPostFilesCleanupDelete(t *testing.T) {
	r, d, key := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) { RegisterDashboardRoutes(v1, d) })
	root := d.Cfg.RootDir

	g, err := d.Group.Create(service.GroupInput{Name: "删除分组"})
	if err != nil {
		t.Fatalf("创建分组失败: %v", err)
	}
	if _, err := d.Link.Create(service.LinkInput{
		Name: "保留图标链接", URL: "https://keep.example.com", GroupID: g.GroupID,
		Icon: "/files/download/used.png",
	}); err != nil {
		t.Fatalf("创建链接失败: %v", err)
	}

	writeTestFile(t, root, "download/used.png", []byte("keep"))
	writeTestFile(t, root, "download/unused1.png", []byte("del1"))
	writeTestFile(t, root, "download/unused2.txt", []byte("del2"))
	writeTestFile(t, root, "upload/up.jpg", []byte("del3"))

	body := `{"rels":["download/unused1.png","upload/up.jpg"]}`
	w := apiReq(t, r, http.MethodPost, "/api/v1/files/cleanup", key, body)
	var res cleanupDeleteResp
	mustOKData(t, w, &res)

	if res.Deleted != 2 {
		t.Fatalf("deleted = %d, want 2", res.Deleted)
	}
	if res.Failed != 0 {
		t.Fatalf("failed = %d, want 0", res.Failed)
	}

	if fileExists(root, "download/unused1.png") {
		t.Fatal("unused1.png 应已被删除")
	}
	if fileExists(root, "upload/up.jpg") {
		t.Fatal("up.jpg 应已被删除")
	}
	if !fileExists(root, "download/used.png") {
		t.Fatal("被链接引用的 used.png 必须保留")
	}
	if !fileExists(root, "download/unused2.txt") {
		t.Fatal("未提交删除的 unused2.txt 必须保留")
	}
}

func TestPostFilesCleanupInvalidRels(t *testing.T) {
	r, d, key := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) { RegisterDashboardRoutes(v1, d) })
	root := d.Cfg.RootDir

	writeTestFile(t, root, "download/a.png", []byte("a"))
	writeTestFile(t, root, "upload/b.jpg", []byte("b"))

	rels := []string{
		"../x",                 // 目录穿越前缀，白名单不匹配
		"etc/passwd",           // 不在 download/upload 白名单
		"download/",            // 缺少文件名
		"download/../x.png",    // 含 .. 穿越
		"download/sub/c.png",   // 多级子目录
		"download/missing.png", // 白名单匹配但文件不存在
	}
	raw, _ := json.Marshal(cleanupDeleteReq{Rels: rels})
	w := apiReq(t, r, http.MethodPost, "/api/v1/files/cleanup", key, string(raw))
	var res cleanupDeleteResp
	mustOKData(t, w, &res)

	if res.Deleted != 0 {
		t.Fatalf("deleted = %d, want 0；非法 rel 不得删除任何文件", res.Deleted)
	}
	if res.Failed != len(rels) {
		t.Fatalf("failed = %d, want %d", res.Failed, len(rels))
	}
	if !fileExists(root, "download/a.png") {
		t.Fatal("非法请求导致 download/a.png 被删除")
	}
	if !fileExists(root, "upload/b.jpg") {
		t.Fatal("非法请求导致 upload/b.jpg 被删除")
	}
}

func TestPostFilesCleanupEmptyRels(t *testing.T) {
	r, _, key := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) { RegisterDashboardRoutes(v1, d) })

	cases := []string{
		`{"rels":[]}`,
		`{}`,
	}
	for _, body := range cases {
		w := apiReq(t, r, http.MethodPost, "/api/v1/files/cleanup", key, body)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("body=%s 状态码 = %d, want 400; resp=%s", body, w.Code, w.Body.String())
		}
		var env Response
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatalf("错误响应非信封 JSON: %v", err)
		}
		if env.Code != http.StatusBadRequest {
			t.Fatalf("code = %d, want 400", env.Code)
		}
	}
}

func TestDashboardRequiresKey(t *testing.T) {
	r, _, _ := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) { RegisterDashboardRoutes(v1, d) })

	w := apiReq(t, r, http.MethodGet, "/api/v1/dashboard", "", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("无 key GET /dashboard 状态码 = %d, want 401", w.Code)
	}
}

func TestGetDashboardWithVisits(t *testing.T) {
	r, d, key := newTestV1(t, func(v1 *gin.RouterGroup, d *Deps) { RegisterDashboardRoutes(v1, d) })

	// 无 stats 时 visits 为空数组结构而非 null
	w := apiReq(t, r, http.MethodGet, "/api/v1/dashboard", key, "")
	var before dashboardSummaryResp
	mustOKData(t, w, &before)
	if before.Visits.Labels == nil || before.Visits.PV == nil || before.Visits.IP == nil {
		t.Fatalf("无 stats 时 visits 数组应归一为空数组: %+v", before.Visits)
	}
	if before.Visits.Today != 0 {
		t.Fatalf("无 stats 时 today = %d, want 0", before.Visits.Today)
	}

	// 构造今日统计日志：PV=3，1 个独立 IP
	logsDir := filepath.Join(d.Cfg.RootDir, "logs-visits")
	todayFile := filepath.Join(logsDir, time.Now().Format("20060102")+".log")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		t.Fatalf("创建统计目录失败: %v", err)
	}
	if err := os.WriteFile(todayFile, []byte(`{"pv":3,"ips":{"1.1.1.1":3}}`), 0o644); err != nil {
		t.Fatalf("写入今日统计失败: %v", err)
	}
	d.Stats = stats.New(logsDir, nil)

	w = apiReq(t, r, http.MethodGet, "/api/v1/dashboard", key, "")
	var after dashboardSummaryResp
	mustOKData(t, w, &after)
	if after.Visits.Today != 3 || after.Visits.TodayIP != 1 {
		t.Fatalf("visits today/today_ip = %d/%d, want 3/1", after.Visits.Today, after.Visits.TodayIP)
	}
	if len(after.Visits.Labels) != 7 || len(after.Visits.PV) != 7 || len(after.Visits.IP) != 7 {
		t.Fatalf("近 7 天图表数组长度异常: %+v", after.Visits)
	}
	if after.Visits.PV[6] != 3 {
		t.Fatalf("图表今日 PV（末位） = %d, want 3", after.Visits.PV[6])
	}
}

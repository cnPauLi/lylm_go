package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"lylme_spage/internal/models"
	"lylme_spage/internal/service"
	"lylme_spage/internal/site"
)

func init() { gin.SetMode(gin.TestMode) }

func newTestEngine(t *testing.T, mw func(*service.APIKeyService) gin.HandlerFunc) (*gin.Engine, *service.APIKeyService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Config{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	store, err := site.NewStore(db, t.TempDir())
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}
	keys := service.NewAPIKeyService(store)
	r := gin.New()
	r.GET("/api/v1/ping", mw(keys), func(c *gin.Context) { OK(c, gin.H{"pong": true}) })
	return r, keys
}

func TestRequireKeyRejectsBadCredentials(t *testing.T) {
	r, _ := newTestEngine(t, RequireKey)

	cases := []struct {
		name   string
		header string
	}{
		{"无 Authorization", ""},
		{"非 Bearer 方案", "Basic sk-xxx"},
		{"Bearer 空值", "Bearer "},
		{"错误密钥", "Bearer sk-wrong"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("状态码 = %d, want 401", w.Code)
			}
			var body Response
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("响应非统一信封 JSON: %v", err)
			}
			if body.Code != http.StatusUnauthorized || body.Data != nil {
				t.Fatalf("错误响应结构异常: %+v", body)
			}
		})
	}
}

func TestRequireKeyAcceptsValidKey(t *testing.T) {
	r, keys := newTestEngine(t, RequireKey)
	key, err := keys.Ensure()
	if err != nil {
		t.Fatalf("Ensure 失败: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	req.Header.Set("Authorization", "Bearer "+key)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, want 200；body=%s", w.Code, w.Body.String())
	}
	var body Response
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应非 JSON: %v", err)
	}
	if body.Code != CodeOK {
		t.Fatalf("code = %d, want 200", body.Code)
	}
}

func TestRequireKeyRateLimited(t *testing.T) {
	// 限流阈值 3 次/分钟
	r, keys := newTestEngine(t, func(k *service.APIKeyService) gin.HandlerFunc {
		return RequireKeyWithLimit(k, 3, time.Minute)
	})
	key, err := keys.Ensure()
	if err != nil {
		t.Fatalf("Ensure 失败: %v", err)
	}
	var got429 bool
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
		req.Header.Set("Authorization", "Bearer "+key)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			got429 = true
			break
		}
	}
	if !got429 {
		t.Fatal("连续请求未触发 429 限流")
	}
}

package service

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"strings"
	"time"

	"lylme_spage/internal/site"
)

const (
	cfgAPIKey          = "api_key"
	cfgAPIKeyCreatedAt = "api_key_created_at"
	keyPrefix          = "sk-"
	keyRandomBytes     = 24 // 192 bit 熵 → 48 个十六进制字符
)

// APIKeyInfo 当前密钥信息
type APIKeyInfo struct {
	Key       string    `json:"key"`
	CreatedAt time.Time `json:"created_at"`
	Exists    bool      `json:"exists"`
}

// APIKeyService 单一 API/MCP 密钥的生成、读取、重新生成与校验
type APIKeyService struct {
	store *site.Store
}

// NewAPIKeyService 创建密钥服务
func NewAPIKeyService(store *site.Store) *APIKeyService {
	return &APIKeyService{store: store}
}

// GenerateKey 仅生成新密钥（sk- + 48 hex），不落库
func GenerateKey() (string, error) {
	b := make([]byte, keyRandomBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return keyPrefix + hex.EncodeToString(b), nil
}

// Get 读取当前密钥；不存在时 Exists=false
func (s *APIKeyService) Get() APIKeyInfo {
	key := strings.TrimSpace(s.store.Get(cfgAPIKey, ""))
	if key == "" {
		return APIKeyInfo{}
	}
	return APIKeyInfo{Key: key, CreatedAt: parseTime(s.store.Get(cfgAPIKeyCreatedAt, "")), Exists: true}
}

// Ensure 密钥不存在时生成并持久化；已存在则原样返回
func (s *APIKeyService) Ensure() (string, error) {
	if info := s.Get(); info.Exists {
		return info.Key, nil
	}
	return s.persist(GenerateKey)
}

// Regenerate 重新生成密钥；旧密钥立即失效
func (s *APIKeyService) Regenerate() (string, error) {
	return s.persist(GenerateKey)
}

// Validate 恒定时间校验 presented 是否为当前有效密钥；密钥未设置时一律拒绝
func (s *APIKeyService) Validate(presented string) bool {
	info := s.Get()
	if !info.Exists {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(presented), []byte(info.Key)) == 1
}

// persist 执行生成函数并落库（密钥 + 创建时间）
func (s *APIKeyService) persist(gen func() (string, error)) (string, error) {
	key, err := gen()
	if err != nil {
		return "", err
	}
	now := time.Now().Format(time.RFC3339)
	if err := s.store.Set(cfgAPIKey, key, "API/MCP接口密钥"); err != nil {
		return "", err
	}
	if err := s.store.Set(cfgAPIKeyCreatedAt, now, ""); err != nil {
		return "", err
	}
	return key, nil
}

// parseTime 解析 RFC3339 时间，失败返回零值
func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(s))
	if err != nil {
		return time.Time{}
	}
	return t
}

package service

import "errors"

// WxPlusConfig 微信推送配置
type WxPlusConfig struct {
	Key  string // 配置键 wxplus：微信推送密钥
	Time string // 配置键 wxplustime：每日推送时间
}

// WxPlus 读取微信推送配置（未配置时各字段为空串）
func (s *ConfigService) WxPlus() WxPlusConfig {
	return WxPlusConfig{
		Key:  s.store.Get("wxplus", ""),
		Time: s.store.Get("wxplustime", ""),
	}
}

// UpdateWxPlus 更新微信推送密钥与推送时间（按原始提交值落库；空 key 表示关闭推送）。
// 单项写入失败不阻断另一项（与原版一致），错误经 errors.Join 聚合返回。
func (s *ConfigService) UpdateWxPlus(key, t string) error {
	var errs []error
	if err := s.store.Set("wxplus", key, ""); err != nil {
		errs = append(errs, err)
	}
	if err := s.store.Set("wxplustime", t, ""); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

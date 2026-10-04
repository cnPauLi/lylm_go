package service

import "errors"

// LicenseConfig 网站授权配置
type LicenseConfig struct {
	Code     string // 授权码（配置键 c）
	HostMode string // 域名获取方式："1" HTTP_HOST / "2" SERVER_NAME；缺省或其他值归一为 "2"
}

// License 读取网站授权配置。hostmode 仅 "1"/"2" 两值，配置缺失或为其他值时按 "2"。
func (s *ConfigService) License() LicenseConfig {
	hostMode := "2"
	if s.store.Get("hostmode", "2") == "1" {
		hostMode = "1"
	}
	return LicenseConfig{
		Code:     s.store.Get("c", ""),
		HostMode: hostMode,
	}
}

// UpdateLicense 更新授权码与域名获取方式（按原始提交值落库）。
// 单项写入失败不阻断另一项（与原版一致），错误经 errors.Join 聚合返回。
func (s *ConfigService) UpdateLicense(code, hostmode string) error {
	var errs []error
	if err := s.store.Set("c", code, "程序授权码"); err != nil {
		errs = append(errs, err)
	}
	if err := s.store.Set("hostmode", hostmode, "域名获取方式"); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

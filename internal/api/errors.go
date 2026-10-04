package api

import (
	"errors"

	"github.com/gin-gonic/gin"

	"lylme_spage/internal/service"
)

// WriteServiceError 将 service 哨兵错误统一映射为 HTTP 响应。
// 各域 handler 不得自行分散实现错误映射。
func WriteServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		FailBadRequest(c, err.Error())
	case errors.Is(err, service.ErrGroupNotFound),
		errors.Is(err, service.ErrLinkNotFound),
		errors.Is(err, service.ErrTagNotFound),
		errors.Is(err, service.ErrSouNotFound),
		errors.Is(err, service.ErrPwdNotFound),
		errors.Is(err, service.ErrThemeNotFound),
		errors.Is(err, service.ErrApplyNotFound):
		FailNotFound(c, err.Error())
	case errors.Is(err, service.ErrWrongPassword),
		errors.Is(err, service.ErrNoChange):
		FailBadRequest(c, err.Error())
	default:
		FailInternal(c, "internal error")
	}
}

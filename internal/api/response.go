package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 业务码约定：成功 200；其余与 HTTP 状态码语义对齐。
const (
	CodeOK = 200
)

// Response REST API 统一响应信封
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// PageResult 分页列表统一出参
type PageResult[T any] struct {
	List     []T   `json:"list"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

// NewPageResult 构建分页结果，自动规整页码参数
func NewPageResult[T any](list []T, page, pageSize int, total int64) PageResult[T] {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	return PageResult[T]{List: list, Page: page, PageSize: pageSize, Total: total}
}

// OK 写入成功响应
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{Code: CodeOK, Message: "ok", Data: data})
}

// Fail 写入失败响应（HTTP 状态码与业务码一致）
func Fail(c *gin.Context, status, code int, message string) {
	c.JSON(status, Response{Code: code, Message: message, Data: nil})
}

// FailUnauthorized 401
func FailUnauthorized(c *gin.Context, message string) {
	Fail(c, http.StatusUnauthorized, http.StatusUnauthorized, message)
}

// FailBadRequest 400
func FailBadRequest(c *gin.Context, message string) {
	Fail(c, http.StatusBadRequest, http.StatusBadRequest, message)
}

// FailNotFound 404
func FailNotFound(c *gin.Context, message string) {
	Fail(c, http.StatusNotFound, http.StatusNotFound, message)
}

// FailInternal 500
func FailInternal(c *gin.Context, message string) {
	Fail(c, http.StatusInternalServerError, http.StatusInternalServerError, message)
}

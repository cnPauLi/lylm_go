package service

import "errors"

// service 层通用哨兵错误。service 只返回强类型结果与哨兵错误，
// 不依赖 gin、不渲染 HTML；由各处理器（admin/api/mcp）自行映射输出。
var (
	// ErrInvalidInput 输入参数不合法（通用校验失败）
	ErrInvalidInput = errors.New("service: 输入参数不合法")

	// ErrGroupNotFound 分组不存在
	ErrGroupNotFound = errors.New("group: 分组不存在")

	// ErrGroupNameEmpty 分组名称为空；归属于 ErrInvalidInput
	ErrGroupNameEmpty = errors.Join(ErrInvalidInput, errors.New("group: 分组名称不能为空"))

	// ErrLinkNotFound 链接不存在
	ErrLinkNotFound = errors.New("link: 链接不存在")

	// ErrLinkHeadUnreachable 目标站点无法访问（抓取 head 时 title/icon 均为空）
	ErrLinkHeadUnreachable = errors.New("link: 无法访问目标站点")
)

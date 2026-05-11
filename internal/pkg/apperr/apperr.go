package apperr

import "net/http"

type AppError struct {
	Code       int    `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
	Fields     any    `json:"fields,omitempty"`
}

func (e *AppError) Error() string { return e.Message }

func New(code int, httpStatus int, msg string) *AppError {
	return &AppError{Code: code, HTTPStatus: httpStatus, Message: msg}
}

func WithFields(e *AppError, fields any) *AppError {
	cp := *e
	cp.Fields = fields
	return &cp
}

var (
	ErrBadRequest       = New(1001, http.StatusBadRequest, "参数无效")
	ErrUnauthorized     = New(1002, http.StatusUnauthorized, "未认证或令牌无效")
	ErrForbidden        = New(1003, http.StatusForbidden, "无权限")
	ErrNotFound         = New(1004, http.StatusNotFound, "资源不存在")
	ErrConflict         = New(1005, http.StatusConflict, "资源冲突")
	ErrUpstreamTimeout  = New(2001, http.StatusRequestTimeout, "上游超时")
	ErrRateLimited      = New(2002, http.StatusTooManyRequests, "限流")
	ErrUpstream         = New(2003, http.StatusBadGateway, "上游错误")
	ErrUpstreamAuth     = New(2004, http.StatusUnauthorized, "上游鉴权失败")
	ErrInternal         = New(3001, http.StatusInternalServerError, "内部错误")
	ErrEngineBusy       = New(3002, http.StatusServiceUnavailable, "引擎繁忙")
	ErrExportNotReady   = New(3003, http.StatusServiceUnavailable, "导出依赖未就绪")
)

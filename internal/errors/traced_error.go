package errors

import (
	"runtime/debug"
	"time"
)

// TracedError 带追踪信息的错误
type TracedError struct {
	*AppError
	Stack     string
	Trace     []string
	Labels    map[string]string
	Timestamp time.Time
	Context   ErrorContext
}

// ErrorContext 错误上下文信息
type ErrorContext struct {
	RequestID string
	UserID    int
	Path      string
	Method    string
	Params    map[string]interface{}
	Timestamp time.Time
}

// NewTracedError 创建带追踪信息的错误
func NewTracedError(err error, ctx ErrorContext) *TracedError {
	var appErr *AppError
	if ae, ok := err.(*AppError); ok {
		appErr = ae
	} else {
		appErr = &AppError{
			Code:    ErrInternal,
			Message: err.Error(),
			Err:     err,
		}
	}

	return &TracedError{
		AppError:  appErr,
		Stack:     string(debug.Stack()),
		Trace:     []string{},
		Labels:    make(map[string]string),
		Timestamp: time.Now(),
		Context:   ctx,
	}
}

// AddLabel 添加标签
func (e *TracedError) AddLabel(key, value string) *TracedError {
	e.Labels[key] = value
	return e
}

// AddTrace 添加追踪信息
func (e *TracedError) AddTrace(trace string) *TracedError {
	e.Trace = append(e.Trace, trace)
	return e
}

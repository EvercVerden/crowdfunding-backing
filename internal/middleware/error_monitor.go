package middleware

import (
	"crowdfunding-backend/internal/errors"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ErrorMonitor struct {
	errorCounts map[errors.ErrorCode]int
	mu          sync.RWMutex
}

func NewErrorMonitor() *ErrorMonitor {
	return &ErrorMonitor{
		errorCounts: make(map[errors.ErrorCode]int),
	}
}

func (m *ErrorMonitor) RecordError(err error) {
	if appErr, ok := err.(*errors.AppError); ok {
		m.mu.Lock()
		m.errorCounts[appErr.Code]++
		m.mu.Unlock()
	}
}

func (m *ErrorMonitor) GetErrorCounts() map[errors.ErrorCode]int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	counts := make(map[errors.ErrorCode]int)
	for code, count := range m.errorCounts {
		counts[code] = count
	}
	return counts
}

func ErrorMonitorMiddleware(monitor *ErrorMonitor) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				monitor.RecordError(e.Err)
				// 记录错误日志
				if appErr, ok := e.Err.(*errors.AppError); ok {
					zap.L().Error("请求处理错误",
						zap.Int("error_code", int(appErr.Code)),
						zap.String("error_message", appErr.Message),
						zap.Error(appErr.Err),
						zap.String("path", c.Request.URL.Path),
						zap.String("method", c.Request.Method))
				}
			}
		}
	}
}

package errors

import (
	"sync"
	"time"
)

// ErrorAnalytics 错误分析
type ErrorAnalytics struct {
	mu            sync.RWMutex
	TotalErrors   int
	ErrorsByCode  map[ErrorCode]int
	ErrorsByPath  map[string]int
	ErrorPatterns map[string]int
	LastErrorTime time.Time
}

// NewErrorAnalytics 创建错误分析器
func NewErrorAnalytics() *ErrorAnalytics {
	return &ErrorAnalytics{
		ErrorsByCode:  make(map[ErrorCode]int),
		ErrorsByPath:  make(map[string]int),
		ErrorPatterns: make(map[string]int),
	}
}

// Record 记录错误
func (a *ErrorAnalytics) Record(err *TracedError) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.TotalErrors++
	a.ErrorsByCode[err.Code]++
	a.ErrorsByPath[err.Context.Path]++
	a.LastErrorTime = time.Now()

	// 识别错误模式
	pattern := a.identifyPattern(err)
	if pattern != "" {
		a.ErrorPatterns[pattern]++
	}
}

// identifyPattern 识别错误模式
func (a *ErrorAnalytics) identifyPattern(err *TracedError) string {
	// 实现错误模式识别逻辑
	return ""
}

// GetStats 获取统计信息
func (a *ErrorAnalytics) GetStats() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return map[string]interface{}{
		"total_errors":   a.TotalErrors,
		"errors_by_code": a.ErrorsByCode,
		"errors_by_path": a.ErrorsByPath,
		"error_patterns": a.ErrorPatterns,
		"last_error":     a.LastErrorTime,
	}
}

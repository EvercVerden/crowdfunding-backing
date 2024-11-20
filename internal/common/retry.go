package common

import (
	"database/sql"
	"time"
)

// IsTemporary 判断是否为临时性错误
func IsTemporary(err error) bool {
	if temp, ok := err.(interface{ Temporary() bool }); ok {
		return temp.Temporary()
	}
	return false
}

// IsRetryable 判断是否可重试
func IsRetryable(err error) bool {
	return IsTemporary(err) || err == sql.ErrConnDone
}

// WithRetry 通用重试机制
func WithRetry(operation func() error, maxRetries int) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		if err = operation(); err == nil {
			return nil
		}
		if !IsRetryable(err) {
			return err
		}
		time.Sleep(time.Second * time.Duration(i+1))
	}
	return err
}

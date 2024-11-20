package errors

import "fmt"

// ServiceError 定义服务层错误
type ServiceError struct {
	Code    ErrorCode
	Message string
	Err     error
}

// ErrorCode 定义错误码类型
type ErrorCode int

const (
	// 数据库错误
	ErrDatabase ErrorCode = iota + 1000
	ErrNotFound
	ErrDuplicate

	// 业务逻辑错误
	ErrInvalidInput
	ErrUnauthorized
	ErrForbidden

	// 系统错误
	ErrInternal
	ErrThirdParty
)

func (e *ServiceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// New 创建新的服务错误
func New(code ErrorCode, message string) error {
	return &ServiceError{
		Code:    code,
		Message: message,
	}
}

// Wrap 包装已有错误
func Wrap(code ErrorCode, message string, err error) error {
	return &ServiceError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// IsServiceError 判断是否为服务错误
func IsServiceError(err error) bool {
	_, ok := err.(*ServiceError)
	return ok
}

// GetErrorCode 获取错误码
func GetErrorCode(err error) ErrorCode {
	if se, ok := err.(*ServiceError); ok {
		return se.Code
	}
	return ErrInternal
}

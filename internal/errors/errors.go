package errors

import "fmt"

// ErrorCode 定义错误码类型
type ErrorCode int

// 定义系统级错误码 (1000-1999)
const (
	ErrInternal ErrorCode = 1000 + iota
	ErrDatabase
	ErrCache
	ErrTimeout
)

// 定义认证相关错误码 (2000-2999)
const (
	ErrUnauthorized ErrorCode = 2000 + iota
	ErrForbidden
	ErrInvalidToken
	ErrTokenExpired
	ErrInvalidCredentials
)

// 定义请求相关错误码 (3000-3999)
const (
	ErrBadRequest ErrorCode = 3000 + iota
	ErrValidation
	ErrResourceNotFound
	ErrResourceExists
	ErrResourceConflict
)

// 定义业务相关错误码 (4000-4999)
const (
	ErrUserNotFound ErrorCode = 4000 + iota
	ErrUserExists
	ErrWeakPassword
	ErrProjectNotFound
	ErrInsufficientFunds
)

// AppError 定义应用错误结构
type AppError struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// New 创建新的应用错误
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap 包装已有错误
func Wrap(code ErrorCode, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

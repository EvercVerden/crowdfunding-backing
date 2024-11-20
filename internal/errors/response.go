package errors

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse 定义错误响应结构
type ErrorResponse struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Error   string    `json:"error,omitempty"`
}

// SuccessResponse 定义成功响应结构
type SuccessResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// 错误码与HTTP状态码映射
var errorStatusMap = map[ErrorCode]int{
	// 系统错误 (1000-1999)
	ErrInternal: http.StatusInternalServerError,
	ErrDatabase: http.StatusInternalServerError,
	ErrCache:    http.StatusInternalServerError,
	ErrTimeout:  http.StatusRequestTimeout,

	// 认证错误 (2000-2999)
	ErrUnauthorized:       http.StatusUnauthorized,
	ErrForbidden:          http.StatusForbidden,
	ErrInvalidToken:       http.StatusUnauthorized,
	ErrTokenExpired:       http.StatusUnauthorized,
	ErrInvalidCredentials: http.StatusUnauthorized,

	// 请求错误 (3000-3999)
	ErrBadRequest:       http.StatusBadRequest,
	ErrValidation:       http.StatusBadRequest,
	ErrResourceNotFound: http.StatusNotFound,
	ErrResourceExists:   http.StatusConflict,
	ErrResourceConflict: http.StatusConflict,

	// 业务错误 (4000-4999)
	ErrUserNotFound:      http.StatusNotFound,
	ErrUserExists:        http.StatusConflict,
	ErrWeakPassword:      http.StatusBadRequest,
	ErrProjectNotFound:   http.StatusNotFound,
	ErrInsufficientFunds: http.StatusBadRequest,
}

// HandleError 统一处理错误响应
func HandleError(c *gin.Context, err error) {
	if appErr, ok := err.(*AppError); ok {
		status := errorStatusMap[appErr.Code]
		if status == 0 {
			status = http.StatusInternalServerError
		}

		resp := ErrorResponse{
			Code:    appErr.Code,
			Message: appErr.Message,
		}

		if appErr.Err != nil {
			resp.Error = appErr.Err.Error()
		}

		c.JSON(status, resp)
		return
	}

	// 处理非 AppError 类型的错误
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Code:    ErrInternal,
		Message: "Internal Server Error",
		Error:   err.Error(),
	})
}

// HandleSuccess 统一处理成功响应
func HandleSuccess(c *gin.Context, data interface{}, message string) {
	resp := SuccessResponse{
		Code:    http.StatusOK,
		Message: message,
		Data:    data,
	}
	c.JSON(http.StatusOK, resp)
}

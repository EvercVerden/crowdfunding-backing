package util

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type AppError struct {
	StatusCode int
	Message    string
}

func NewAppError(statusCode int, message string) *AppError {
	return &AppError{
		StatusCode: statusCode,
		Message:    message,
	}
}

// 添加 Error 方法以实现 error 接口
func (e *AppError) Error() string {
	return e.Message
}

func HandleError(c *gin.Context, err error) {
	if appErr, ok := err.(*AppError); ok {
		c.JSON(appErr.StatusCode, gin.H{"error": appErr.Message})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
	}
}

package middleware

import (
	"crowdfunding-backend/internal/errors"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// 记录堆栈信息
				stack := string(debug.Stack())
				zap.L().Error("发生panic",
					zap.Any("error", r),
					zap.String("stack", stack))

				errors.HandleError(c, errors.New(errors.ErrInternal, "系统内部错误"))
			}
		}()
		c.Next()
	}
}

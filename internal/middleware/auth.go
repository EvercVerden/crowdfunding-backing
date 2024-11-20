package middleware

import (
	"context"
	"crowdfunding-backend/internal/errors"
	"crowdfunding-backend/internal/service"
	"crowdfunding-backend/internal/util"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func AuthMiddleware(userService *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		util.Logger.Info("进入认证中间件",
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method))

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			errors.HandleError(c, errors.New(errors.ErrUnauthorized, "需要认证"))
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			errors.HandleError(c, errors.New(errors.ErrUnauthorized, "无效的认证格式"))
			c.Abort()
			return
		}

		if userService.IsTokenBlacklisted(parts[1]) {
			errors.HandleError(c, errors.New(errors.ErrUnauthorized, "令牌已被撤销"))
			c.Abort()
			return
		}

		userID, err := util.ValidateToken(parts[1])
		if err != nil {
			errors.HandleError(c, errors.Wrap(errors.ErrUnauthorized, "无效或过期的令牌", err))
			c.Abort()
			return
		}

		c.Set("user_id", userID)

		select {
		case <-ctx.Done():
			errors.HandleError(c, errors.New(errors.ErrTimeout, "请求超时"))
			c.Abort()
			return
		default:
			c.Next()
		}
	}
}

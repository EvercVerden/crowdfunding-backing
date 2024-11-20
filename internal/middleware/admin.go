package middleware

import (
	"crowdfunding-backend/internal/service"
	"crowdfunding-backend/internal/util"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AdminMiddleware 确保只有管理员可以访问某些路由
func AdminMiddleware(userService *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		util.Logger.Info("进入管理员中间件",
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method))

		userID, exists := c.Get("user_id")
		if !exists {
			util.Logger.Warn("用户ID不存在")
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "需要认证",
				"error":   "User ID not found in context",
			})
			c.Abort()
			return
		}

		user, err := userService.GetUserByID(userID.(int))
		if err != nil || user.Role != "admin" {
			util.Logger.Warn("非管理员访问",
				zap.Int("user_id", userID.(int)),
				zap.Error(err))
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "需要管理员权限",
				"error":   "Admin access required",
			})
			c.Abort()
			return
		}

		util.Logger.Info("管理员验证通过",
			zap.Int("user_id", userID.(int)))
		c.Next()
	}
}

package user

import (
	"crowdfunding-backend/internal/errors"
	"crowdfunding-backend/internal/model"
	"crowdfunding-backend/internal/service"
	"crowdfunding-backend/internal/util"
	"unicode"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuthHandler 处理与认证相关的HTTP请求
type AuthHandler struct {
	userService service.UserServiceInterface
}

// NewAuthHandler 创建一个新的 AuthHandler 实例
func NewAuthHandler(userService service.UserServiceInterface) *AuthHandler {
	return &AuthHandler{userService}
}

// Register 处理用户注册请求
func (h *AuthHandler) Register(c *gin.Context) {
	var registerData struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&registerData); err != nil {
		util.Logger.Warn("注册失败，无效的请求数据", zap.Error(err))
		errors.HandleError(c, errors.Wrap(errors.ErrValidation, "无效的请求数据", err))
		return
	}

	if !isPasswordStrong(registerData.Password) {
		errors.HandleError(c, errors.New(errors.ErrWeakPassword, "密码强度不足"))
		return
	}

	user := &model.User{
		Username:     registerData.Username,
		Email:        registerData.Email,
		PasswordHash: registerData.Password,
	}

	if err := h.userService.Register(user); err != nil {
		if appErr, ok := err.(*errors.AppError); ok && appErr.Code == errors.ErrUserExists {
			util.Logger.Warn("注册失败，用户名已存在",
				zap.String("username", user.Username))
			errors.HandleError(c, err)
			return
		}
		util.Logger.Error("注册失败", zap.Error(err))
		errors.HandleError(c, errors.Wrap(errors.ErrInternal, "注册失败", err))
		return
	}

	errors.HandleSuccess(c, gin.H{
		"user_id": user.ID,
	}, "注册成功")
}

// Login 处理用户登录请求
func (h *AuthHandler) Login(c *gin.Context) {
	var loginData struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&loginData); err != nil {
		errors.HandleError(c, errors.Wrap(errors.ErrValidation, "无效的请求数据", err))
		return
	}

	user, err := h.userService.Login(loginData.Email, loginData.Password)
	if err != nil {
		errors.HandleError(c, errors.Wrap(errors.ErrInvalidCredentials, "登录失败", err))
		return
	}

	token, err := util.GenerateToken(user.ID)
	if err != nil {
		errors.HandleError(c, errors.Wrap(errors.ErrInternal, "生成令牌失败", err))
		return
	}

	errors.HandleSuccess(c, gin.H{
		"token": token,
		"user":  user,
	}, "登录成功")
}

// RequestPasswordReset 处理密码重置请求
func (h *AuthHandler) RequestPasswordReset(c *gin.Context) {
	var requestData struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		errors.HandleError(c, errors.Wrap(errors.ErrValidation, "无效的邮箱格式", err))
		return
	}

	if err := h.userService.RequestPasswordReset(requestData.Email); err != nil {
		errors.HandleError(c, errors.Wrap(errors.ErrInternal, "请求密码重置失败", err))
		return
	}

	errors.HandleSuccess(c, nil, "密码重置邮件已发送")
}

// ResetPassword 处理密码重置
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var resetData struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&resetData); err != nil {
		errors.HandleError(c, errors.Wrap(errors.ErrValidation, "无效的请求数据", err))
		return
	}

	if !isPasswordStrong(resetData.NewPassword) {
		errors.HandleError(c, errors.New(errors.ErrWeakPassword, "新密码强度不足"))
		return
	}

	if err := h.userService.ResetPassword(resetData.Token, resetData.NewPassword); err != nil {
		errors.HandleError(c, errors.Wrap(errors.ErrInternal, "重置密码失败", err))
		return
	}

	errors.HandleSuccess(c, nil, "密码重置成功")
}

// Logout 处理用户登出
func (h *AuthHandler) Logout(c *gin.Context) {
	userID := c.GetInt("user_id")
	if err := h.userService.Logout(userID); err != nil {
		errors.HandleError(c, errors.Wrap(errors.ErrInternal, "登出失败", err))
		return
	}
	errors.HandleSuccess(c, nil, "已成功登出")
}

// VerifyEmail 处理邮箱验证
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		errors.HandleError(c, errors.New(errors.ErrValidation, "缺少验证令牌"))
		return
	}

	if err := h.userService.VerifyEmail(token); err != nil {
		errors.HandleError(c, errors.Wrap(errors.ErrInternal, "验证邮箱失败", err))
		return
	}

	errors.HandleSuccess(c, nil, "邮箱验证成功")
}

// RefreshToken 处理令牌刷新
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")
	if tokenString == "" {
		errors.HandleError(c, errors.New(errors.ErrUnauthorized, "缺少令牌"))
		return
	}

	newToken, err := util.RefreshToken(tokenString)
	if err != nil {
		errors.HandleError(c, errors.Wrap(errors.ErrUnauthorized, "刷新令牌失败", err))
		return
	}

	errors.HandleSuccess(c, gin.H{"token": newToken}, "令牌刷新成功")
}

func isPasswordStrong(password string) bool {
	var (
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)
	if len(password) < 8 {
		return false
	}
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}
	return hasUpper && hasLower && hasNumber && hasSpecial
}

// AdminLogin 处理管理员登录
func (h *AuthHandler) AdminLogin(c *gin.Context) {
	var loginData struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&loginData); err != nil {
		errors.HandleError(c, errors.Wrap(errors.ErrValidation, "无效的请求数据", err))
		return
	}

	user, err := h.userService.Login(loginData.Email, loginData.Password)
	if err != nil {
		errors.HandleError(c, errors.Wrap(errors.ErrInvalidCredentials, "登录失败", err))
		return
	}

	if user.Role != "admin" {
		errors.HandleError(c, errors.New(errors.ErrForbidden, "需要管理员权限"))
		return
	}

	token, err := util.GenerateToken(user.ID)
	if err != nil {
		errors.HandleError(c, errors.Wrap(errors.ErrInternal, "生成令牌失败", err))
		return
	}

	errors.HandleSuccess(c, gin.H{
		"token": token,
		"user": gin.H{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"role":       user.Role,
			"avatar_url": user.AvatarURL,
		},
	}, "管理员登录成功")
}

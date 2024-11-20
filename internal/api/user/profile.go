package user

import (
	"crowdfunding-backend/config"
	"crowdfunding-backend/internal/errors"
	"crowdfunding-backend/internal/service"
	"crowdfunding-backend/internal/storage"
	"crowdfunding-backend/internal/util"
	"fmt"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ProfileHandler struct {
	userService *service.UserService
	storage     *storage.LocalStorage
}

func NewProfileHandler(userService *service.UserService, storage *storage.LocalStorage) *ProfileHandler {
	return &ProfileHandler{userService, storage}
}

func (h *ProfileHandler) GetProfile(c *gin.Context) {
	userID := c.GetInt("user_id")
	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		util.Logger.Error("获取用户资料失败", zap.Error(err))
		if appErr, ok := err.(*errors.AppError); ok {
			errors.HandleError(c, appErr)
			return
		}
		errors.HandleError(c, errors.Wrap(errors.ErrInternal, "获取用户资料失败", err))
		return
	}

	errors.HandleSuccess(c, gin.H{
		"user": user,
	}, "")
}

func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetInt("user_id")

	currentUser, err := h.userService.GetUserByID(userID)
	if err != nil {
		util.Logger.Error("获取用户信息失败", zap.Error(err))
		errors.HandleError(c, errors.Wrap(errors.ErrUserNotFound, "获取用户信息失败", err))
		return
	}

	var updateData struct {
		Username string `json:"username"`
		Bio      string `json:"bio"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		util.Logger.Warn("更新用户资料失败，无效的请求数据", zap.Error(err))
		errors.HandleError(c, errors.Wrap(errors.ErrValidation, "无效的请求数据", err))
		return
	}

	// 只更新允许用户修改的字段
	if updateData.Username != "" {
		currentUser.Username = updateData.Username
	}
	if updateData.Bio != "" {
		currentUser.Bio = updateData.Bio
	}

	if err := h.userService.UpdateUser(currentUser); err != nil {
		util.Logger.Error("更新用户资料失败", zap.Error(err))
		errors.HandleError(c, errors.Wrap(errors.ErrInternal, "更新用户资料失败", err))
		return
	}

	errors.HandleSuccess(c, gin.H{
		"user": currentUser,
	}, "资料更新成功")
}

func (h *ProfileHandler) UploadAvatar(c *gin.Context) {
	userID, _ := c.Get("user_id")

	file, err := c.FormFile("avatar")
	if err != nil {
		util.Logger.Error("获取上传文件失败", zap.Error(err))
		errors.HandleError(c, errors.Wrap(errors.ErrBadRequest, "无法获取上传文件", err))
		return
	}

	filename := util.GenerateUniqueFilename(file.Filename)
	path := fmt.Sprintf("avatars/%d/%s", userID, filename)

	avatarURL, err := h.storage.UploadFile(file, path)
	if err != nil {
		util.Logger.Error("上传头像失败", zap.Error(err))
		errors.HandleError(c, errors.Wrap(errors.ErrInternal, "上传头像失败", err))
		return
	}

	fullAvatarURL := fmt.Sprintf("%s/uploads/%s", config.AppConfig.BackendURL, avatarURL)

	if err := h.userService.UpdateAvatar(userID.(int), fullAvatarURL); err != nil {
		util.Logger.Error("更新用户头像失败", zap.Error(err))
		errors.HandleError(c, errors.Wrap(errors.ErrInternal, "更新用户头像失败", err))
		return
	}

	errors.HandleSuccess(c, gin.H{
		"avatar_url": fullAvatarURL,
	}, "头像上传成功")
}

func (h *ProfileHandler) DeleteAccount(c *gin.Context) {
	userID := c.GetInt("user_id")

	if err := h.userService.DeleteAccount(userID); err != nil {
		util.Logger.Error("注销账户失败", zap.Error(err))
		errors.HandleError(c, errors.Wrap(errors.ErrInternal, "注销账户失败", err))
		return
	}

	errors.HandleSuccess(c, nil, "账户已成功注销")
}

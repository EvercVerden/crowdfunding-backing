package user

import (
	"crowdfunding-backend/internal/errors"
	"crowdfunding-backend/internal/model"
	"crowdfunding-backend/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService}
}

func (h *UserHandler) CreateAddress(c *gin.Context) {
	var address model.UserAddress
	if err := c.ShouldBindJSON(&address); err != nil {
		errors.HandleError(c, errors.Wrap(errors.ErrValidation, "无效的地址数据", err))
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		errors.HandleError(c, errors.New(errors.ErrUnauthorized, "未授权的访问"))
		return
	}

	address.UserID = userID.(int)
	if err := h.userService.CreateAddress(&address); err != nil {
		errors.HandleError(c, errors.Wrap(errors.ErrInternal, "创建地址失败", err))
		return
	}

	errors.HandleSuccess(c, address, "地址创建成功")
}

func (h *UserHandler) UpdateAddress(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		errors.HandleError(c, errors.Wrap(errors.ErrValidation, "无效的地址ID", err))
		return
	}

	var address model.UserAddress
	if err := c.ShouldBindJSON(&address); err != nil {
		errors.HandleError(c, errors.Wrap(errors.ErrValidation, "无效的地址数据", err))
		return
	}

	address.ID = id
	userID, _ := c.Get("user_id")
	address.UserID = userID.(int)

	if err := h.userService.UpdateAddress(&address); err != nil {
		errors.HandleError(c, errors.Wrap(errors.ErrInternal, "更新地址失败", err))
		return
	}

	errors.HandleSuccess(c, address, "地址更新成功")
}

func (h *UserHandler) DeleteAddress(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		errors.HandleError(c, errors.Wrap(errors.ErrValidation, "无效的地址ID", err))
		return
	}

	if err := h.userService.DeleteAddress(id); err != nil {
		errors.HandleError(c, errors.Wrap(errors.ErrInternal, "删除地址失败", err))
		return
	}

	errors.HandleSuccess(c, nil, "地址删除成功")
}

func (h *UserHandler) ListAddresses(c *gin.Context) {
	userID, _ := c.Get("user_id")
	addresses, err := h.userService.ListUserAddresses(userID.(int))
	if err != nil {
		errors.HandleError(c, errors.Wrap(errors.ErrInternal, "获取地址列表失败", err))
		return
	}

	errors.HandleSuccess(c, addresses, "")
}

func (h *UserHandler) SetDefaultAddress(c *gin.Context) {
	addressID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		errors.HandleError(c, errors.Wrap(errors.ErrValidation, "无效的地址ID", err))
		return
	}

	userID, _ := c.Get("user_id")
	if err := h.userService.SetDefaultAddress(userID.(int), addressID); err != nil {
		errors.HandleError(c, errors.Wrap(errors.ErrInternal, "设置默认地址失败", err))
		return
	}

	errors.HandleSuccess(c, nil, "默认地址设置成功")
}

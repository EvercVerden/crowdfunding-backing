package payment

import (
	"crowdfunding-backend/internal/service"
	"crowdfunding-backend/internal/util"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RefundHandler 处理退款相关的请求
type RefundHandler struct {
	refundService *service.RefundService
}

func NewRefundHandler(refundService *service.RefundService) *RefundHandler {
	return &RefundHandler{refundService}
}

// RequestRefund 处理退款申请
func (h *RefundHandler) RequestRefund(c *gin.Context) {
	orderID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		util.Logger.Error("无效的订单ID",
			zap.Error(err),
			zap.Int("order_id", orderID))
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid order ID",
		})
		return
	}

	var input struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		util.Logger.Error("无效的请求数据",
			zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid input",
		})
		return
	}

	userID, _ := c.Get("user_id")
	err = h.refundService.RequestRefund(orderID, userID.(int), input.Reason)
	if err != nil {
		util.Logger.Error("申请退款失败",
			zap.Error(err),
			zap.Int("order_id", orderID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to request refund",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Refund request submitted successfully",
	})
}

// GetRefundStatus 获取退款状态
func (h *RefundHandler) GetRefundStatus(c *gin.Context) {
	orderID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid order ID",
		})
		return
	}

	refundRequest, err := h.refundService.GetRefundStatus(orderID)
	if err != nil {
		util.Logger.Error("获取退款状态失败",
			zap.Error(err),
			zap.Int("order_id", orderID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to get refund status",
			"error":   err.Error(),
		})
		return
	}

	if refundRequest == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "No refund request found for this order",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"id":            refundRequest.ID,
			"status":        refundRequest.Status,
			"reason":        refundRequest.Reason,
			"admin_comment": refundRequest.AdminComment,
			"created_at":    refundRequest.CreatedAt,
			"updated_at":    refundRequest.UpdatedAt,
		},
	})
}

// ProcessRefund 处理退款申请（管理员）
func (h *RefundHandler) ProcessRefund(c *gin.Context) {
	requestID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid request ID",
		})
		return
	}

	var input struct {
		Approved bool   `json:"approved" binding:"required"`
		Comment  string `json:"comment"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid input",
		})
		return
	}

	err = h.refundService.ProcessRefund(requestID, input.Approved, input.Comment)
	if err != nil {
		util.Logger.Error("处理退款申请失败",
			zap.Error(err),
			zap.Int("request_id", requestID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to process refund",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Refund request processed successfully",
	})
}

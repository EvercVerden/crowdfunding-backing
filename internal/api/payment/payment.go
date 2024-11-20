package payment

import (
	"crowdfunding-backend/internal/model"
	"crowdfunding-backend/internal/service"
	"crowdfunding-backend/internal/util"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PaymentHandler struct {
	paymentService *service.PaymentService
	projectService *service.ProjectService
}

func NewPaymentHandler(paymentService *service.PaymentService, projectService *service.ProjectService) *PaymentHandler {
	return &PaymentHandler{paymentService, projectService}
}

func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		util.Logger.Error("无效的项目ID", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid project ID",
		})
		return
	}

	var input struct {
		Amount    float64 `json:"amount" binding:"required"`
		AddressID int     `json:"address_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		util.Logger.Error("无效的请求数据", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid input data",
			"details": err.Error(),
		})
		return
	}

	userID, _ := c.Get("user_id")

	// 创建支付记录
	payment := &model.Payment{
		UserID:    userID.(int),
		ProjectID: projectID,
		Amount:    input.Amount,
		Status:    "pending",
	}

	util.Logger.Info("开始创建支付流程",
		zap.Int("user_id", payment.UserID),
		zap.Int("project_id", payment.ProjectID),
		zap.Float64("amount", payment.Amount))

	// 创建订单
	order := &model.Order{
		UserID:    payment.UserID,
		ProjectID: payment.ProjectID,
		Amount:    payment.Amount,
		Status:    "pending",
		AddressID: &input.AddressID,
	}

	util.Logger.Info("准备创建订单",
		zap.Int("user_id", order.UserID),
		zap.Int("project_id", order.ProjectID),
		zap.Float64("amount", order.Amount),
		zap.Int("address_id", *order.AddressID))

	// 处理支付和创建订单
	order, err = h.paymentService.ProcessPayment(payment, input.AddressID)
	if err != nil {
		util.Logger.Error("处理支付失败",
			zap.Error(err),
			zap.Any("payment", payment),
			zap.Any("order", order))

		// 检查是否是截日期错误
		if err.Error() == "project has ended, cannot accept new payments" {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "项目已截止，无法继续支持",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to process payment",
			"details": err.Error(),
		})
		return
	}

	util.Logger.Info("支付处理成功",
		zap.Int("order_id", order.ID),
		zap.String("order_number", order.OrderNumber))

	c.JSON(http.StatusCreated, gin.H{
		"code": 201,
		"data": gin.H{
			"order":   order,
			"payment": payment,
		},
		"message": "Payment processed successfully",
	})
}

func (h *PaymentHandler) RequestRefundForFailedProject(c *gin.Context) {
	orderID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		util.Logger.Error("无效的订单ID", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的订单ID",
		})
		return
	}

	userID, _ := c.Get("user_id")

	err = h.paymentService.RequestRefundForFailedProject(orderID, userID.(int))
	if err != nil {
		// 根据错误类型返回不同的状态码和消息
		switch {
		case strings.Contains(err.Error(), "订单不存在"):
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": err.Error(),
			})
		case strings.Contains(err.Error(), "订单不属于当前用户"):
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": err.Error(),
			})
		case strings.Contains(err.Error(), "已存在退款申请"):
			c.JSON(http.StatusConflict, gin.H{
				"code":    409,
				"message": err.Error(),
			})
		default:
			util.Logger.Error("申请退款失败",
				zap.Error(err),
				zap.Int("order_id", orderID))
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "申请退款失败",
				"error":   err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "退款申请已提交",
	})
}

func (h *PaymentHandler) GetOrder(c *gin.Context) {
	orderID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid order ID",
		})
		return
	}

	order, err := h.paymentService.GetOrderByID(orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to get order",
			"details": err.Error(),
		})
		return
	}

	if order == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "Order not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"id":     order.ID,
			"status": order.Status,
			"amount": order.Amount,
			"project": gin.H{
				"id": order.ProjectID,
			},
			"address": gin.H{
				"receiver_name": order.Address.ReceiverName,
				"phone":         order.Address.Phone,
				"full_address": fmt.Sprintf("%s%s%s%s",
					order.Address.Province,
					order.Address.City,
					order.Address.District,
					order.Address.DetailAddress),
			},
			"created_at": order.CreatedAt,
		},
	})
}

func (h *PaymentHandler) ListOrders(c *gin.Context) {
	userID, _ := c.Get("user_id")
	orders, err := h.paymentService.GetOrdersByUser(userID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to get orders",
			"details": err.Error(),
		})
		return
	}

	// 转换为前端需要的格式
	var formattedOrders []gin.H
	for _, order := range orders {
		// 获取项目信息
		project, err := h.projectService.GetProjectByID(order.ProjectID)
		if err != nil || project == nil {
			continue
		}

		// 同步订单状态与项目状态
		err = h.paymentService.SyncOrdersWithProjectStatus(order.ProjectID)
		if err != nil {
			util.Logger.Error("同步订单状态失败",
				zap.Error(err),
				zap.Int("project_id", order.ProjectID))
		}

		// 重新获取更新后的订单信息
		updatedOrder, err := h.paymentService.GetOrderByID(order.ID)
		if err != nil || updatedOrder == nil {
			continue
		}
		order = updatedOrder

		// 获取项目图片
		images, err := h.projectService.GetProjectImages(order.ProjectID)
		if err != nil {
			images = []model.ProjectImage{}
		}

		// 获取主图（第一张图片）
		var mainImage string
		for _, img := range images {
			if img.IsPrimary {
				mainImage = img.ImageURL
				break
			}
		}
		// 如果没有设置主图，使用第一张图片
		if mainImage == "" && len(images) > 0 {
			mainImage = images[0].ImageURL
		}

		// 获取发货信息
		shipment, err := h.paymentService.GetShipmentByOrderID(order.ID)
		if err != nil || shipment == nil {
			shipment = &model.Shipment{
				Status:              "not_shipped",
				EstimatedDeliveryAt: time.Time{},
				ShippedAt:           time.Time{},
				DeliveredAt:         time.Time{},
				CreatedAt:           time.Time{},
				UpdatedAt:           time.Time{},
			}
		}

		// 获取退款信息
		refund, err := h.paymentService.GetRefundStatus(order.ID)
		if err != nil || refund == nil {
			refund = &model.RefundRequest{
				Status:    "not_requested",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
			}
		}

		// 生成订单号
		orderNumber := fmt.Sprintf("ORD-%d-%04d", time.Now().Year(), order.ID)

		// 确保所有字段都有默认值
		formattedOrder := gin.H{
			"id":                 order.ID,
			"order_number":       orderNumber,
			"product_name":       project.Title,
			"amount":             order.Amount,
			"status":             order.Status,
			"shipping_status":    shipment.Status,
			"tracking_number":    shipment.TrackingNumber,
			"created_at":         order.CreatedAt,
			"estimated_delivery": shipment.EstimatedDeliveryAt,
			"project_status":     project.Status,
			"refund_status":      refund.Status,
			"rating":             nil,
			"image":              mainImage,
			"is_reward":          order.IsReward,
		}

		formattedOrders = append(formattedOrders, formattedOrder)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"orders": formattedOrders,
		},
	})
}

func (h *PaymentHandler) GetRefundStatus(c *gin.Context) {
	orderID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid order ID",
			"details": err.Error(),
		})
		return
	}

	refundRequest, err := h.paymentService.GetRefundStatus(orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to get refund status",
			"details": err.Error(),
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

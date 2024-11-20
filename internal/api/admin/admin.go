package admin

import (
	"crowdfunding-backend/internal/model"
	"crowdfunding-backend/internal/service"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// AdminHandler 按功能模块组织处理方法
type AdminHandler struct {
	adminService *service.AdminService
}

// NewAdminHandler 创建一个新的 AdminHandler 实例
func NewAdminHandler(adminService *service.AdminService) *AdminHandler {
	return &AdminHandler{adminService}
}

// 项目管理
func (h *AdminHandler) GetProjects(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	status := c.DefaultQuery("status", "")
	keyword := c.DefaultQuery("keyword", "")

	projects, total, err := h.adminService.GetProjects(page, pageSize, status, keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取项目列表失败",
			"error":   err.Error(),
		})
		return
	}

	// 格式化项目数据
	var formattedProjects []gin.H
	for _, p := range projects {
		formattedProject := gin.H{
			"id":                p.ID,
			"title":             p.Title,
			"status":            p.Status,
			"created_at":        p.CreatedAt.Format(time.RFC3339),
			"total_amount":      p.TotalAmount,
			"total_goal_amount": p.TotalGoalAmount,
			"creator": gin.H{
				"username": p.Creator.Username,
				"email":    p.Creator.Email,
			},
		}
		formattedProjects = append(formattedProjects, formattedProject)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"projects": formattedProjects,
			"pagination": gin.H{
				"current_page": page,
				"page_size":    pageSize,
				"total":        total,
				"total_pages":  (total + pageSize - 1) / pageSize,
			},
		},
	})
}

func (h *AdminHandler) ReviewProject(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的项目ID",
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
			"message": "无效的���求数据",
		})
		return
	}

	err = h.adminService.ReviewProject(projectID, input.Approved, input.Comment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "审核项目失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "项目审核完成",
	})
}

func (h *AdminHandler) UpdateProjectStatus(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid project ID",
		})
		return
	}

	var input struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid input data",
		})
		return
	}

	// 验证状态值
	validStatuses := map[string]bool{
		"pending_review": true,
		"active":         true,
		"completed":      true,
		"failed":         true,
		"rejected":       true,
	}
	if !validStatuses[input.Status] {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid status value",
		})
		return
	}

	_, err = h.adminService.UpdateProjectStatus(projectID, input.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to update project status",
			"error":   err.Error(),
		})
		return
	}

	// 返回前端期望的响应格式
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Project status updated successfully",
	})
}

func (h *AdminHandler) DeleteProject(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的项目ID",
		})
		return
	}

	err = h.adminService.DeleteProject(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "删除项目失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "项目删除成功",
	})
}

func (h *AdminHandler) GetProjectPledgers(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的项目ID",
		})
		return
	}

	pledgers, err := h.adminService.GetProjectPledgers(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取支持者列表失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": pledgers,
	})
}

// 用户管理
func (h *AdminHandler) GetUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	users, err := h.adminService.GetUsers(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取用户列表失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": users,
	})
}

func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的用户ID",
		})
		return
	}

	var input struct {
		Role string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求数据",
		})
		return
	}

	err = h.adminService.UpdateUserRole(userID, input.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新用户角色失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "用户角色更新成功",
	})
}

// 订单和退款管理
func (h *AdminHandler) GetRefundRequests(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	requests, total, err := h.adminService.GetAllRefundRequests(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取退款申请列表失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"requests": requests,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}

func (h *AdminHandler) ProcessRefund(c *gin.Context) {
	requestID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的退款申请ID",
		})
		return
	}

	// 直接同意退款
	err = h.adminService.ProcessRefund(requestID, true, "管理员已同意退款")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "处理退款申请失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "退款申请已同意",
	})
}

// 发货管理
func (h *AdminHandler) CreateShipment(c *gin.Context) {
	var shipment model.Shipment
	if err := c.ShouldBindJSON(&shipment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的发货信息",
		})
		return
	}

	err := h.adminService.CreateShipmentAndUpdateOrder(&shipment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建发货记录失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "发货记录创建成功",
		"data":    shipment,
	})
}

func (h *AdminHandler) UpdateShipmentStatus(c *gin.Context) {
	shipmentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的发货记录ID",
		})
		return
	}

	var input struct {
		Status         string `json:"status" binding:"required"`
		TrackingNumber string `json:"tracking_number"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求数据",
		})
		return
	}

	err = h.adminService.UpdateShipmentStatus(shipmentID, input.Status, input.TrackingNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新发货状态失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "发货状态更新成功",
	})
}

// 系统管理
func (h *AdminHandler) GetSystemStats(c *gin.Context) {
	stats, err := h.adminService.GetSystemStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取系统统计数据失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": stats,
	})
}

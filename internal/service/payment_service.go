package service

import (
	"crowdfunding-backend/internal/model"
	"crowdfunding-backend/internal/repository/interfaces"
	"crowdfunding-backend/internal/util"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
)

type PaymentService struct {
	paymentRepo interfaces.PaymentRepository
	userRepo    interfaces.UserRepository
	projectRepo interfaces.ProjectRepository
	db          *sql.DB
}

// NewPaymentService 创建一个新的 PaymentService 实例
func NewPaymentService(
	paymentRepo interfaces.PaymentRepository,
	userRepo interfaces.UserRepository,
	projectRepo interfaces.ProjectRepository,
	db *sql.DB,
) *PaymentService {
	return &PaymentService{
		paymentRepo: paymentRepo,
		userRepo:    userRepo,
		projectRepo: projectRepo,
		db:          db,
	}
}

// ProcessPayment 处理支付
func (s *PaymentService) ProcessPayment(payment *model.Payment, addressID int) (*model.Order, error) {
	// 开始事务
	tx, err := s.db.Begin()
	if err != nil {
		util.Logger.Error("开始事务失败", zap.Error(err))
		return nil, err
	}
	defer tx.Rollback()

	// 获取项目信息
	project, err := s.projectRepo.GetProjectByID(payment.ProjectID)
	if err != nil {
		util.Logger.Error("获取项目信息失败", zap.Error(err))
		return nil, err
	}
	if project == nil {
		util.Logger.Error("项目不存在", zap.Int("project_id", payment.ProjectID))
		return nil, errors.New("project not found")
	}

	// 创建 pledge 记录
	pledge := &model.Pledge{
		UserID:    payment.UserID,
		ProjectID: payment.ProjectID,
		Amount:    payment.Amount,
		Status:    "pending",
		AddressID: &addressID,
		CreatedAt: time.Now(),
	}

	// 插入 pledge 记录
	query := `
		INSERT INTO pledges (user_id, project_id, amount, status, address_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`
	result, err := tx.Exec(query,
		pledge.UserID,
		pledge.ProjectID,
		pledge.Amount,
		pledge.Status,
		pledge.AddressID,
		pledge.CreatedAt)
	if err != nil {
		util.Logger.Error("创建 pledge 失败", zap.Error(err))
		return nil, fmt.Errorf("failed to create pledge: %w", err)
	}

	pledgeID, err := result.LastInsertId()
	if err != nil {
		util.Logger.Error("获取 pledge ID 失败", zap.Error(err))
		return nil, fmt.Errorf("failed to get pledge ID: %w", err)
	}
	pledge.ID = int(pledgeID)

	// 检查支付金额是否达到最低有奖支持金额
	isReward := payment.Amount >= project.MinRewardAmount

	// 创建订单
	order := &model.Order{
		OrderNumber: fmt.Sprintf("ORD-%d-%04d", time.Now().Year(), pledgeID),
		UserID:      payment.UserID,
		ProjectID:   payment.ProjectID,
		PledgeID:    pledge.ID,
		Amount:      payment.Amount,
		Status:      "pending",
		IsReward:    isReward,
		AddressID:   &addressID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 插入订单记录
	query = `
		INSERT INTO orders (
			order_number, user_id, project_id, pledge_id,
			amount, status, address_id, is_reward,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err = tx.Exec(query,
		order.OrderNumber,
		order.UserID,
		order.ProjectID,
		order.PledgeID,
		order.Amount,
		order.Status,
		order.AddressID,
		order.IsReward,
		order.CreatedAt,
		order.UpdatedAt)

	if err != nil {
		util.Logger.Error("创建订单失败", zap.Error(err))
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	orderID, err := result.LastInsertId()
	if err != nil {
		util.Logger.Error("获取订单 ID 失败", zap.Error(err))
		return nil, fmt.Errorf("failed to get order ID: %w", err)
	}
	order.ID = int(orderID)

	// 更新项目总金额和进度
	query = `
		UPDATE projects 
		SET total_amount = total_amount + ?,
			progress = CASE 
				WHEN total_goal_amount > 0 THEN ((total_amount + ?) / total_goal_amount) * 100
				ELSE 0 
			END,
			updated_at = NOW()
		WHERE id = ?`

	_, err = tx.Exec(query, payment.Amount, payment.Amount, payment.ProjectID)
	if err != nil {
		util.Logger.Error("更新项目金额失败", zap.Error(err))
		return nil, fmt.Errorf("failed to update project amount: %w", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		util.Logger.Error("提交事务失败", zap.Error(err))
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	util.Logger.Info("支付处理成功",
		zap.Int("order_id", order.ID),
		zap.String("order_number", order.OrderNumber),
		zap.Bool("is_reward", order.IsReward),
		zap.Float64("amount", payment.Amount))

	return order, nil
}

func (s *PaymentService) CreateOrder(pledge *model.Pledge, addressID int) (*model.Order, error) {
	order := &model.Order{
		UserID:    pledge.UserID,
		ProjectID: pledge.ProjectID,
		PledgeID:  pledge.ID,
		Amount:    pledge.Amount,
		Status:    "pending",
		AddressID: &addressID,
	}
	return order, s.paymentRepo.CreateOrder(order)
}

func (s *PaymentService) RequestRefund(orderID int, userID int, reason string) error {
	request := &model.RefundRequest{
		OrderID: orderID,
		UserID:  userID,
		Reason:  reason,
		Status:  "pending",
	}
	return s.paymentRepo.CreateRefundRequest(request)
}

func (s *PaymentService) ProcessRefundRequest(requestID int, approved bool, comment string) error {
	util.Logger.Info("开始处理退款申请",
		zap.Int("request_id", requestID),
		zap.Bool("approved", approved))

	// 获取退款申请信息
	request, err := s.paymentRepo.GetRefundRequestByID(requestID)
	if err != nil {
		util.Logger.Error("获取退款申请失败", zap.Error(err))
		return err
	}

	if request == nil {
		return errors.New("refund request not found")
	}

	// 开始事务
	tx, err := s.db.Begin()
	if err != nil {
		util.Logger.Error("开始事务失败", zap.Error(err))
		return err
	}
	defer tx.Rollback()

	// 更新退款申请状态
	if approved {
		request.Status = "approved"
		// 更新订单状态为已退款
		err = s.paymentRepo.UpdateOrderStatus(request.OrderID, "refunded")
		if err != nil {
			util.Logger.Error("更新订单状态失败", zap.Error(err))
			return err
		}
	} else {
		request.Status = "rejected"
		// 更新订单状态为退款被拒绝
		err = s.paymentRepo.UpdateOrderStatus(request.OrderID, "refund_rejected")
		if err != nil {
			util.Logger.Error("更新订单状态失败", zap.Error(err))
			return err
		}
	}

	request.AdminComment = comment
	err = s.paymentRepo.UpdateRefundRequest(request)
	if err != nil {
		util.Logger.Error("更新退款申请失败", zap.Error(err))
		return err
	}

	err = tx.Commit()
	if err != nil {
		util.Logger.Error("提交事务失败", zap.Error(err))
		return err
	}

	util.Logger.Info("退款申请处理完成",
		zap.Int("request_id", requestID),
		zap.String("status", request.Status))

	return nil
}

func (s *PaymentService) GetOrderByID(orderID int) (*model.Order, error) {
	order, err := s.paymentRepo.GetOrderByID(orderID)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (s *PaymentService) GetOrdersByUser(userID int) ([]*model.Order, error) {
	orders, err := s.paymentRepo.GetOrdersByUser(userID)
	if err != nil {
		return nil, err
	}
	return orders, nil
}

func (s *PaymentService) GetRefundStatus(orderID int) (*model.RefundRequest, error) {
	util.Logger.Info("开始获取退款状态", zap.Int("order_id", orderID))

	// 调用仓库层的方法获取退款状态
	refundRequest, err := s.paymentRepo.GetRefundStatus(orderID)
	if err != nil {
		util.Logger.Error("获取退款状态失败",
			zap.Error(err),
			zap.Int("order_id", orderID))
		return nil, fmt.Errorf("failed to get refund status: %w", err)
	}

	if refundRequest == nil {
		util.Logger.Info("订单没有退款申请",
			zap.Int("order_id", orderID))
		return nil, nil
	}

	util.Logger.Info("成功获取退款状态",
		zap.Int("order_id", orderID),
		zap.String("status", refundRequest.Status))

	return refundRequest, nil
}

func (s *PaymentService) GetShipmentByOrderID(orderID int) (*model.Shipment, error) {
	return s.paymentRepo.GetShipmentByOrderID(orderID)
}

// RequestRefundForFailedProject 申请退款
func (s *PaymentService) RequestRefundForFailedProject(orderID, userID int) error {
	util.Logger.Info("开始处理退款申请",
		zap.Int("order_id", orderID),
		zap.Int("user_id", userID))

	// 获取订单信息
	order, err := s.paymentRepo.GetOrderByID(orderID)
	if err != nil {
		util.Logger.Error("获取订单信息失败", zap.Error(err))
		return err
	}
	if order == nil {
		return errors.New("订单不存在")
	}

	// 验证订单属于当前用户
	if order.UserID != userID {
		util.Logger.Warn("订单不属于该用户",
			zap.Int("order_id", orderID),
			zap.Int("user_id", userID))
		return errors.New("订单不属于当前用户")
	}

	// 检查是否已经存在退款申请
	refundRequest, err := s.paymentRepo.GetRefundStatus(orderID)
	if err != nil {
		util.Logger.Error("检查退款状态失败", zap.Error(err))
		return err
	}
	if refundRequest != nil {
		util.Logger.Warn("订单已存在退款申请",
			zap.Int("order_id", orderID),
			zap.String("refund_status", refundRequest.Status))
		return fmt.Errorf("该订单已存在退款申请，状态为：%s", refundRequest.Status)
	}

	// 创建退款申请
	refundRequest = &model.RefundRequest{
		OrderID: orderID,
		UserID:  userID,
		Reason:  "用户申请退款",
		Status:  "pending",
	}

	err = s.paymentRepo.CreateRefundRequest(refundRequest)
	if err != nil {
		util.Logger.Error("创建退款申请失败", zap.Error(err))
		return err
	}

	util.Logger.Info("退款申请创建成功",
		zap.Int("order_id", orderID),
		zap.Int("refund_request_id", refundRequest.ID))

	return nil
}

// ProcessProjectFailure 处理项目失败时的订单状态更新
func (s *PaymentService) ProcessProjectFailure(projectID int) error {
	util.Logger.Info("开始处理项目失败的订单状态更新",
		zap.Int("project_id", projectID))

	// 获取项目的所有待处理订单
	orders, err := s.paymentRepo.GetOrdersByProject(projectID)
	if err != nil {
		util.Logger.Error("获取项目订单失败", zap.Error(err))
		return err
	}

	// 开始事务
	tx, err := s.db.Begin()
	if err != nil {
		util.Logger.Error("开始事务失败", zap.Error(err))
		return err
	}
	defer tx.Rollback()

	// 更新所有相关订单的状态
	for _, order := range orders {
		if order.Status == "pending" || order.Status == "paid" {
			err = s.paymentRepo.UpdateOrderStatus(order.ID, "crowdfunding_failed")
			if err != nil {
				util.Logger.Error("更新订单状态失败",
					zap.Error(err),
					zap.Int("order_id", order.ID))
				return err
			}
			util.Logger.Info("订单状态已更新为众筹失败",
				zap.Int("order_id", order.ID))
		}
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		util.Logger.Error("提交事务失败", zap.Error(err))
		return err
	}

	util.Logger.Info("项目失败订单状态更新完成",
		zap.Int("project_id", projectID),
		zap.Int("updated_orders", len(orders)))

	return nil
}

// SyncOrdersWithProjectStatus 同步订单状态与项目状态
func (s *PaymentService) SyncOrdersWithProjectStatus(projectID int) error {
	util.Logger.Info("开始同步订单状态与项目状态",
		zap.Int("project_id", projectID))

	// 获取项目信息
	project, err := s.projectRepo.GetProjectByID(projectID)
	if err != nil {
		util.Logger.Error("获取项目信息失败", zap.Error(err))
		return err
	}
	if project == nil {
		return errors.New("project not found")
	}

	// 如果项目状态是失败，更新所有相关订单
	if project.Status == "failed" {
		// 获取项目的所有订单
		orders, err := s.paymentRepo.GetOrdersByProject(projectID)
		if err != nil {
			util.Logger.Error("获取项目订单失败", zap.Error(err))
			return err
		}

		// 开始事务
		tx, err := s.db.Begin()
		if err != nil {
			util.Logger.Error("开始事务失败", zap.Error(err))
			return err
		}
		defer tx.Rollback()

		// 更新所有相关订单的状态
		for _, order := range orders {
			if order.Status == "pending" || order.Status == "paid" {
				err = s.paymentRepo.UpdateOrderStatus(order.ID, "crowdfunding_failed")
				if err != nil {
					util.Logger.Error("更新订单状态失败",
						zap.Error(err),
						zap.Int("order_id", order.ID))
					return err
				}
				util.Logger.Info("订单状态已更新为众筹失败",
					zap.Int("order_id", order.ID))
			}
		}

		// 提交事务
		if err = tx.Commit(); err != nil {
			util.Logger.Error("提交事务失败", zap.Error(err))
			return err
		}

		util.Logger.Info("订单状态同步完成",
			zap.Int("project_id", projectID),
			zap.Int("updated_orders", len(orders)))
	}

	return nil
}

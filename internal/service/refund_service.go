package service

import (
	"crowdfunding-backend/internal/model"
	"crowdfunding-backend/internal/repository/interfaces"
	"database/sql"
	"errors"
	"fmt"
)

type RefundService struct {
	paymentRepo interfaces.PaymentRepository
	db          *sql.DB
}

func NewRefundService(paymentRepo interfaces.PaymentRepository, db *sql.DB) *RefundService {
	return &RefundService{
		paymentRepo: paymentRepo,
		db:          db,
	}
}

// RequestRefund 申请退款
func (s *RefundService) RequestRefund(orderID, userID int, reason string) error {
	// 获取订单信息
	order, err := s.paymentRepo.GetOrderByID(orderID)
	if err != nil {
		return err
	}
	if order == nil {
		return errors.New("订单不存在")
	}

	// 验证订单属于当前用户
	if order.UserID != userID {
		return errors.New("订单不属于当前用户")
	}

	// 检查是否已经存在退款申请
	refundRequest, err := s.paymentRepo.GetRefundStatus(orderID)
	if err != nil {
		return err
	}
	if refundRequest != nil {
		return fmt.Errorf("该订单已存在退款申请，状态为：%s", refundRequest.Status)
	}

	// 创建退款申请
	refundRequest = &model.RefundRequest{
		OrderID: orderID,
		UserID:  userID,
		Reason:  reason,
		Status:  "pending",
	}

	return s.paymentRepo.CreateRefundRequest(refundRequest)
}

// ProcessRefund 处理退款申请
func (s *RefundService) ProcessRefund(requestID int, approved bool, comment string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 获取退款申请信息
	request, err := s.paymentRepo.GetRefundRequestByID(requestID)
	if err != nil {
		return err
	}
	if request == nil {
		return errors.New("退款申请不存在")
	}

	// 更新退款申请状态
	if approved {
		request.Status = "approved"
		err = s.paymentRepo.UpdateOrderStatus(request.OrderID, "refunded")
	} else {
		request.Status = "rejected"
		err = s.paymentRepo.UpdateOrderStatus(request.OrderID, "refund_rejected")
	}
	if err != nil {
		return err
	}

	request.AdminComment = comment
	err = s.paymentRepo.UpdateRefundRequest(request)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetRefundStatus 获取退款状态
func (s *RefundService) GetRefundStatus(orderID int) (*model.RefundRequest, error) {
	return s.paymentRepo.GetRefundStatus(orderID)
}

// AutoRefundForFailedProject 为众筹失败的项目自动创建退款申请
func (s *RefundService) AutoRefundForFailedProject(projectID int) error {
	orders, err := s.paymentRepo.GetOrdersByProject(projectID)
	if err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, order := range orders {
		if order.Status == "pending" || order.Status == "paid" {
			// 创建退款申请
			refundRequest := &model.RefundRequest{
				OrderID: order.ID,
				UserID:  order.UserID,
				Reason:  "项目众筹失败自动退款",
				Status:  "approved",
			}

			err = s.paymentRepo.CreateRefundRequest(refundRequest)
			if err != nil {
				return err
			}

			err = s.paymentRepo.UpdateOrderStatus(order.ID, "refunded")
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

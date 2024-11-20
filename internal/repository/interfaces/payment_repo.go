package interfaces

import "crowdfunding-backend/internal/model"

type PaymentRepository interface {
	CreatePayment(payment *model.Payment) error
	CreateOrder(order *model.Order) error
	UpdateOrderStatus(orderID int, status string) error
	GetOrderByID(id int) (*model.Order, error)
	GetOrdersByUser(userID int) ([]*model.Order, error)
	GetOrdersByProject(projectID int) ([]*model.Order, error)
	CreateRefundRequest(request *model.RefundRequest) error
	UpdateRefundRequest(request *model.RefundRequest) error
	GetRefundRequestsByUser(userID int) ([]*model.RefundRequest, error)
	GetPendingRefundRequests() ([]*model.RefundRequest, error)
	CreatePledge(pledge *model.Pledge) error
	GetShipmentByOrderID(orderID int) (*model.Shipment, error)
	UpdateOrderAfterShipment(orderID int, shipmentID int) error
	GetRefundRequestByID(requestID int) (*model.RefundRequest, error)
	CheckProjectGoalStatus(projectID int) (bool, error)
	UpdateOrdersToFailedByProject(projectID int) error
	CheckProjectEndDate(projectID int) (bool, error)
	GetRefundStatus(orderID int) (*model.RefundRequest, error)
	GetAllRefundRequests(page, pageSize int) ([]*model.RefundRequest, int, error)
}

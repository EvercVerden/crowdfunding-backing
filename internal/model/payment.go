package model

import "time"

type Payment struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	ProjectID int       `json:"project_id"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Refund struct {
	ID        int       `json:"id"`
	PaymentID int       `json:"payment_id"`
	Amount    float64   `json:"amount"`
	Reason    string    `json:"reason"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Order 订单模型
type Order struct {
	ID          int          `json:"id"`
	OrderNumber string       `json:"order_number"`
	UserID      int          `json:"user_id"`
	ProjectID   int          `json:"project_id"`
	PledgeID    int          `json:"pledge_id"`
	Amount      float64      `json:"amount"`
	Status      string       `json:"status"`
	IsReward    bool         `json:"is_reward"`
	AddressID   *int         `json:"address_id,omitempty"`
	Address     *UserAddress `json:"address,omitempty"`
	Shipment    *Shipment    `json:"shipment,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// RefundRequest 退款申请模型
type RefundRequest struct {
	ID           int       `json:"id"`
	OrderID      int       `json:"order_id"`
	UserID       int       `json:"user_id"`
	Reason       string    `json:"reason"`
	Status       string    `json:"status"`
	AdminComment string    `json:"admin_comment,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Order        *Order    `json:"order,omitempty"`
	User         *User     `json:"user,omitempty"`
	ProjectTitle string    `json:"project_title,omitempty"`
}

type Shipment struct {
	ID                  int          `json:"id"`
	ProjectID           int          `json:"project_id"`
	UserID              int          `json:"user_id"`
	OrderID             int          `json:"order_id"`
	AddressID           int          `json:"address_id"`
	Status              string       `json:"status"`
	TrackingNumber      string       `json:"tracking_number,omitempty"`
	ShippingCompany     string       `json:"shipping_company,omitempty"`
	ShippedAt           time.Time    `json:"shipped_at,omitempty"`
	DeliveredAt         time.Time    `json:"delivered_at,omitempty"`
	EstimatedDeliveryAt time.Time    `json:"estimated_delivery_at,omitempty"`
	CreatedAt           time.Time    `json:"created_at"`
	UpdatedAt           time.Time    `json:"updated_at"`
	Address             *UserAddress `json:"address,omitempty"`
}

package mysql

import (
	"crowdfunding-backend/internal/model"
	"crowdfunding-backend/internal/util"
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"
)

type PaymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db}
}

func (r *PaymentRepository) CreatePayment(payment *model.Payment) error {
	util.Logger.Info("开始创建支付记录",
		zap.Int("user_id", payment.UserID),
		zap.Int("project_id", payment.ProjectID),
		zap.Float64("amount", payment.Amount),
		zap.String("status", payment.Status))

	// 设置创建时间
	payment.CreatedAt = time.Now()
	payment.UpdatedAt = time.Now()

	query := `INSERT INTO payments (user_id, project_id, amount, status, created_at, updated_at) 
			  VALUES (?, ?, ?, ?, ?, ?)`

	util.Logger.Debug("执行支付记录插入SQL",
		zap.String("query", query),
		zap.Any("params", []interface{}{
			payment.UserID,
			payment.ProjectID,
			payment.Amount,
			payment.Status,
			payment.CreatedAt,
			payment.UpdatedAt,
		}))

	result, err := r.db.Exec(query,
		payment.UserID,
		payment.ProjectID,
		payment.Amount,
		payment.Status,
		payment.CreatedAt,
		payment.UpdatedAt)

	if err != nil {
		util.Logger.Error("创建支付记录失败",
			zap.Error(err),
			zap.Any("payment", payment),
			zap.String("error_type", fmt.Sprintf("%T", err)))
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		util.Logger.Error("获取支付记录ID失败", zap.Error(err))
		return err
	}

	payment.ID = int(id)
	util.Logger.Info("支付记录创建成功",
		zap.Int("payment_id", payment.ID),
		zap.String("status", payment.Status))
	return nil
}

func (r *PaymentRepository) CreateOrder(order *model.Order) error {
	util.Logger.Info("开始创建订单",
		zap.Int("user_id", order.UserID),
		zap.Int("project_id", order.ProjectID),
		zap.Float64("amount", order.Amount),
		zap.Any("address_id", order.AddressID))

	// 验证必要字段
	if order.UserID == 0 || order.ProjectID == 0 || order.Amount <= 0 {
		util.Logger.Error("订单参数验证失败",
			zap.Int("user_id", order.UserID),
			zap.Int("project_id", order.ProjectID),
			zap.Float64("amount", order.Amount))
		return fmt.Errorf("invalid order parameters")
	}

	// 验证地址ID
	if order.AddressID == nil {
		util.Logger.Error("缺少收货地址")
		return fmt.Errorf("shipping address is required")
	}

	tx, err := r.db.Begin()
	if err != nil {
		util.Logger.Error("开始事务失败", zap.Error(err))
		return err
	}
	defer tx.Rollback()

	// 设置初始状态
	if order.Status == "" {
		order.Status = "pending"
	}

	// 先插入订单获取ID
	query := `INSERT INTO orders (order_number, user_id, project_id, pledge_id, amount, status, address_id, created_at)
			  VALUES ('TEMP', ?, ?, ?, ?, ?, ?, NOW())`

	util.Logger.Debug("执行订单插入SQL",
		zap.String("query", query),
		zap.Any("params", []interface{}{
			order.UserID,
			order.ProjectID,
			order.PledgeID,
			order.Amount,
			order.Status,
			order.AddressID,
		}))

	result, err := tx.Exec(query,
		order.UserID, order.ProjectID, order.PledgeID,
		order.Amount, order.Status, order.AddressID)
	if err != nil {
		util.Logger.Error("插入订单记录失败",
			zap.Error(err),
			zap.String("error_type", fmt.Sprintf("%T", err)))
		return fmt.Errorf("failed to insert order: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		util.Logger.Error("获取订单ID失败", zap.Error(err))
		return fmt.Errorf("failed to get order ID: %w", err)
	}
	order.ID = int(id)

	// 生成并更新订单编号
	orderNumber := generateOrderNumber(order.ID)
	updateQuery := "UPDATE orders SET order_number = ? WHERE id = ?"

	util.Logger.Debug("执行订单编号更新SQL",
		zap.String("query", updateQuery),
		zap.String("order_number", orderNumber),
		zap.Int("order_id", order.ID))

	_, err = tx.Exec(updateQuery, orderNumber, order.ID)
	if err != nil {
		util.Logger.Error("更新订单编号失败",
			zap.Error(err),
			zap.String("order_number", orderNumber),
			zap.Int("order_id", order.ID))
		return fmt.Errorf("failed to update order number: %w", err)
	}

	order.OrderNumber = orderNumber

	if err := tx.Commit(); err != nil {
		util.Logger.Error("提交事务失败", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	util.Logger.Info("订单创建成功",
		zap.Int("order_id", order.ID),
		zap.String("order_number", order.OrderNumber),
		zap.String("status", order.Status))
	return nil
}

// generateOrderNumber 生成订单编号
// 格式: ORD-年份-4位序号，例如: ORD-2024-0001
func generateOrderNumber(orderID int) string {
	year := time.Now().Year()
	return fmt.Sprintf("ORD-%d-%04d", year, orderID)
}

func (r *PaymentRepository) UpdateOrderStatus(orderID int, status string) error {
	query := `UPDATE orders SET status = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.db.Exec(query, status, orderID)
	return err
}

func (r *PaymentRepository) GetOrderByID(id int) (*model.Order, error) {
	util.Logger.Info("开始获取订单详情", zap.Int("order_id", id))

	query := `
		SELECT o.id, o.order_number, o.user_id, o.project_id, o.pledge_id, 
			   o.amount, o.status, o.address_id, o.is_reward, o.created_at, o.updated_at,
			   a.id, a.user_id, a.receiver_name, a.phone, a.province, a.city, 
			   a.district, a.detail_address, a.is_default, a.created_at, a.updated_at,
			   COALESCE(s.status, '') as shipment_status,
			   COALESCE(s.tracking_number, '') as tracking_number,
			   COALESCE(s.shipping_company, '') as shipping_company,
			   s.shipped_at, s.estimated_delivery_at
		FROM orders o
		LEFT JOIN user_addresses a ON o.address_id = a.id
		LEFT JOIN shipments s ON o.id = s.order_id
		WHERE o.id = ?`

	var order model.Order
	var address model.UserAddress
	var shipment model.Shipment
	var addressID sql.NullInt64
	var shippedAt, estimatedDeliveryAt sql.NullTime
	var shipmentStatus, trackingNumber, shippingCompany string

	err := r.db.QueryRow(query, id).Scan(
		&order.ID, &order.OrderNumber, &order.UserID, &order.ProjectID, &order.PledgeID,
		&order.Amount, &order.Status, &addressID, &order.IsReward, &order.CreatedAt, &order.UpdatedAt,
		&address.ID, &address.UserID, &address.ReceiverName, &address.Phone,
		&address.Province, &address.City, &address.District, &address.DetailAddress,
		&address.IsDefault, &address.CreatedAt, &address.UpdatedAt,
		&shipmentStatus, &trackingNumber, &shippingCompany,
		&shippedAt, &estimatedDeliveryAt)

	if err != nil {
		if err == sql.ErrNoRows {
			util.Logger.Info("订单不存在", zap.Int("order_id", id))
			return nil, nil
		}
		util.Logger.Error("查询订单失败",
			zap.Error(err),
			zap.Int("order_id", id))
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// 处理可能为 NULL 的字段
	if addressID.Valid {
		order.AddressID = &[]int{int(addressID.Int64)}[0]
		order.Address = &address
	}

	// 处理可能为 NULL 的时间字段
	if shippedAt.Valid {
		shipment.ShippedAt = shippedAt.Time
	}
	if estimatedDeliveryAt.Valid {
		shipment.EstimatedDeliveryAt = estimatedDeliveryAt.Time
	}

	// 设置发货信息
	shipment.Status = shipmentStatus
	shipment.TrackingNumber = trackingNumber
	shipment.ShippingCompany = shippingCompany

	// 只有当有发货信息时才设置发货信息
	if shipment.Status != "" {
		order.Shipment = &shipment
	}

	util.Logger.Info("成功获取订单详情",
		zap.Int("order_id", id),
		zap.String("status", order.Status))

	return &order, nil
}

func (r *PaymentRepository) GetOrdersByUser(userID int) ([]*model.Order, error) {
	util.Logger.Info("开始获取用户订单列表",
		zap.Int("user_id", userID))

	query := `
		SELECT o.id, o.order_number, o.user_id, o.project_id, o.pledge_id, 
			   o.amount, o.status, o.address_id, o.is_reward, o.created_at, o.updated_at,
			   a.id, a.user_id, a.receiver_name, a.phone, a.province, a.city, 
			   a.district, a.detail_address, a.is_default, a.created_at, a.updated_at
		FROM orders o
		LEFT JOIN user_addresses a ON o.address_id = a.id
		WHERE o.user_id = ?
		ORDER BY o.created_at DESC`

	util.Logger.Debug("执行SQL查询",
		zap.String("query", query),
		zap.Int("user_id", userID))

	rows, err := r.db.Query(query, userID)
	if err != nil {
		util.Logger.Error("查询订单失败",
			zap.Error(err),
			zap.Int("user_id", userID))
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var order model.Order
		var address model.UserAddress
		var addressID sql.NullInt64

		err := rows.Scan(
			&order.ID, &order.OrderNumber, &order.UserID, &order.ProjectID, &order.PledgeID,
			&order.Amount, &order.Status, &addressID, &order.IsReward, &order.CreatedAt, &order.UpdatedAt,
			&address.ID, &address.UserID, &address.ReceiverName, &address.Phone,
			&address.Province, &address.City, &address.District, &address.DetailAddress,
			&address.IsDefault, &address.CreatedAt, &address.UpdatedAt,
		)
		if err != nil {
			util.Logger.Error("扫描订单数据失败",
				zap.Error(err),
				zap.Int("user_id", userID),
				zap.String("error_type", fmt.Sprintf("%T", err)))
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}

		util.Logger.Debug("成功扫描订单数据",
			zap.Int("order_id", order.ID),
			zap.String("order_number", order.OrderNumber),
			zap.String("status", order.Status),
			zap.Bool("has_address", addressID.Valid))

		// 只有当地址ID存在时才设置地址信息
		if addressID.Valid {
			order.AddressID = &[]int{int(addressID.Int64)}[0]
			order.Address = &address
			util.Logger.Debug("设置订单地址信息",
				zap.Int("order_id", order.ID),
				zap.Int("address_id", *order.AddressID))
		}

		orders = append(orders, &order)
	}

	if err = rows.Err(); err != nil {
		util.Logger.Error("遍历订单数据时发生错误",
			zap.Error(err),
			zap.Int("user_id", userID))
		return nil, fmt.Errorf("error iterating orders: %w", err)
	}

	util.Logger.Info("成功获取用户订单列表",
		zap.Int("user_id", userID),
		zap.Int("order_count", len(orders)))

	return orders, nil
}

func (r *PaymentRepository) GetOrdersByProject(projectID int) ([]*model.Order, error) {
	query := `
			SELECT o.*, a.*
			FROM orders o
			LEFT JOIN user_addresses a ON o.address_id = a.id
			WHERE o.project_id = ?
			ORDER BY o.created_at DESC`

	rows, err := r.db.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var order model.Order
		var address model.UserAddress
		err := rows.Scan(
			&order.ID, &order.UserID, &order.ProjectID, &order.PledgeID,
			&order.Amount, &order.Status, &order.AddressID, &order.CreatedAt,
			&order.UpdatedAt,
			&address.ID, &address.UserID, &address.ReceiverName, &address.Phone,
			&address.Province, &address.City, &address.District,
			&address.DetailAddress, &address.IsDefault,
			&address.CreatedAt, &address.UpdatedAt)
		if err != nil {
			return nil, err
		}
		order.Address = &address
		orders = append(orders, &order)
	}
	return orders, nil
}

func (r *PaymentRepository) CreateRefundRequest(request *model.RefundRequest) error {
	query := `INSERT INTO refund_requests (order_id, user_id, reason, status, created_at)
			  VALUES (?, ?, ?, ?, NOW())`
	result, err := r.db.Exec(query,
		request.OrderID, request.UserID, request.Reason, request.Status)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	request.ID = int(id)
	return nil
}

func (r *PaymentRepository) UpdateRefundRequest(request *model.RefundRequest) error {
	query := `UPDATE refund_requests 
			  SET status = ?, admin_comment = ?, updated_at = NOW()
			  WHERE id = ?`
	_, err := r.db.Exec(query,
		request.Status, request.AdminComment, request.ID)
	return err
}

func (r *PaymentRepository) GetRefundRequestsByUser(userID int) ([]*model.RefundRequest, error) {
	util.Logger.Info("开始获取用户退款请列表", zap.Int("user_id", userID))

	query := `
		SELECT r.id, r.order_id, r.user_id, r.reason, r.status, 
			   COALESCE(r.admin_comment, '') as admin_comment,
			   r.created_at, r.updated_at,
			   o.order_number, o.amount,
			   p.title as project_title
			FROM refund_requests r
			LEFT JOIN orders o ON r.order_id = o.id
			LEFT JOIN projects p ON o.project_id = p.id
			WHERE r.user_id = ?
			ORDER BY r.created_at DESC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		util.Logger.Error("查询退款申请失败",
			zap.Error(err),
			zap.Int("user_id", userID))
		return nil, fmt.Errorf("failed to query refund requests: %w", err)
	}
	defer rows.Close()

	var requests []*model.RefundRequest
	for rows.Next() {
		var req model.RefundRequest
		var order model.Order
		var projectTitle string

		err := rows.Scan(
			&req.ID, &req.OrderID, &req.UserID, &req.Reason, &req.Status,
			&req.AdminComment, &req.CreatedAt, &req.UpdatedAt,
			&order.OrderNumber, &order.Amount,
			&projectTitle)
		if err != nil {
			util.Logger.Error("扫描退款申请数据失败",
				zap.Error(err),
				zap.Int("user_id", userID))
			return nil, fmt.Errorf("failed to scan refund request: %w", err)
		}

		order.ID = req.OrderID
		req.Order = &order
		req.ProjectTitle = projectTitle
		requests = append(requests, &req)
	}

	if err = rows.Err(); err != nil {
		util.Logger.Error("遍历退款申请数据失败",
			zap.Error(err),
			zap.Int("user_id", userID))
		return nil, fmt.Errorf("error iterating refund requests: %w", err)
	}

	util.Logger.Info("成功获取用户退款申请列表",
		zap.Int("user_id", userID),
		zap.Int("count", len(requests)))

	return requests, nil
}

func (r *PaymentRepository) GetPendingRefundRequests() ([]*model.RefundRequest, error) {
	util.Logger.Info("开始获取待处理退款申请列表")

	query := `
		SELECT r.id, r.order_id, r.user_id, r.reason, r.status, 
			   COALESCE(r.admin_comment, '') as admin_comment,
			   r.created_at, r.updated_at,
			   o.order_number, o.amount,
			   u.username, u.email,
			   p.title as project_title
		FROM refund_requests r
		LEFT JOIN orders o ON r.order_id = o.id
		LEFT JOIN users u ON r.user_id = u.id
		LEFT JOIN projects p ON o.project_id = p.id
		WHERE r.status = 'pending'
		ORDER BY r.created_at ASC`

	rows, err := r.db.Query(query)
	if err != nil {
		util.Logger.Error("查询待处理退款申请失败", zap.Error(err))
		return nil, fmt.Errorf("failed to query pending refund requests: %w", err)
	}
	defer rows.Close()

	var requests []*model.RefundRequest
	for rows.Next() {
		var req model.RefundRequest
		var order model.Order
		var user model.User
		var projectTitle string

		err := rows.Scan(
			&req.ID, &req.OrderID, &req.UserID, &req.Reason, &req.Status,
			&req.AdminComment, &req.CreatedAt, &req.UpdatedAt,
			&order.OrderNumber, &order.Amount,
			&user.Username, &user.Email,
			&projectTitle)
		if err != nil {
			util.Logger.Error("扫描退款申请数据失败", zap.Error(err))
			return nil, fmt.Errorf("failed to scan refund request: %w", err)
		}

		order.ID = req.OrderID
		req.Order = &order
		req.User = &user
		req.ProjectTitle = projectTitle
		requests = append(requests, &req)
	}

	util.Logger.Info("成功获取待处理退款申请列表",
		zap.Int("count", len(requests)))

	return requests, nil
}

func (r *PaymentRepository) CreatePledge(pledge *model.Pledge) error {
	query := `INSERT INTO pledges (user_id, project_id, amount, status, address_id, created_at)
			  VALUES (?, ?, ?, ?, ?, NOW())`
	result, err := r.db.Exec(query,
		pledge.UserID, pledge.ProjectID, pledge.Amount,
		pledge.Status, pledge.AddressID)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	pledge.ID = int(id)
	return nil
}

func (r *PaymentRepository) GetShipmentByOrderID(orderID int) (*model.Shipment, error) {
	query := `
		SELECT s.*, a.*
		FROM shipments s
		LEFT JOIN user_addresses a ON s.address_id = a.id
		WHERE s.order_id = ?`

	var shipment model.Shipment
	var address model.UserAddress
	err := r.db.QueryRow(query, orderID).Scan(
		&shipment.ID, &shipment.ProjectID, &shipment.UserID, &shipment.AddressID,
		&shipment.Status, &shipment.TrackingNumber, &shipment.ShippingCompany,
		&shipment.ShippedAt, &shipment.DeliveredAt, &shipment.EstimatedDeliveryAt,
		&shipment.CreatedAt, &shipment.UpdatedAt,
		&address.ID, &address.UserID, &address.ReceiverName, &address.Phone,
		&address.Province, &address.City, &address.District,
		&address.DetailAddress, &address.IsDefault,
		&address.CreatedAt, &address.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	shipment.Address = &address
	return &shipment, nil
}

func (r *PaymentRepository) UpdateOrderAfterShipment(orderID int, shipmentID int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 更新订单状态为已发货
	_, err = tx.Exec(`
		UPDATE orders 
		SET status = 'shipped', updated_at = NOW() 
		WHERE id = ?`, orderID)
	if err != nil {
		return err
	}

	// 更新发货记录状态
	_, err = tx.Exec(`
		UPDATE shipments 
		SET status = 'shipped', shipped_at = NOW(), updated_at = NOW() 
		WHERE id = ?`, shipmentID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// CheckProjectGoalStatus 检查项目第一个目标是否达成
func (r *PaymentRepository) CheckProjectGoalStatus(projectID int) (bool, error) {
	query := `
		SELECT pg.amount <= p.total_amount
		FROM project_goals pg
		JOIN projects p ON p.id = pg.project_id
		WHERE p.id = ?
		ORDER BY pg.amount ASC
		LIMIT 1`

	var isAchieved bool
	err := r.db.QueryRow(query, projectID).Scan(&isAchieved)
	if err != nil {
		return false, err
	}
	return isAchieved, nil
}

// UpdateOrdersToFailedByProject 将项目的所有订单更新为众筹失败状态
func (r *PaymentRepository) UpdateOrdersToFailedByProject(projectID int) error {
	query := `
		UPDATE orders 
		SET status = 'crowdfunding_failed', 
			updated_at = NOW() 
		WHERE project_id = ? 
		AND status = 'paid'`

	_, err := r.db.Exec(query, projectID)
	return err
}

// GetRefundRequestByID 通过ID获取退款申请
func (r *PaymentRepository) GetRefundRequestByID(requestID int) (*model.RefundRequest, error) {
	query := `
		SELECT r.*, o.*
		FROM refund_requests r
		LEFT JOIN orders o ON r.order_id = o.id
		WHERE r.id = ?`

	var request model.RefundRequest
	var order model.Order
	err := r.db.QueryRow(query, requestID).Scan(
		&request.ID, &request.OrderID, &request.UserID, &request.Reason,
		&request.Status, &request.AdminComment, &request.CreatedAt,
		&request.UpdatedAt,
		&order.ID, &order.UserID, &order.ProjectID, &order.PledgeID,
		&order.Amount, &order.Status, &order.AddressID,
		&order.CreatedAt, &order.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	request.Order = &order
	return &request, nil
}

// CheckProjectEndDate 检查项目是否已截止
func (r *PaymentRepository) CheckProjectEndDate(projectID int) (bool, error) {
	query := `
		SELECT end_date <= NOW()
		FROM projects
		WHERE id = ?`

	var isEnded bool
	err := r.db.QueryRow(query, projectID).Scan(&isEnded)
	if err != nil {
		return false, err
	}
	return isEnded, nil
}

// GetRefundStatus 获取订单的退款状态
func (r *PaymentRepository) GetRefundStatus(orderID int) (*model.RefundRequest, error) {
	util.Logger.Info("开始获取订单退款状态", zap.Int("order_id", orderID))

	query := `
		SELECT id, order_id, user_id, reason, status, 
			   COALESCE(admin_comment, '') as admin_comment, 
			   created_at, updated_at
		FROM refund_requests
		WHERE order_id = ?
		ORDER BY created_at DESC
		LIMIT 1`

	var request model.RefundRequest
	var adminComment string

	err := r.db.QueryRow(query, orderID).Scan(
		&request.ID,
		&request.OrderID,
		&request.UserID,
		&request.Reason,
		&request.Status,
		&adminComment,
		&request.CreatedAt,
		&request.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		util.Logger.Info("订单没有退款申请",
			zap.Int("order_id", orderID))
		return nil, nil
	}

	if err != nil {
		util.Logger.Error("查询退款状态失败",
			zap.Error(err),
			zap.Int("order_id", orderID))
		return nil, fmt.Errorf("failed to get refund status: %w", err)
	}

	// 设置管理员评论
	request.AdminComment = adminComment

	util.Logger.Info("成功获取订单退款状态",
		zap.Int("order_id", orderID),
		zap.String("status", request.Status))

	return &request, nil
}

// GetAllRefundRequests 获取所有退款申请（管理员用）
func (r *PaymentRepository) GetAllRefundRequests(page, pageSize int) ([]*model.RefundRequest, int, error) {
	util.Logger.Info("开始获取所有退款申请",
		zap.Int("page", page),
		zap.Int("pageSize", pageSize))

	// 获取总数
	var total int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM refund_requests`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	query := `
		SELECT r.id, r.order_id, r.user_id, r.reason, r.status, 
			   COALESCE(r.admin_comment, '') as admin_comment,
			   r.created_at, r.updated_at,
			   o.order_number, o.amount, o.project_id,
			   u.username, u.email,
			   p.title as project_title,
			   p.status as project_status
		FROM refund_requests r
		LEFT JOIN orders o ON r.order_id = o.id
		LEFT JOIN users u ON r.user_id = u.id
		LEFT JOIN projects p ON o.project_id = p.id
		ORDER BY r.created_at DESC
			LIMIT ? OFFSET ?`

	offset := (page - 1) * pageSize
	rows, err := r.db.Query(query, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var requests []*model.RefundRequest
	for rows.Next() {
		var req model.RefundRequest
		var order model.Order
		var user model.User
		var projectTitle string
		var projectStatus string

		err := rows.Scan(
			&req.ID, &req.OrderID, &req.UserID, &req.Reason, &req.Status,
			&req.AdminComment, &req.CreatedAt, &req.UpdatedAt,
			&order.OrderNumber, &order.Amount, &order.ProjectID,
			&user.Username, &user.Email,
			&projectTitle, &projectStatus)
		if err != nil {
			return nil, 0, err
		}

		order.ID = req.OrderID
		req.Order = &order
		req.User = &user
		req.ProjectTitle = projectTitle
		requests = append(requests, &req)
	}

	util.Logger.Info("成功获取退款申请列表",
		zap.Int("total", total),
		zap.Int("count", len(requests)))

	return requests, total, nil
}

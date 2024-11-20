package model

// SystemStats 系统统计数据
type SystemStats struct {
	TotalUsers     int     `json:"total_users"`
	TotalProjects  int     `json:"total_projects"`
	TotalOrders    int     `json:"total_orders"`
	TotalAmount    float64 `json:"total_amount"`
	ActiveProjects int     `json:"active_projects"`
	PendingOrders  int     `json:"pending_orders"`
}

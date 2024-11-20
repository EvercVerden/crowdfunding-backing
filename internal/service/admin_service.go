package service

import (
	"crowdfunding-backend/internal/model"
	"crowdfunding-backend/internal/repository/interfaces"
	"database/sql"
	"errors"
)

// AdminService 按功能模块组织业务逻辑
type AdminService struct {
	userRepo    interfaces.UserRepository
	projectRepo interfaces.ProjectRepository
	paymentRepo interfaces.PaymentRepository
	db          *sql.DB
}

// NewAdminService 创建一个新的 AdminService 实例
func NewAdminService(userRepo interfaces.UserRepository, projectRepo interfaces.ProjectRepository, paymentRepo interfaces.PaymentRepository, db *sql.DB) *AdminService {
	return &AdminService{
		userRepo:    userRepo,
		projectRepo: projectRepo,
		paymentRepo: paymentRepo,
		db:          db,
	}
}

// 项目管理
func (s *AdminService) GetProjects(page, pageSize int, status, search string) ([]*model.Project, int, error) {
	return s.projectRepo.GetProjectsForAdmin(page, pageSize, status, search)
}

func (s *AdminService) ReviewProject(projectID int, approved bool, comment string) error {
	project, err := s.projectRepo.GetProjectByID(projectID)
	if err != nil {
		return err
	}
	if project == nil {
		return errors.New("project not found")
	}

	if approved {
		project.Status = "active"
	} else {
		project.Status = "rejected"
	}

	return s.projectRepo.UpdateProjectStatus(project)
}

func (s *AdminService) UpdateProjectStatus(projectID int, status string) (*model.Project, error) {
	project, err := s.projectRepo.GetProjectByID(projectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, errors.New("project not found")
	}

	project.Status = status
	err = s.projectRepo.UpdateProjectStatus(project)
	if err != nil {
		return nil, err
	}

	return project, nil
}

func (s *AdminService) DeleteProject(projectID int) error {
	return s.projectRepo.DeleteProject(projectID)
}

func (s *AdminService) GetProjectPledgers(projectID int) ([]*model.Pledge, error) {
	return s.projectRepo.GetProjectSuccessfulPledgers(projectID)
}

// 用户管理
func (s *AdminService) GetUsers(page, pageSize int) ([]*model.User, error) {
	return s.userRepo.FindAll(page, pageSize)
}

func (s *AdminService) UpdateUserRole(userID int, role string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	user.Role = role
	return s.userRepo.Update(user)
}

// 订单和退款管理
func (s *AdminService) GetAllRefundRequests(page, pageSize int) ([]*model.RefundRequest, int, error) {
	return s.paymentRepo.GetAllRefundRequests(page, pageSize)
}

func (s *AdminService) ProcessRefund(requestID int, approved bool, comment string) error {
	request, err := s.paymentRepo.GetRefundRequestByID(requestID)
	if err != nil {
		return err
	}
	if request == nil {
		return errors.New("refund request not found")
	}

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
	return s.paymentRepo.UpdateRefundRequest(request)
}

// 发货管理
func (s *AdminService) CreateShipmentAndUpdateOrder(shipment *model.Shipment) error {
	err := s.projectRepo.CreateShipment(shipment)
	if err != nil {
		return err
	}

	return s.paymentRepo.UpdateOrderAfterShipment(shipment.OrderID, shipment.ID)
}

func (s *AdminService) UpdateShipmentStatus(shipmentID int, status, trackingNumber string) error {
	shipment := &model.Shipment{
		ID:             shipmentID,
		Status:         status,
		TrackingNumber: trackingNumber,
	}
	return s.projectRepo.UpdateShipment(shipment)
}

// 系统管理
func (s *AdminService) GetSystemStats() (*model.SystemStats, error) {
	stats := &model.SystemStats{}

	// 获取用户总数
	userCount, err := s.userRepo.Count()
	if err != nil {
		return nil, err
	}
	stats.TotalUsers = userCount

	// TODO: 实现其他统计数据的获取
	// 这里需要添加获取项目、订单等统计数据的逻辑

	return stats, nil
}

func (s *AdminService) CheckExpiredProjects() error {
	projects, err := s.projectRepo.GetExpiredActiveProjects()
	if err != nil {
		return err
	}

	for _, project := range projects {
		project.Status = "failed"
		err = s.projectRepo.UpdateProjectStatus(project)
		if err != nil {
			return err
		}
	}

	return nil
}

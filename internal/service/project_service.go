package service

import (
	"crowdfunding-backend/internal/model"
	"crowdfunding-backend/internal/repository/interfaces"
	"crowdfunding-backend/internal/util"
	"errors"
	"time"

	"go.uber.org/zap"
)

// ProjectService 处理与项目相关的业务逻辑
type ProjectService struct {
	repo interfaces.ProjectRepository
}

// NewProjectService 创建一个新的 ProjectService 实例
func NewProjectService(repo interfaces.ProjectRepository) *ProjectService {
	return &ProjectService{repo}
}

// CreateProject 创建新项目
func (s *ProjectService) CreateProject(project *model.Project, goals []model.ProjectGoal) error {
	tx, err := s.repo.BeginTx()
	if err != nil {
		util.Logger.Error("开始事务失败", zap.Error(err))
		return err
	}
	defer tx.Rollback()

	util.Logger.Info("开始创建新项目", zap.String("title", project.Title))

	project.Status = "pending_review"
	project.CreatedAt = time.Now()
	project.UpdatedAt = time.Now()

	err = s.repo.CreateProjectTx(tx, project, goals, project.Images)
	if err != nil {
		util.Logger.Error("创建项目失败", zap.Error(err))
		return err
	}

	if err := tx.Commit(); err != nil {
		util.Logger.Error("提交事务失败", zap.Error(err))
		return err
	}

	util.Logger.Info("项目创建成功", zap.Int("project_id", project.ID))
	return nil
}

// GetProjectByID 通过ID获取项目
func (s *ProjectService) GetProjectByID(id int) (*model.Project, error) {
	util.Logger.Info("开始获取项目", zap.Int("project_id", id))

	project, err := s.repo.GetProjectByID(id)
	if err != nil {
		util.Logger.Error("获取项目失败", zap.Error(err), zap.Int("project_id", id))
		return nil, err
	}

	util.Logger.Info("成功获取项", zap.Int("project_id", id))
	return project, nil
}

// UpdateProject 更新项目信息
func (s *ProjectService) UpdateProject(project *model.Project, goals []model.ProjectGoal) error {
	util.Logger.Info("开始更新项目", zap.Int("project_id", project.ID))

	project.UpdatedAt = time.Now()

	err := s.repo.UpdateProject(project, goals)
	if err != nil {
		util.Logger.Error("更新项目失败", zap.Error(err), zap.Int("project_id", project.ID))
		return err
	}

	util.Logger.Info("项目更新成功", zap.Int("project_id", project.ID))
	return nil
}

// ListProjects 获取项目列表
func (s *ProjectService) ListProjects(page, pageSize int) ([]model.Project, error) {
	return s.repo.ListProjects(page, pageSize)
}

// PledgeToProject 支持项目
func (s *ProjectService) PledgeToProject(userID, projectID int, amount float64) error {
	util.Logger.Info("开始支持项目", zap.Int("user_id", userID), zap.Int("project_id", projectID), zap.Float64("amount", amount))

	pledge := &model.Pledge{
		UserID:    userID,
		ProjectID: projectID,
		Amount:    amount,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	err := s.repo.CreatePledge(pledge)
	if err != nil {
		util.Logger.Error("创建支持记录失败", zap.Error(err))
		return err
	}

	util.Logger.Info("成功支持项目", zap.Int("pledge_id", pledge.ID))
	return nil
}

// ReviewProject 审核项目
func (s *ProjectService) ReviewProject(projectID int, approved bool, comment string) error {
	util.Logger.Info("开始审核项目", zap.Int("project_id", projectID), zap.Bool("approved", approved))

	project, err := s.repo.GetProjectByID(projectID)
	if err != nil {
		util.Logger.Error("获取项目失败", zap.Error(err), zap.Int("project_id", projectID))
		return err
	}

	if project.Status != "pending_review" {
		util.Logger.Warn("项目状态不是待审核", zap.String("current_status", project.Status))
		return errors.New("project is not in pending review status")
	}

	if approved {
		project.Status = "active"
	} else {
		project.Status = "rejected"
	}

	project.UpdatedAt = time.Now()

	err = s.repo.UpdateProjectStatus(project)
	if err != nil {
		util.Logger.Error("更新项目状态失败", zap.Error(err), zap.Int("project_id", projectID))
		return err
	}

	// TODO: 发送通知给项目创建者

	util.Logger.Info("项目审核完成", zap.Int("project_id", projectID), zap.String("new_status", project.Status))
	return nil
}

// GetProjectGoals 获取项目目标
func (s *ProjectService) GetProjectGoals(projectID int) ([]model.ProjectGoal, error) {
	return s.repo.GetProjectGoals(projectID)
}

// GetProjectImages 获取项目图片
func (s *ProjectService) GetProjectImages(projectID int) ([]model.ProjectImage, error) {
	return s.repo.GetProjectImages(projectID)
}

// SearchProjects 搜索项目
func (s *ProjectService) SearchProjects(filters model.ProjectFilters, page, pageSize int) ([]model.Project, int, error) {
	util.Logger.Info("开始搜索项目", zap.Any("filters", filters), zap.Int("page", page), zap.Int("pageSize", pageSize))
	return s.repo.SearchProjects(filters, page, pageSize)
}

// CreateCategory 创建项目分类
func (s *ProjectService) CreateCategory(category *model.ProjectCategory) error {
	util.Logger.Info("开始创建项目分类", zap.String("name", category.Name))
	return s.repo.CreateCategory(category)
}

// GetCategories 获取所有项目分类
func (s *ProjectService) GetCategories() ([]model.ProjectCategory, error) {
	util.Logger.Info("开始获取所有项目分类")
	return s.repo.GetCategories()
}

// CreateTag 创建项目标签
func (s *ProjectService) CreateTag(tag *model.ProjectTag) error {
	util.Logger.Info("开始创建项目标签", zap.String("name", tag.Name))
	return s.repo.CreateTag(tag)
}

// GetTags 获取所有项目标签
func (s *ProjectService) GetTags() ([]model.ProjectTag, error) {
	util.Logger.Info("开始获取所有项目标签")
	return s.repo.GetTags()
}

// AddTagToProject 为项目添加标签
func (s *ProjectService) AddTagToProject(projectID, tagID int) error {
	util.Logger.Info("开始为项目添加标签", zap.Int("project_id", projectID), zap.Int("tag_id", tagID))
	return s.repo.AddTagToProject(projectID, tagID)
}

// CreateProjectUpdate 创建项目更新
func (s *ProjectService) CreateProjectUpdate(update *model.ProjectUpdate) error {
	util.Logger.Info("开始创建项目更新", zap.Int("project_id", update.ProjectID))
	return s.repo.CreateProjectUpdate(update)
}

// GetProjectUpdates 获取项目的所有更新
func (s *ProjectService) GetProjectUpdates(projectID int) ([]model.ProjectUpdate, error) {
	util.Logger.Info("开始获取项目的所有更新", zap.Int("project_id", projectID))
	return s.repo.GetProjectUpdates(projectID)
}

// CreateProjectComment 创建项目评论
func (s *ProjectService) CreateProjectComment(comment *model.ProjectComment) error {
	util.Logger.Info("开始创建项目评论", zap.Int("project_id", comment.ProjectID), zap.Int("user_id", comment.UserID))
	return s.repo.CreateProjectComment(comment)
}

// GetProjectComments 获取项目的评论
func (s *ProjectService) GetProjectComments(projectID int, page, pageSize int) ([]model.ProjectComment, error) {
	util.Logger.Info("开始获取项目的评论", zap.Int("project_id", projectID), zap.Int("page", page), zap.Int("pageSize", pageSize))
	return s.repo.GetProjectComments(projectID, page, pageSize)
}

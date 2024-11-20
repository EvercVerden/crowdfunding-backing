package interfaces

import (
	"crowdfunding-backend/internal/model"
	"database/sql"
)

type ProjectRepository interface {
	BeginTx() (*sql.Tx, error)
	CreateProjectTx(tx *sql.Tx, project *model.Project, goals []model.ProjectGoal, images []model.ProjectImage) error
	GetProjectByID(id int) (*model.Project, error)
	UpdateProject(project *model.Project, goals []model.ProjectGoal) error
	ListProjects(page, pageSize int) ([]model.Project, error)
	CreatePledge(pledge *model.Pledge) error
	UpdateProjectStatus(project *model.Project) error
	GetProjectGoals(projectID int) ([]model.ProjectGoal, error)
	GetProjectImages(projectID int) ([]model.ProjectImage, error)
	SearchProjects(filters model.ProjectFilters, page, pageSize int) ([]model.Project, int, error)
	CreateCategory(category *model.ProjectCategory) error
	GetCategories() ([]model.ProjectCategory, error)
	CreateTag(tag *model.ProjectTag) error
	GetTags() ([]model.ProjectTag, error)
	AddTagToProject(projectID, tagID int) error
	RemoveTagFromProject(projectID, tagID int) error
	GetProjectTags(projectID int) ([]model.ProjectTag, error)
	CreateProjectUpdate(update *model.ProjectUpdate) error
	GetProjectUpdates(projectID int) ([]model.ProjectUpdate, error)
	CreateProjectComment(comment *model.ProjectComment) error
	GetProjectComments(projectID int, page, pageSize int) ([]model.ProjectComment, error)
	CreateShipment(shipment *model.Shipment) error
	UpdateShipment(shipment *model.Shipment) error
	GetShipmentsByProject(projectID int) ([]*model.Shipment, error)
	GetShipmentsByUser(userID int) ([]*model.Shipment, error)
	GetProjectSuccessfulPledgers(projectID int) ([]*model.Pledge, error)
	GetExpiredActiveProjects() ([]*model.Project, error)
	GetProjectsForAdmin(page, pageSize int, status, search string) ([]*model.Project, int, error)
	DeleteProject(projectID int) error
}

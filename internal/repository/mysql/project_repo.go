package mysql

import (
	"crowdfunding-backend/config"
	"crowdfunding-backend/internal/common"
	"crowdfunding-backend/internal/model"
	"crowdfunding-backend/internal/util"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ProjectRepository 实现了项目相关的数据库操作
type ProjectRepository struct {
	db *sql.DB
}

// NewProjectRepository 创建一个新的 ProjectRepository 实例
func NewProjectRepository(db *sql.DB) *ProjectRepository {
	return &ProjectRepository{db}
}

// BeginTx 开始一个新的数据库事务
func (r *ProjectRepository) BeginTx() (*sql.Tx, error) {
	util.Logger.Info("开始数据库事务")
	tx, err := r.db.Begin()
	if err != nil {
		util.Logger.Error("开始事务失败", zap.Error(err))
		return nil, err
	}
	return tx, nil
}

// CreateProjectTx 在事务中创建新项目
func (r *ProjectRepository) CreateProjectTx(tx *sql.Tx, project *model.Project, goals []model.ProjectGoal, images []model.ProjectImage) error {
	util.Logger.Info("开始创建新项目",
		zap.String("title", project.Title),
		zap.Float64("min_reward_amount", project.MinRewardAmount))

	// 插入项目
	result, err := tx.Exec(`
		INSERT INTO projects (
			title, description, creator_id, status, 
			created_at, updated_at, end_date, min_reward_amount
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, project.Title, project.Description, project.CreatorID, project.Status,
		project.CreatedAt, project.UpdatedAt, project.EndDate, project.MinRewardAmount)
	if err != nil {
		util.Logger.Error("插入项目失败", zap.Error(err))
		return err
	}

	projectID, err := result.LastInsertId()
	if err != nil {
		util.Logger.Error("获取项目ID失败", zap.Error(err))
		return err
	}
	project.ID = int(projectID)

	// 插入项目图片
	for _, image := range images {
		_, err = tx.Exec(`
			INSERT INTO project_images (project_id, image_url, is_primary, image_type)
			VALUES (?, ?, ?, ?)
		`, projectID, image.ImageURL, image.IsPrimary, image.ImageType)
		if err != nil {
			util.Logger.Error("插入项目图片失败", zap.Error(err))
			return err
		}
	}

	// 插入目标和目标图片
	for _, goal := range goals {
		result, err := tx.Exec(`
			INSERT INTO project_goals (project_id, amount, description)
			VALUES (?, ?, ?)
		`, projectID, goal.Amount, goal.Description)
		if err != nil {
			return err
		}

		goalID, err := result.LastInsertId()
		if err != nil {
			return err
		}

		for _, image := range goal.Images {
			_, err = tx.Exec(`
				INSERT INTO project_images (project_id, goal_id, image_url, image_type)
				VALUES (?, ?, ?, 'goal')
			`, projectID, goalID, image.ImageURL)
			if err != nil {
				return err
			}
		}
	}

	util.Logger.Info("项目创建成功",
		zap.Int("project_id", int(projectID)),
		zap.Float64("min_reward_amount", project.MinRewardAmount))
	return nil
}

// GetProjectByID 通过ID获取项目
func (r *ProjectRepository) GetProjectByID(id int) (*model.Project, error) {
	util.Logger.Info("开始获取项目", zap.Int("project_id", id))

	var project model.Project
	query := `
		SELECT p.id, p.title, p.description, p.creator_id, p.status, 
			   p.total_amount, p.total_goal_amount, p.progress,
			   p.min_reward_amount,
			   p.created_at, p.updated_at, p.end_date, p.category_id,
			   u.username as creator_username
		FROM projects p
		LEFT JOIN users u ON p.creator_id = u.id
		WHERE p.id = ?`

	var creator model.User
	err := r.db.QueryRow(query, id).Scan(
		&project.ID,
		&project.Title,
		&project.Description,
		&project.CreatorID,
		&project.Status,
		&project.TotalAmount,
		&project.TotalGoalAmount,
		&project.Progress,
		&project.MinRewardAmount,
		&project.CreatedAt,
		&project.UpdatedAt,
		&project.EndDate,
		&project.CategoryID,
		&creator.Username,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			util.Logger.Info("项目不存在", zap.Int("project_id", id))
			return nil, nil
		}
		util.Logger.Error("获取项目失败", zap.Error(err))
		return nil, err
	}

	// 设置创建者信息
	creator.ID = project.CreatorID
	project.Creator = &creator

	util.Logger.Info("成功获取项目",
		zap.Int("project_id", project.ID),
		zap.String("title", project.Title),
		zap.Float64("min_reward_amount", project.MinRewardAmount))

	return &project, nil
}

// UpdateProject 更新项目信息
func (r *ProjectRepository) UpdateProject(project *model.Project, goals []model.ProjectGoal) error {
	util.Logger.Info("开始更新项目", zap.Int("project_id", project.ID))

	tx, err := r.db.Begin()
	if err != nil {
		util.Logger.Error("开始事务失败", zap.Error(err))
		return err
	}
	defer tx.Rollback()

	// 更新项目基本信息
	_, err = tx.Exec(`
		UPDATE projects
		SET title = ?, description = ?, status = ?, total_amount = ?, updated_at = ?, end_date = ?
		WHERE id = ?
	`, project.Title, project.Description, project.Status, project.TotalAmount,
		project.UpdatedAt, project.EndDate, project.ID)
	if err != nil {
		util.Logger.Error("更新项目基本信息失败", zap.Error(err), zap.Int("project_id", project.ID))
		return err
	}

	// 更新目标
	for _, goal := range goals {
		if goal.ID == 0 {
			// 新目标
			_, err = tx.Exec(`
				INSERT INTO project_goals (project_id, amount, description)
				VALUES (?, ?, ?)
			`, project.ID, goal.Amount, goal.Description)
		} else {
			// 更新现有目标
			_, err = tx.Exec(`
				UPDATE project_goals
				SET amount = ?, description = ?
				WHERE id = ? AND project_id = ?
			`, goal.Amount, goal.Description, goal.ID, project.ID)
		}
		if err != nil {
			util.Logger.Error("更新项目目标失败", zap.Error(err), zap.Int("project_id", project.ID), zap.Int("goal_id", goal.ID))
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		util.Logger.Error("提交事务失败", zap.Error(err))
		return err
	}

	util.Logger.Info("项目更新成功", zap.Int("project_id", project.ID))
	return nil
}

// ListProjects 获取项目列表
func (r *ProjectRepository) ListProjects(page, pageSize int) ([]model.Project, error) {
	offset := (page - 1) * pageSize
	query := `
		SELECT p.id, p.title, p.description, p.creator_id, p.status, 
			   p.total_amount, 
			   (SELECT SUM(amount) FROM project_goals WHERE project_id = p.id) as total_goal_amount,
			   p.created_at, p.updated_at, p.end_date, 
			   pi.image_url AS primary_image
		FROM projects p
		LEFT JOIN (
			SELECT project_id, image_url
			FROM project_images
			WHERE is_primary = true
		) pi ON p.id = pi.project_id
		ORDER BY p.created_at DESC
		LIMIT ? OFFSET ?
	`
	rows, err := r.db.Query(query, pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []model.Project
	for rows.Next() {
		var p model.Project
		var primaryImage sql.NullString
		var totalGoalAmount sql.NullFloat64
		err := rows.Scan(
			&p.ID, &p.Title, &p.Description, &p.CreatorID, &p.Status,
			&p.TotalAmount, &totalGoalAmount,
			&p.CreatedAt, &p.UpdatedAt, &p.EndDate, &primaryImage,
		)
		if err != nil {
			return nil, err
		}

		// 设置总目标金额
		if totalGoalAmount.Valid {
			p.TotalGoalAmount = totalGoalAmount.Float64
		}

		// 计算总体进度
		if p.TotalGoalAmount > 0 {
			p.Progress = (p.TotalAmount / p.TotalGoalAmount) * 100
		}

		// 获取项目目标
		goals, err := r.GetProjectGoals(p.ID)
		if err != nil {
			return nil, err
		}

		// 计算每个目标的进度
		for i := range goals {
			if goals[i].Amount > 0 {
				goals[i].Progress = (p.TotalAmount / goals[i].Amount) * 100
				goals[i].IsReached = p.TotalAmount >= goals[i].Amount
			}
		}
		p.Goals = goals

		if primaryImage.Valid {
			p.PrimaryImage = config.AppConfig.BackendURL + "/uploads/" + primaryImage.String
		}
		projects = append(projects, p)
	}

	return projects, nil
}

// CreatePledge 创建新的支持记录
func (r *ProjectRepository) CreatePledge(pledge *model.Pledge) error {
	util.Logger.Info("开始创建支持记录",
		zap.Int("user_id", pledge.UserID),
		zap.Int("project_id", pledge.ProjectID),
		zap.Float64("amount", pledge.Amount))

	_, err := r.db.Exec(`
		INSERT INTO pledges (user_id, project_id, amount, status, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, pledge.UserID, pledge.ProjectID, pledge.Amount, pledge.Status, pledge.CreatedAt)
	if err != nil {
		util.Logger.Error("创建支持记录失败", zap.Error(err))
		return err
	}

	util.Logger.Info("支持记录创建成功")
	return nil
}

// UpdateProjectStatus 更新项目状态
func (r *ProjectRepository) UpdateProjectStatus(project *model.Project) error {
	util.Logger.Info("开始更新项目状态",
		zap.Int("project_id", project.ID),
		zap.String("new_status", project.Status))

	// 直接更新项目状态
	query := `
		UPDATE projects 
		SET status = ?, updated_at = NOW() 
		WHERE id = ?`

	result, err := r.db.Exec(query, project.Status, project.ID)
	if err != nil {
		util.Logger.Error("更新项目状态失败", zap.Error(err))
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		util.Logger.Error("获取影响行数失败", zap.Error(err))
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("project not found or status not changed")
	}

	util.Logger.Info("项目状态更新成功",
		zap.Int("project_id", project.ID),
		zap.String("status", project.Status))

	return nil
}

// GetProjectGoals 获取项目目标
func (r *ProjectRepository) GetProjectGoals(projectID int) ([]model.ProjectGoal, error) {
	query := `
		SELECT id, project_id, amount, description, is_reached, progress
		FROM project_goals
		WHERE project_id = ?
		ORDER BY amount ASC
	`
	rows, err := r.db.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []model.ProjectGoal
	for rows.Next() {
		var goal model.ProjectGoal
		err := rows.Scan(
			&goal.ID, &goal.ProjectID, &goal.Amount, &goal.Description,
			&goal.IsReached, &goal.Progress,
		)
		if err != nil {
			return nil, err
		}

		// 获取目标的图片
		images, err := r.getGoalImages(goal.ID)
		if err != nil {
			return nil, err
		}
		goal.Images = images
		goals = append(goals, goal)
	}

	return goals, nil
}

// getGoalImages 获取目标的图片
func (r *ProjectRepository) getGoalImages(goalID int) ([]model.ProjectImage, error) {
	query := `
		SELECT id, project_id, goal_id, image_url, is_primary
		FROM project_images
		WHERE goal_id = ?
	`
	rows, err := r.db.Query(query, goalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []model.ProjectImage
	for rows.Next() {
		var img model.ProjectImage
		err := rows.Scan(
			&img.ID, &img.ProjectID, &img.GoalID, &img.ImageURL, &img.IsPrimary,
		)
		if err != nil {
			return nil, err
		}
		// 添加完整的URL
		img.ImageURL = config.AppConfig.BackendURL + "/uploads/" + img.ImageURL
		images = append(images, img)
	}

	return images, nil
}

// GetProjectImages 获取项目图片
func (r *ProjectRepository) GetProjectImages(projectID int) ([]model.ProjectImage, error) {
	query := `
		SELECT id, project_id, goal_id, image_url, is_primary, image_type
		FROM project_images
		WHERE project_id = ?
		ORDER BY is_primary DESC, id ASC
	`
	rows, err := r.db.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []model.ProjectImage
	for rows.Next() {
		var img model.ProjectImage
		err := rows.Scan(
			&img.ID, &img.ProjectID, &img.GoalID, &img.ImageURL,
			&img.IsPrimary, &img.ImageType,
		)
		if err != nil {
			return nil, err
		}
		// 添加完整的URL
		img.ImageURL = config.AppConfig.BackendURL + "/uploads/" + img.ImageURL
		images = append(images, img)
	}

	return images, nil
}

// SearchProjects 搜索项目
func (r *ProjectRepository) SearchProjects(filters model.ProjectFilters, page, pageSize int) ([]model.Project, int, error) {
	util.Logger.Info("开始搜索目", zap.Any("filters", filters), zap.Int("page", page), zap.Int("pageSize", pageSize))

	query := `SELECT p.id, p.title, p.description, p.creator_id, p.status, p.total_amount, p.created_at, p.updated_at, p.end_date
			  FROM projects p
			  LEFT JOIN project_tag_relations ptr ON p.id = ptr.project_id
			  WHERE 1=1`

	countQuery := `SELECT COUNT(DISTINCT p.id) FROM projects p
				   LEFT JOIN project_tag_relations ptr ON p.id = ptr.project_id
				   WHERE 1=1`

	var args []interface{}
	var conditions []string

	if filters.Keyword != "" {
		conditions = append(conditions, "(p.title LIKE ? OR p.description LIKE ?)")
		args = append(args, "%"+filters.Keyword+"%", "%"+filters.Keyword+"%")
	}

	if filters.Category != 0 {
		conditions = append(conditions, "p.category_id = ?")
		args = append(args, filters.Category)
	}

	if filters.Status != "" {
		conditions = append(conditions, "p.status = ?")
		args = append(args, filters.Status)
	}

	if filters.MinAmount > 0 {
		conditions = append(conditions, "p.total_amount >= ?")
		args = append(args, filters.MinAmount)
	}

	if filters.MaxAmount > 0 {
		conditions = append(conditions, "p.total_amount <= ?")
		args = append(args, filters.MaxAmount)
	}

	if !filters.StartDate.IsZero() {
		conditions = append(conditions, "p.created_at >= ?")
		args = append(args, filters.StartDate)
	}

	if !filters.EndDate.IsZero() {
		conditions = append(conditions, "p.end_date <= ?")
		args = append(args, filters.EndDate)
	}

	if len(filters.Tags) > 0 {
		placeholders := make([]string, len(filters.Tags))
		for i := range filters.Tags {
			placeholders[i] = "?"
			args = append(args, filters.Tags[i])
		}
		conditions = append(conditions, "ptr.tag_id IN ("+strings.Join(placeholders, ",")+")")
	}

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
		countQuery += " AND " + strings.Join(conditions, " AND ")
	}

	query += " GROUP BY p.id ORDER BY p.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, (page-1)*pageSize)

	// 执行计数查询
	var totalCount int
	err := r.db.QueryRow(countQuery, args[:len(args)-2]...).Scan(&totalCount)
	if err != nil {
		util.Logger.Error("计算项目总数失败", zap.Error(err))
		return nil, 0, err
	}

	// 执行主查询
	rows, err := r.db.Query(query, args...)
	if err != nil {
		util.Logger.Error("执行高级搜索查询失败", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	var projects []model.Project
	for rows.Next() {
		var p model.Project
		err := rows.Scan(
			&p.ID, &p.Title, &p.Description, &p.CreatorID, &p.Status,
			&p.TotalAmount, &p.CreatedAt, &p.UpdatedAt, &p.EndDate,
		)
		if err != nil {
			util.Logger.Error("扫描项目数据失败", zap.Error(err))
			return nil, 0, err
		}
		projects = append(projects, p)
	}

	util.Logger.Info("高级搜索项目成功", zap.Int("count", len(projects)), zap.Int("total", totalCount))
	return projects, totalCount, nil
}

// CreateCategory 创建项目分类
func (r *ProjectRepository) CreateCategory(category *model.ProjectCategory) error {
	util.Logger.Info("开始创建项目分类", zap.String("name", category.Name))

	query := `INSERT INTO project_categories (name) VALUES (?)`
	result, err := r.db.Exec(query, category.Name)
	if err != nil {
		util.Logger.Error("创建项目分类失败", zap.Error(err))
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		util.Logger.Error("获取新分类ID失败", zap.Error(err))
		return err
	}
	category.ID = int(id)

	util.Logger.Info("项目分类创建成功", zap.Int("category_id", category.ID))
	return nil
}

// AddTagToProject 为项目添加标签
func (r *ProjectRepository) AddTagToProject(projectID, tagID int) error {
	util.Logger.Info("开始为目添加标签", zap.Int("project_id", projectID), zap.Int("tag_id", tagID))

	query := `INSERT INTO project_tag_relations (project_id, tag_id) VALUES (?, ?)`
	_, err := r.db.Exec(query, projectID, tagID)
	if err != nil {
		util.Logger.Error("为项目添加签失败", zap.Error(err))
		return err
	}

	util.Logger.Info("成功为项目添加标签", zap.Int("project_id", projectID), zap.Int("tag_id", tagID))
	return nil
}

// GetCategories 获取所有项目分类
func (r *ProjectRepository) GetCategories() ([]model.ProjectCategory, error) {
	util.Logger.Info("开始获取所有项目分类")

	query := `SELECT id, name, created_at FROM project_categories`
	rows, err := r.db.Query(query)
	if err != nil {
		util.Logger.Error("获取项目分类失败", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var categories []model.ProjectCategory
	for rows.Next() {
		var c model.ProjectCategory
		err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt)
		if err != nil {
			util.Logger.Error("扫描项目分类数据失败", zap.Error(err))
			return nil, err
		}
		categories = append(categories, c)
	}

	util.Logger.Info("成功获取所有项目分类", zap.Int("count", len(categories)))
	return categories, nil
}

// CreateTag 创建项标签
func (r *ProjectRepository) CreateTag(tag *model.ProjectTag) error {
	util.Logger.Info("开始创建项目标签", zap.String("name", tag.Name))

	query := `INSERT INTO project_tags (name) VALUES (?)`
	result, err := r.db.Exec(query, tag.Name)
	if err != nil {
		util.Logger.Error("创建项目标签失败", zap.Error(err))
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		util.Logger.Error("获取新标签ID败", zap.Error(err))
		return err
	}
	tag.ID = int(id)

	util.Logger.Info("目标签创建成功", zap.Int("tag_id", tag.ID))
	return nil
}

// GetTags 获取所有项目标
func (r *ProjectRepository) GetTags() ([]model.ProjectTag, error) {
	util.Logger.Info("开始获取所有项目标签")

	query := `SELECT id, name, created_at FROM project_tags`
	rows, err := r.db.Query(query)
	if err != nil {
		util.Logger.Error("获取项目标签失败", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var tags []model.ProjectTag
	for rows.Next() {
		var t model.ProjectTag
		err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt)
		if err != nil {
			util.Logger.Error("扫描项目标签数据失败", zap.Error(err))
			return nil, err
		}
		tags = append(tags, t)
	}

	util.Logger.Info("成功获取所有项目标签", zap.Int("count", len(tags)))
	return tags, nil
}

// RemoveTagFromProject 从项目中移除标签
func (r *ProjectRepository) RemoveTagFromProject(projectID, tagID int) error {
	util.Logger.Info("开始从项目中移除标签", zap.Int("project_id", projectID), zap.Int("tag_id", tagID))

	query := `DELETE FROM project_tag_relations WHERE project_id = ? AND tag_id = ?`
	_, err := r.db.Exec(query, projectID, tagID)
	if err != nil {
		util.Logger.Error("从项目中移除标签失", zap.Error(err))
		return err
	}

	util.Logger.Info("成功从项目中移除标签", zap.Int("project_id", projectID), zap.Int("tag_id", tagID))
	return nil
}

// GetProjectTags 获取项目的所有标签
func (r *ProjectRepository) GetProjectTags(projectID int) ([]model.ProjectTag, error) {
	util.Logger.Info("开始获取项目的所有标签", zap.Int("project_id", projectID))

	query := `
		SELECT pt.id, pt.name, pt.created_at
		FROM project_tags pt
		JOIN project_tag_relations ptr ON pt.id = ptr.tag_id
		WHERE ptr.project_id = ?
	`
	rows, err := r.db.Query(query, projectID)
	if err != nil {
		util.Logger.Error("获取项标签失败", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var tags []model.ProjectTag
	for rows.Next() {
		var t model.ProjectTag
		err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt)
		if err != nil {
			util.Logger.Error("扫描项目标签数据失败", zap.Error(err))
			return nil, err
		}
		tags = append(tags, t)
	}

	util.Logger.Info("成功获取项目的所有标签", zap.Int("project_id", projectID), zap.Int("count", len(tags)))
	return tags, nil
}

// CreateProjectUpdate 创建项目更新
func (r *ProjectRepository) CreateProjectUpdate(update *model.ProjectUpdate) error {
	util.Logger.Info("开始创建项目新", zap.Int("project_id", update.ProjectID))

	query := `INSERT INTO project_updates (project_id, title, content) VALUES (?, ?, ?)`
	result, err := r.db.Exec(query, update.ProjectID, update.Title, update.Content)
	if err != nil {
		util.Logger.Error("创建项目更新失败", zap.Error(err))
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		util.Logger.Error("获取新项目更新ID失败", zap.Error(err))
		return err
	}
	update.ID = int(id)

	util.Logger.Info("项目更新创建成功", zap.Int("update_id", update.ID))
	return nil
}

// GetProjectUpdates 获取项目的所有更新
func (r *ProjectRepository) GetProjectUpdates(projectID int) ([]model.ProjectUpdate, error) {
	util.Logger.Info("开始获取项目的所有更新", zap.Int("project_id", projectID))

	query := `SELECT id, project_id, title, content, created_at FROM project_updates WHERE project_id = ? ORDER BY created_at DESC`
	rows, err := r.db.Query(query, projectID)
	if err != nil {
		util.Logger.Error("获取项目更新失败", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var updates []model.ProjectUpdate
	for rows.Next() {
		var u model.ProjectUpdate
		err := rows.Scan(&u.ID, &u.ProjectID, &u.Title, &u.Content, &u.CreatedAt)
		if err != nil {
			util.Logger.Error("扫描项目更新数据失败", zap.Error(err))
			return nil, err
		}
		updates = append(updates, u)
	}

	util.Logger.Info("成功获取项目的所有更新", zap.Int("project_id", projectID), zap.Int("count", len(updates)))
	return updates, nil
}

// CreateProjectComment 创建项目评论
func (r *ProjectRepository) CreateProjectComment(comment *model.ProjectComment) error {
	util.Logger.Info("开始创建项目评论", zap.Int("project_id", comment.ProjectID), zap.Int("user_id", comment.UserID))

	query := `INSERT INTO project_comments (project_id, user_id, content) VALUES (?, ?, ?)`
	result, err := r.db.Exec(query, comment.ProjectID, comment.UserID, comment.Content)
	if err != nil {
		util.Logger.Error("创建项目评论失败", zap.Error(err))
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		util.Logger.Error("获取项目评论ID失败", zap.Error(err))
		return err
	}
	comment.ID = int(id)

	util.Logger.Info("项目评论创建成功", zap.Int("comment_id", comment.ID))
	return nil
}

// GetProjectComments 获取项目的评论
func (r *ProjectRepository) GetProjectComments(projectID int, page, pageSize int) ([]model.ProjectComment, error) {
	util.Logger.Info("开始获取项的评论", zap.Int("project_id", projectID), zap.Int("page", page), zap.Int("pageSize", pageSize))

	offset := (page - 1) * pageSize
	query := `
		SELECT id, project_id, user_id, content, created_at, updated_at
		FROM project_comments
		WHERE project_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	rows, err := r.db.Query(query, projectID, pageSize, offset)
	if err != nil {
		util.Logger.Error("获取项目评论失败", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var comments []model.ProjectComment
	for rows.Next() {
		var c model.ProjectComment
		err := rows.Scan(&c.ID, &c.ProjectID, &c.UserID, &c.Content, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			util.Logger.Error("扫描项目评论数据失败", zap.Error(err))
			return nil, err
		}
		comments = append(comments, c)
	}

	util.Logger.Info("成功获取项目的评论", zap.Int("project_id", projectID), zap.Int("count", len(comments)))
	return comments, nil
}

// CreateShipment 创建发货记录
func (r *ProjectRepository) CreateShipment(shipment *model.Shipment) error {
	query := `INSERT INTO shipments (project_id, user_id, address_id, status, created_at) 
			  VALUES (?, ?, ?, ?, NOW())`
	result, err := r.db.Exec(query,
		shipment.ProjectID, shipment.UserID, shipment.AddressID, shipment.Status)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	shipment.ID = int(id)
	return nil
}

// UpdateShipment 更新发货记录
func (r *ProjectRepository) UpdateShipment(shipment *model.Shipment) error {
	query := `
		UPDATE shipments 
		SET status = ?, 
			tracking_number = ?, 
			ipping_company = ?, 
			shipped_at = CASE WHEN status = 'shipped' THEN NOW() ELSE shipped_at END,
				estimated_delivery_at = ?,
			updated_at = NOW()
		WHERE id = ?`

	_, err := r.db.Exec(query,
		shipment.Status,
		shipment.TrackingNumber,
		shipment.ShippingCompany,
		shipment.EstimatedDeliveryAt,
		shipment.ID)

	return err
}

// GetShipmentsByProject 获取项目的所有发货记录
func (r *ProjectRepository) GetShipmentsByProject(projectID int) ([]*model.Shipment, error) {
	query := `
		SELECT s.*, a.*
		FROM shipments s
		LEFT JOIN user_addresses a ON s.address_id = a.id
		WHERE s.project_id = ?
		ORDER BY s.created_at DESC`

	rows, err := r.db.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shipments []*model.Shipment
	for rows.Next() {
		var s model.Shipment
		var a model.UserAddress
		err := rows.Scan(
			&s.ID, &s.ProjectID, &s.UserID, &s.OrderID, &s.AddressID,
			&s.Status, &s.TrackingNumber, &s.ShippingCompany,
			&s.ShippedAt, &s.DeliveredAt, &s.EstimatedDeliveryAt,
			&s.CreatedAt, &s.UpdatedAt,
			&a.ID, &a.UserID, &a.ReceiverName, &a.Phone,
			&a.Province, &a.City, &a.District, &a.DetailAddress,
			&a.IsDefault, &a.CreatedAt, &a.UpdatedAt)
		if err != nil {
			return nil, err
		}
		s.Address = &a
		shipments = append(shipments, &s)
	}
	return shipments, nil
}

// GetShipmentsByUser 获取用户的所有发货记录
func (r *ProjectRepository) GetShipmentsByUser(userID int) ([]*model.Shipment, error) {
	query := `
		SELECT s.*, a.*
		FROM shipments s
		LEFT JOIN user_addresses a ON s.address_id = a.id
		WHERE s.user_id = ?
		ORDER BY s.created_at DESC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shipments []*model.Shipment
	for rows.Next() {
		var s model.Shipment
		var a model.UserAddress
		err := rows.Scan(
			&s.ID, &s.ProjectID, &s.UserID, &s.OrderID, &s.AddressID,
			&s.Status, &s.TrackingNumber, &s.ShippingCompany,
			&s.ShippedAt, &s.DeliveredAt, &s.EstimatedDeliveryAt,
			&s.CreatedAt, &s.UpdatedAt,
			&a.ID, &a.UserID, &a.ReceiverName, &a.Phone,
			&a.Province, &a.City, &a.District, &a.DetailAddress,
			&a.IsDefault, &a.CreatedAt, &a.UpdatedAt)
		if err != nil {
			return nil, err
		}
		s.Address = &a
		shipments = append(shipments, &s)
	}
	return shipments, nil
}

// GetProjectSuccessfulPledgers 获取项目的成功支持者
func (r *ProjectRepository) GetProjectSuccessfulPledgers(projectID int) ([]*model.Pledge, error) {
	query := `
		SELECT p.*, a.*
		FROM pledges p
		LEFT JOIN user_addresses a ON p.address_id = a.id
		WHERE p.project_id = ? AND p.status = 'completed'
		ORDER BY p.created_at DESC`

	rows, err := r.db.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pledges []*model.Pledge
	for rows.Next() {
		var p model.Pledge
		var a model.UserAddress
		err := rows.Scan(
			&p.ID, &p.UserID, &p.ProjectID, &p.Amount, &p.Status,
			&p.AddressID, &p.CreatedAt,
			&a.ID, &a.UserID, &a.ReceiverName, &a.Phone,
			&a.Province, &a.City, &a.District, &a.DetailAddress,
			&a.IsDefault, &a.CreatedAt, &a.UpdatedAt)
		if err != nil {
			return nil, err
		}
		p.Address = &a
		pledges = append(pledges, &p)
	}
	return pledges, nil
}

// GetExpiredActiveProjects 获取所有已过期但状态未更新的项目
func (r *ProjectRepository) GetExpiredActiveProjects() ([]*model.Project, error) {
	util.Logger.Info("开始查询过期项目")

	// 主查询
	query := `
		SELECT p.id, p.title, p.description, p.creator_id, p.status, 
			   p.total_amount, p.total_goal_amount, p.progress,
			   p.min_reward_amount, p.created_at, p.updated_at, p.end_date, p.category_id,
			   COALESCE(SUM(o.amount), 0) as total_pledged,
			   (
				   SELECT MIN(pg.amount)
				   FROM project_goals pg
				   WHERE pg.project_id = p.id
			   ) as first_goal_amount
		FROM projects p
		LEFT JOIN orders o ON p.id = o.project_id 
			AND o.status = 'paid'
		WHERE p.status NOT IN ('failed', 'success', 'rejected')
			AND p.end_date <= NOW()
		GROUP BY p.id, p.title, p.description, p.creator_id, p.status,
				 p.total_amount, p.total_goal_amount, p.progress,
				 p.min_reward_amount, p.created_at, p.updated_at, p.end_date, p.category_id
		HAVING (
			first_goal_amount IS NOT NULL 
				AND total_pledged < first_goal_amount
		)`

	util.Logger.Debug("执行主查询SQL",
		zap.String("query", query),
		zap.String("current_time", time.Now().Format("2006-01-02 15:04:05")))

	rows, err := r.db.Query(query)
	if err != nil {
		util.Logger.Error("查询过期项目失败", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var projects []*model.Project
	for rows.Next() {
		var project model.Project
		var totalPledged, firstGoalAmount sql.NullFloat64
		err := rows.Scan(
			&project.ID,
			&project.Title,
			&project.Description,
			&project.CreatorID,
			&project.Status,
			&project.TotalAmount,
			&project.TotalGoalAmount,
			&project.Progress,
			&project.MinRewardAmount,
			&project.CreatedAt,
			&project.UpdatedAt,
			&project.EndDate,
			&project.CategoryID,
			&totalPledged,
			&firstGoalAmount,
		)
		if err != nil {
			util.Logger.Error("扫描项目数据失败", zap.Error(err))
			return nil, err
		}

		project.Status = "failed" // 设置状态为失败
		util.Logger.Info("找到过期项目",
			zap.Int("project_id", project.ID),
			zap.String("title", project.Title),
			zap.Float64("total_pledged", totalPledged.Float64),
			zap.Float64("first_goal_amount", firstGoalAmount.Float64),
			zap.Time("end_date", project.EndDate),
			zap.Time("current_time", time.Now()),
			zap.String("end_date_str", project.EndDate.Format("2006-01-02 15:04:05")))
		projects = append(projects, &project)
	}

	util.Logger.Info("过期项目查询完成",
		zap.Int("count", len(projects)),
		zap.Time("current_time", time.Now()))
	return projects, nil
}

// GetProjectsForAdmin 获取项目列表（管理员视图）
func (r *ProjectRepository) GetProjectsForAdmin(page, pageSize int, status, search string) ([]*model.Project, int, error) {
	// 构建基础查询
	baseQuery := `
		SELECT p.id, p.title, p.description, p.creator_id, p.status,
			   p.total_amount, 
			   (SELECT amount 
			    FROM project_goals 
			    WHERE project_id = p.id 
			    ORDER BY amount DESC 
			    LIMIT 1) as total_goal_amount,
			   p.progress,
			   p.created_at, p.updated_at,
			   u.username as creator_username, u.email as creator_email
		FROM projects p
		LEFT JOIN users u ON p.creator_id = u.id
		WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM projects p WHERE 1=1`

	var params []interface{}
	var conditions []string

	// 添加搜索条件
	if search != "" {
		searchCond := ` AND (p.title LIKE ? OR p.description LIKE ?)`
		searchParam := "%" + search + "%"
		conditions = append(conditions, searchCond)
		params = append(params, searchParam, searchParam)
	}

	// 添加状态过滤
	if status != "" {
		statusCond := ` AND p.status = ?`
		conditions = append(conditions, statusCond)
		params = append(params, status)
	}

	// 拼接条件
	if len(conditions) > 0 {
		baseQuery += strings.Join(conditions, "")
		countQuery += strings.Join(conditions, "")
	}

	// 获取总数
	var total int
	err := r.db.QueryRow(countQuery, params...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 添加分页和排序
	baseQuery += ` ORDER BY p.created_at DESC LIMIT ? OFFSET ?`
	params = append(params, pageSize, (page-1)*pageSize)

	// 执行查询
	rows, err := r.db.Query(baseQuery, params...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var projects []*model.Project
	for rows.Next() {
		var p model.Project
		var creator model.User
		err := rows.Scan(
			&p.ID, &p.Title, &p.Description, &p.CreatorID,
			&p.Status, &p.TotalAmount, &p.TotalGoalAmount,
			&p.Progress, &p.CreatedAt, &p.UpdatedAt,
			&creator.Username, &creator.Email,
		)
		if err != nil {
			return nil, 0, err
		}

		// 设置创建者信息
		creator.ID = p.CreatorID
		p.Creator = &creator

		projects = append(projects, &p)
	}

	return projects, total, nil
}

// DeleteProject 删除项目及其相关数据
func (r *ProjectRepository) DeleteProject(projectID int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 删除项目相关的所有数据
	tables := []string{
		"project_images",
		"project_goals",
		"project_tag_relations",
		"project_updates",
		"project_comments",
	}

	for _, table := range tables {
		_, err = tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE project_id = ?", table), projectID)
		if err != nil {
			return err
		}
	}

	// 最后删除项目本身
	_, err = tx.Exec("DELETE FROM projects WHERE id = ?", projectID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// 使用 common 包中的重试机制
func (r *ProjectRepository) CreateProjectWithRetry(project *model.Project, goals []model.ProjectGoal, images []model.ProjectImage) error {
	return common.WithRetry(func() error {
		tx, err := r.db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		err = r.CreateProjectTx(tx, project, goals, images)
		if err != nil {
			return err
		}

		return tx.Commit()
	}, 3)
}

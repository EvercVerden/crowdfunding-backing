package project

import (
	"crowdfunding-backend/internal/model"
	"crowdfunding-backend/internal/service"
	"crowdfunding-backend/internal/util"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"crowdfunding-backend/internal/storage"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ProjectHandler 处理与项目相关的HTTP请求
type ProjectHandler struct {
	projectService *service.ProjectService
	storage        *storage.LocalStorage
}

// NewProjectHandler 创建一个新的 ProjectHandler 实例
func NewProjectHandler(projectService *service.ProjectService, storage *storage.LocalStorage) *ProjectHandler {
	return &ProjectHandler{projectService, storage}
}

// CreateProject 处理创建新项目的请求
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	util.Logger.Info("开始处理创建项目请求")

	// 解析多部分表单
	err := c.Request.ParseMultipartForm(32 << 20) // 32 MB
	if err != nil {
		util.Logger.Error("解析多部分表单失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法解析表单数据"})
		return
	}

	// 获取并记录所有表单字段
	util.Logger.Info("表单数据",
		zap.Any("form_values", c.Request.Form),
		zap.Any("file_headers", c.Request.MultipartForm.File))

	// 获取项目基本信息
	title := c.PostForm("title")
	description := c.PostForm("description")
	endDateStr := c.PostForm("end_date")
	categoryIDStr := c.PostForm("category_id")
	minRewardAmountStr := c.PostForm("min_reward_amount")

	util.Logger.Info("获取到的基本信息",
		zap.String("title", title),
		zap.String("description", description),
		zap.String("end_date", endDateStr),
		zap.String("category_id", categoryIDStr),
		zap.String("min_reward_amount_str", minRewardAmountStr))

	// 验证必填字段
	if title == "" || description == "" || endDateStr == "" || minRewardAmountStr == "" {
		util.Logger.Error("缺少必要字段",
			zap.String("title", title),
			zap.String("description", description),
			zap.String("end_date", endDateStr),
			zap.String("min_reward_amount", minRewardAmountStr))
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少必要字段",
			"details": map[string]bool{
				"title":             title == "",
				"description":       description == "",
				"end_date":          endDateStr == "",
				"min_reward_amount": minRewardAmountStr == "",
			},
		})
		return
	}

	// 解析最低有奖支持金额
	minRewardAmount, err := strconv.ParseFloat(minRewardAmountStr, 64)
	if err != nil {
		util.Logger.Error("解析最低有奖支持金额失败",
			zap.Error(err),
			zap.String("min_reward_amount_str", minRewardAmountStr))
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的最低有奖支持金额",
			"details": err.Error(),
		})
		return
	}

	// 验证最低有奖支持金额
	if minRewardAmount <= 0 {
		util.Logger.Error("最低有奖支持金额必须大于0",
			zap.Float64("min_reward_amount", minRewardAmount))
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "最低有奖支持金额必须大于0",
		})
		return
	}

	// 解析结束日期
	endDate, err := time.Parse(time.RFC3339, endDateStr)
	if err != nil {
		util.Logger.Warn("无效的结束日期", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束日期格式"})
		return
	}

	// 解析分类ID
	categoryID, err := strconv.Atoi(categoryIDStr)
	if err != nil {
		util.Logger.Warn("无效的分类ID", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的分类ID"})
		return
	}

	userID, _ := c.Get("user_id")
	project := &model.Project{
		Title:           title,
		Description:     description,
		CreatorID:       userID.(int),
		EndDate:         endDate,
		CategoryID:      &categoryID,
		MinRewardAmount: minRewardAmount,
	}

	util.Logger.Info("准备创建项目",
		zap.String("title", project.Title),
		zap.Float64("min_reward_amount", project.MinRewardAmount),
		zap.Time("end_date", project.EndDate))

	// 处理项目主图和其他图片
	form, _ := c.MultipartForm()
	projectFiles := form.File["project_images[]"]
	var projectImages []model.ProjectImage
	for i, file := range projectFiles {
		filename := util.GenerateUniqueFilename(file.Filename)
		path := fmt.Sprintf("projects/%d/%s", project.ID, filename)
		imageURL, err := h.storage.UploadFile(file, path)
		if err != nil {
			util.Logger.Error("上传项目图片失败", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存项目图片失败"})
			return
		}
		projectImages = append(projectImages, model.ProjectImage{
			ImageURL:  imageURL,
			IsPrimary: i == 0,
			ImageType: "main",
		})
	}

	// 处理项目详情长图
	longFiles := form.File["long_images[]"]
	for _, file := range longFiles {
		filename := util.GenerateUniqueFilename(file.Filename)
		path := fmt.Sprintf("projects/%d/long/%s", project.ID, filename)
		imageURL, err := h.storage.UploadFile(file, path)
		if err != nil {
			util.Logger.Error("上传长图失败", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存长图失败"})
			return
		}
		projectImages = append(projectImages, model.ProjectImage{
			ImageURL:  imageURL,
			ImageType: "long",
		})
	}

	project.Images = projectImages

	// 解析目标
	var goals []model.ProjectGoal
	for i := 0; ; i++ {
		amountStr := c.PostForm(fmt.Sprintf("goals[%d][amount]", i))
		if amountStr == "" {
			break
		}

		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			util.Logger.Warn("无效的目标金额", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的目标金额"})
			return
		}

		description := c.PostForm(fmt.Sprintf("goals[%d][description]", i))

		// 处理目标图片
		goalFiles := form.File[fmt.Sprintf("goals[%d][images][]", i)]
		var goalImages []model.ProjectImage
		for _, file := range goalFiles {
			filename := util.GenerateUniqueFilename(file.Filename)
			path := fmt.Sprintf("projects/%d/goals/%d/%s", project.ID, i, filename)
			imageURL, err := h.storage.UploadFile(file, path)
			if err != nil {
				util.Logger.Error("保存目标奖励图片失败", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "保存目标奖励图片失败"})
				return
			}
			goalImages = append(goalImages, model.ProjectImage{
				ImageURL:  imageURL,
				ImageType: "goal",
			})
		}

		goals = append(goals, model.ProjectGoal{
			Amount:      amount,
			Description: description,
			Images:      goalImages,
		})
	}

	// 处理项目创建
	if err := h.projectService.CreateProject(project, goals); err != nil {
		util.Logger.Error("创建项目失败",
			zap.Error(err),
			zap.Any("project", project))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建项目失败",
			"details": err.Error(),
		})
		return
	}

	util.Logger.Info("项目创建成功",
		zap.Int("project_id", project.ID),
		zap.Float64("min_reward_amount", project.MinRewardAmount))

	c.JSON(http.StatusCreated, gin.H{
		"code": 201,
		"data": gin.H{
			"project_id":        project.ID,
			"title":             project.Title,
			"min_reward_amount": project.MinRewardAmount,
			"created_at":        project.CreatedAt,
		},
		"message": "Project created successfully",
	})
}

// GetProject 处理获取单个项目详情的请求
func (h *ProjectHandler) GetProject(c *gin.Context) {
	util.Logger.Info("开始处理获取项目详情请求")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		util.Logger.Warn("无效的项目ID", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	project, err := h.projectService.GetProjectByID(id)
	if err != nil {
		util.Logger.Error("获取项目失败", zap.Error(err), zap.Int("project_id", id))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get project"})
		return
	}

	// 获取项目的所有图片
	images, err := h.projectService.GetProjectImages(id)
	if err != nil {
		util.Logger.Error("获取项目图片失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get project images"})
		return
	}

	// 分类处理图片
	var mainImages []model.ProjectImage
	var longImages []string
	for _, img := range images {
		switch img.ImageType {
		case "main":
			mainImages = append(mainImages, img)
		case "long":
			longImages = append(longImages, img.ImageURL)
		}
	}

	// 设置主图和图片列表
	if len(mainImages) > 0 {
		project.PrimaryImage = mainImages[0].ImageURL
		project.Images = mainImages
	}
	project.LongImages = longImages

	// 获取项目目标及其图片
	goals, err := h.projectService.GetProjectGoals(id)
	if err != nil {
		util.Logger.Error("获取项目目标失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get project goals"})
		return
	}
	project.Goals = goals

	util.Logger.Info("成功获取项目详情", zap.Int("project_id", id))
	c.JSON(http.StatusOK, gin.H{
		"project": project,
	})
}

// UpdateProject 处理更新项目信息的请求
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	util.Logger.Info("开始处理更新项目请求")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		util.Logger.Warn("无效的项目ID", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	var input struct {
		Title       string              `json:"title"`
		Description string              `json:"description"`
		EndDate     time.Time           `json:"end_date"`
		Goals       []model.ProjectGoal `json:"goals"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		util.Logger.Warn("更新项目请求数据无效", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	project := &model.Project{
		ID:          id,
		Title:       input.Title,
		Description: input.Description,
		EndDate:     input.EndDate,
	}

	if err := h.projectService.UpdateProject(project, input.Goals); err != nil {
		util.Logger.Error("更新项目失败", zap.Error(err), zap.Int("project_id", id))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project"})
		return
	}

	util.Logger.Info("项目更新成功", zap.Int("project_id", id))
	c.JSON(http.StatusOK, gin.H{"message": "Project updated successfully"})
}

// ListProjects 处理获取项目列表的请求
func (h *ProjectHandler) ListProjects(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	projects, err := h.projectService.ListProjects(page, pageSize)
	if err != nil {
		util.Logger.Error("获取项目列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get projects"})
		return
	}

	// 如果需要，这里可以添加额外的处理逻辑

	c.JSON(http.StatusOK, projects)
}

// PledgeToProject 处理用户支持项目的请求
func (h *ProjectHandler) PledgeToProject(c *gin.Context) {
	util.Logger.Info("开始处理支持项目请求")
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		util.Logger.Warn("无效的项目ID", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	var input struct {
		Amount float64 `json:"amount" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		util.Logger.Warn("持项目请求数据无效", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	if err := h.projectService.PledgeToProject(userID.(int), projectID, input.Amount); err != nil {
		util.Logger.Error("支持项目失败", zap.Error(err), zap.Int("project_id", projectID), zap.Float64("amount", input.Amount))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to pledge to project"})
		return
	}

	util.Logger.Info("成功支持项目", zap.Int("project_id", projectID), zap.Float64("amount", input.Amount))
	c.JSON(http.StatusOK, gin.H{"message": "Successfully pledged to project"})
}

// ReviewProject 处理项目审核请
func (h *ProjectHandler) ReviewProject(c *gin.Context) {
	util.Logger.Info("开始处理项目审核请求")
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		util.Logger.Warn("无效的项目ID", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	var input struct {
		Approved bool   `json:"approved" binding:"required"`
		Comment  string `json:"comment"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		util.Logger.Warn("项目审核请求数据无效", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.projectService.ReviewProject(projectID, input.Approved, input.Comment); err != nil {
		util.Logger.Error("项目审核失败", zap.Error(err), zap.Int("project_id", projectID), zap.Bool("approved", input.Approved))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to review project"})
		return
	}

	util.Logger.Info("项目审核完成", zap.Int("project_id", projectID), zap.Bool("approved", input.Approved))
	c.JSON(http.StatusOK, gin.H{"message": "Project review completed"})
}

// 实现其他必要的处理器方法
// ...

// GetProjectByID 获取项目详情
func (h *ProjectHandler) GetProjectByID(id int) (*model.Project, []model.ProjectGoal, []model.ProjectImage, error) {
	project, err := h.projectService.GetProjectByID(id)
	if err != nil {
		return nil, nil, nil, err
	}

	goals, err := h.projectService.GetProjectGoals(id)
	if err != nil {
		return nil, nil, nil, err
	}

	images, err := h.projectService.GetProjectImages(id)
	if err != nil {
		return nil, nil, nil, err
	}

	return project, goals, images, nil
}

// GetProjects 获取项目列表
func (h *ProjectHandler) GetProjects(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	projects, err := h.projectService.ListProjects(page, pageSize)
	if err != nil {
		util.Logger.Error("获取项目列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list projects"})
		return
	}

	c.JSON(http.StatusOK, projects)
}

// CreateCategory 处理创建项目分类的请求
func (h *ProjectHandler) CreateCategory(c *gin.Context) {
	var category model.ProjectCategory
	if err := c.ShouldBindJSON(&category); err != nil {
		util.Logger.Warn("无效的项目分类数据", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.projectService.CreateCategory(&category); err != nil {
		util.Logger.Error("创建项目分类失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category"})
		return
	}

	util.Logger.Info("项目分类创建成功", zap.Int("category_id", category.ID))
	c.JSON(http.StatusCreated, category)
}

// GetCategories 处理获取所有项目分类的请求
func (h *ProjectHandler) GetCategories(c *gin.Context) {
	categories, err := h.projectService.GetCategories()
	if err != nil {
		util.Logger.Error("获取项目分类失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get categories"})
		return
	}

	util.Logger.Info("成功获取项目分类", zap.Int("count", len(categories)))
	c.JSON(http.StatusOK, categories)
}

// SearchProjects 处理项目搜索请求
func (h *ProjectHandler) SearchProjects(c *gin.Context) {
	var filters model.ProjectFilters
	if err := c.ShouldBindJSON(&filters); err != nil {
		util.Logger.Warn("无效的搜索过滤条件", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid search filters"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	util.Logger.Info("开始搜索项目", zap.Any("filters", filters), zap.Int("page", page), zap.Int("pageSize", pageSize))

	projects, totalCount, err := h.projectService.SearchProjects(filters, page, pageSize)
	if err != nil {
		util.Logger.Error("搜索项目失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search projects"})
		return
	}

	util.Logger.Info("项目搜索成功", zap.Int("results", len(projects)), zap.Int("total", totalCount))
	c.JSON(http.StatusOK, gin.H{
		"projects": projects,
		"total":    totalCount,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetTags 处理获取所有项目标签的请求
func (h *ProjectHandler) GetTags(c *gin.Context) {
	util.Logger.Info("开始获取所有项目标签")

	tags, err := h.projectService.GetTags()
	if err != nil {
		util.Logger.Error("获取项目标签失", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tags"})
		return
	}

	util.Logger.Info("成功获取项目标签", zap.Int("count", len(tags)))
	c.JSON(http.StatusOK, tags)
}

// CreateProjectUpdate 处理创建项目更新的请求
func (h *ProjectHandler) CreateProjectUpdate(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		util.Logger.Warn("无效的项目ID", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	var update model.ProjectUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		util.Logger.Warn("无效的项目更新数据", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	update.ProjectID = projectID

	util.Logger.Info("开始创建项目更新", zap.Int("project_id", projectID))

	if err := h.projectService.CreateProjectUpdate(&update); err != nil {
		util.Logger.Error("创建项目更新失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project update"})
		return
	}

	util.Logger.Info("项目更新创建成功", zap.Int("update_id", update.ID))
	c.JSON(http.StatusCreated, update)
}

// GetProjectUpdates 处理获取项目更新的请求
func (h *ProjectHandler) GetProjectUpdates(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		util.Logger.Warn("无效的项目ID", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	util.Logger.Info("开始获取项目更新", zap.Int("project_id", projectID))

	updates, err := h.projectService.GetProjectUpdates(projectID)
	if err != nil {
		util.Logger.Error("获取项目更新失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get project updates"})
		return
	}

	util.Logger.Info("成功获取项目更新", zap.Int("project_id", projectID), zap.Int("count", len(updates)))
	c.JSON(http.StatusOK, updates)
}

// CreateProjectComment 处理创建项目评论的请求
func (h *ProjectHandler) CreateProjectComment(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		util.Logger.Warn("无效的项目ID", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	var comment model.ProjectComment
	if err := c.ShouldBindJSON(&comment); err != nil {
		util.Logger.Warn("无效的项目评论数据", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comment.ProjectID = projectID
	userID, _ := c.Get("user_id")
	comment.UserID = userID.(int)

	util.Logger.Info("开始创建项目评论", zap.Int("project_id", projectID), zap.Int("user_id", comment.UserID))

	if err := h.projectService.CreateProjectComment(&comment); err != nil {
		util.Logger.Error("创建项目评论失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project comment"})
		return
	}

	util.Logger.Info("项目评论创建成功", zap.Int("comment_id", comment.ID))
	c.JSON(http.StatusCreated, comment)
}

// GetProjectComments 处理获取项目评论的请求
func (h *ProjectHandler) GetProjectComments(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		util.Logger.Warn("无效的项目ID", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	util.Logger.Info("开始获取项目评论", zap.Int("project_id", projectID), zap.Int("page", page), zap.Int("pageSize", pageSize))

	comments, err := h.projectService.GetProjectComments(projectID, page, pageSize)
	if err != nil {
		util.Logger.Error("获取项目评论失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get project comments"})
		return
	}

	util.Logger.Info("成功获取项目评论", zap.Int("project_id", projectID), zap.Int("count", len(comments)))
	c.JSON(http.StatusOK, comments)
}

// CreateTag 处理创建项目标签的请求
func (h *ProjectHandler) CreateTag(c *gin.Context) {
	var tag model.ProjectTag
	if err := c.ShouldBindJSON(&tag); err != nil {
		util.Logger.Warn("无效的项目标签数据", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	util.Logger.Info("开始创建项目标签", zap.String("name", tag.Name))

	if err := h.projectService.CreateTag(&tag); err != nil {
		util.Logger.Error("创建项目标签失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tag"})
		return
	}

	util.Logger.Info("项目标签创建成功", zap.Int("tag_id", tag.ID))
	c.JSON(http.StatusCreated, tag)
}

// 在 ProjectHandler 结构体中添加 AddTagToProject 方法
func (h *ProjectHandler) AddTagToProject(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		util.Logger.Warn("无效项目ID", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	var input struct {
		TagID int `json:"tag_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		util.Logger.Warn("无效的请求数据", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.projectService.AddTagToProject(projectID, input.TagID)
	if err != nil {
		util.Logger.Error("为项添加标签失", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add tag to project"})
		return
	}

	util.Logger.Info("成功为项目添加标签", zap.Int("project_id", projectID), zap.Int("tag_id", input.TagID))
	c.JSON(http.StatusOK, gin.H{"message": "Tag added to project successfully"})
}

// ... 实现其他新的处理函数 ...

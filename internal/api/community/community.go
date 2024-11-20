package community

import (
	"crowdfunding-backend/internal/model"
	"crowdfunding-backend/internal/service"
	"crowdfunding-backend/internal/storage"
	"crowdfunding-backend/internal/util"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type CommunityHandler struct {
	communityService *service.CommunityService
	storage          *storage.LocalStorage
}

func NewCommunityHandler(communityService *service.CommunityService, storage *storage.LocalStorage) *CommunityHandler {
	return &CommunityHandler{
		communityService: communityService,
		storage:          storage,
	}
}

func (h *CommunityHandler) CreatePost(c *gin.Context) {
	// 解析多部分表单
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		util.Logger.Error("无法解析表单数据", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法解析表单数据"})
		return
	}

	content := c.PostForm("content")
	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "内容不能为空"})
		return
	}

	userID, _ := c.Get("user_id")
	post := &model.Post{
		UserID:  userID.(int),
		Content: content,
	}

	// 处理多张图片
	form, _ := c.MultipartForm()
	files := form.File["images[]"]
	var images []string
	for _, file := range files {
		filename := util.GenerateUniqueFilename(file.Filename)
		path := fmt.Sprintf("posts/%d/%s", post.ID, filename)
		imageURL, err := h.storage.UploadFile(file, path)
		if err != nil {
			util.Logger.Error("图片上传失败", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "图片上传失败"})
			return
		}
		images = append(images, imageURL)
	}

	if err := h.communityService.CreatePost(post, images); err != nil {
		util.Logger.Error("创建帖子失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建帖子失败",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 201,
		"data": post,
	})
}

func (h *CommunityHandler) GetPost(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的帖子ID",
		})
		return
	}

	post, err := h.communityService.GetPostByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取帖子失败",
		})
		return
	}

	// 获取点赞数和评论数
	likeCount, _ := h.communityService.GetLikeCount(id)
	commentCount, _ := h.communityService.GetCommentCount(id)
	post.LikeCount = likeCount
	post.CommentCount = commentCount

	// 获取当前用户的点赞状态
	if userID, exists := c.Get("user_id"); exists {
		isLiked, _ := h.communityService.IsPostLikedByUser(id, userID.(int))
		post.IsLiked = isLiked
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": post,
	})
}

func (h *CommunityHandler) UpdatePost(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		util.Logger.Error("无效的帖子ID", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的帖子ID"})
		return
	}

	var post model.Post
	if err := c.ShouldBindJSON(&post); err != nil {
		util.Logger.Error("无效的帖子数据", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的帖子数据"})
		return
	}

	post.ID = id
	if err := h.communityService.UpdatePost(&post); err != nil {
		util.Logger.Error("更新帖子失败", zap.Error(err), zap.Int("post_id", id))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新帖子失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "帖子更新成功"})
}

func (h *CommunityHandler) DeletePost(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		util.Logger.Error("无效的帖子ID", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的帖子ID"})
		return
	}

	if err := h.communityService.DeletePost(id); err != nil {
		util.Logger.Error("删除帖子失败", zap.Error(err), zap.Int("post_id", id))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除帖子失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "帖子删除成功"})
}

func (h *CommunityHandler) ListPosts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	posts, total, err := h.communityService.ListPosts(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取帖子列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"posts": posts,
			"pagination": gin.H{
				"total":       total,
				"page":        page,
				"page_size":   pageSize,
				"total_pages": (total + pageSize - 1) / pageSize,
			},
		},
	})
}

// 实现其他处理器方法...

// 这里需要实现 CreateComment, ListComments, DeleteComment,
// LikePost, UnlikePost, FollowUser, UnfollowUser,
// GetFollowers, 和 GetFollowing 方法

func (h *CommunityHandler) CreateComment(c *gin.Context) {
	postID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid post ID",
		})
		return
	}

	var comment model.Comment
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid form data",
		})
		return
	}

	comment.Content = c.PostForm("content")
	if comment.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Content is required",
		})
		return
	}

	comment.PostID = postID
	userID, _ := c.Get("user_id")
	comment.UserID = userID.(int)

	// 处理可选的图片上传
	file, err := c.FormFile("image")
	if err == nil {
		filename := util.GenerateUniqueFilename(file.Filename)
		path := fmt.Sprintf("comments/%d/%s", postID, filename)
		imageURL, err := h.storage.UploadFile(file, path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "Failed to upload image",
			})
			return
		}
		comment.ImageURL = imageURL
	}

	if err := h.communityService.CreateComment(&comment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to create comment",
		})
		return
	}

	// 获取用户信息
	user, err := h.communityService.GetUserByID(comment.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to get user info",
		})
		return
	}
	comment.User = user

	c.JSON(http.StatusCreated, comment)
}

func (h *CommunityHandler) ListComments(c *gin.Context) {
	postID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		util.Logger.Error("无效的帖子ID", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的帖子ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	comments, err := h.communityService.GetCommentsByPostID(postID, page, pageSize)
	if err != nil {
		util.Logger.Error("获取评论列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取评论列表失败"})
		return
	}

	c.JSON(http.StatusOK, comments)
}

func (h *CommunityHandler) DeleteComment(c *gin.Context) {
	commentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		util.Logger.Error("无效的评论ID", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的评论ID"})
		return
	}

	if err := h.communityService.DeleteComment(commentID); err != nil {
		util.Logger.Error("删除评论失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除评论失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "评论删除成功"})
}

func (h *CommunityHandler) LikePost(c *gin.Context) {
	postID, _ := strconv.Atoi(c.Param("id"))
	userID, _ := c.Get("user_id")

	// 先检查是否已经点赞
	isLiked, err := h.communityService.IsPostLikedByUser(postID, userID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "检查点赞状态失败",
		})
		return
	}

	if isLiked {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "已经点赞过了",
		})
		return
	}

	// 创建点赞记录
	err = h.communityService.CreateLike(&model.Like{
		UserID: userID.(int),
		PostID: postID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "点赞失败",
		})
		return
	}

	// 返回最新的点赞数和状态
	likeCount, _ := h.communityService.GetLikeCount(postID)
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"is_liked":   true,
			"like_count": likeCount,
		},
	})
}

func (h *CommunityHandler) UnlikePost(c *gin.Context) {
	postID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的帖子ID",
		})
		return
	}

	userID, _ := c.Get("user_id")

	// 先检查是否已经点赞
	isLiked, err := h.communityService.IsPostLikedByUser(postID, userID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "检查点赞状态失败",
		})
		return
	}

	if !isLiked {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "还没有点赞",
		})
		return
	}

	if err := h.communityService.DeleteLike(userID.(int), postID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "取消点赞失败",
		})
		return
	}

	likeCount, _ := h.communityService.GetLikeCount(postID)
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"is_liked":   false,
			"like_count": likeCount,
		},
	})
}

func (h *CommunityHandler) FollowUser(c *gin.Context) {
	followedID, _ := strconv.Atoi(c.Param("id"))
	followerID, _ := c.Get("user_id")

	err := h.communityService.CreateFollow(&model.Follow{
		FollowerID: followerID.(int),
		FollowedID: followedID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "关注失败",
		})
		return
	}

	followerCount, _ := h.communityService.GetFollowerCount(followedID)
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"is_following":   true,
			"follower_count": followerCount,
		},
	})
}

func (h *CommunityHandler) UnfollowUser(c *gin.Context) {
	followedID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		util.Logger.Error("无效的用户ID", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	followerID, _ := c.Get("user_id")

	if err := h.communityService.DeleteFollow(followerID.(int), followedID); err != nil {
		util.Logger.Error("取消关注失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取消关注失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "取消关注成功"})
}

func (h *CommunityHandler) GetFollowers(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		util.Logger.Error("无效的用户ID", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	followers, err := h.communityService.GetFollowers(userID, page, pageSize)
	if err != nil {
		util.Logger.Error("获取关注者列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取关注者列表失败"})
		return
	}

	c.JSON(http.StatusOK, followers)
}

func (h *CommunityHandler) GetFollowing(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		util.Logger.Error("无效的用户ID", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	following, err := h.communityService.GetFollowing(userID, page, pageSize)
	if err != nil {
		util.Logger.Error("获取关注列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取关注列表失败"})
		return
	}

	c.JSON(http.StatusOK, following)
}

func (h *CommunityHandler) ListAllPosts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	posts, total, err := h.communityService.ListAllPosts(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取帖子列表失败",
		})
		return
	}

	// 获取当前用户ID（如果已登录）
	userID, exists := c.Get("user_id")
	if exists {
		// 为每个帖子添加点赞和关注状态
		for _, post := range posts {
			// 检查点赞状态
			isLiked, _ := h.communityService.IsPostLikedByUser(post.ID, userID.(int))
			post.IsLiked = isLiked

			// 检查是否关注了帖子作者
			isFollowing, _ := h.communityService.IsFollowing(userID.(int), post.UserID)
			post.IsFollowing = isFollowing
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"posts": posts,
			"pagination": gin.H{
				"total":       total,
				"page":        page,
				"page_size":   pageSize,
				"total_pages": (total + pageSize - 1) / pageSize,
			},
		},
	})
}

func (h *CommunityHandler) GetCurrentUserFollowing(c *gin.Context) {
	userID, _ := c.Get("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	following, err := h.communityService.GetFollowing(userID.(int), page, pageSize)
	if err != nil {
		util.Logger.Error("获取当前用户关注列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取关注列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"following": following,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *CommunityHandler) GetUserPosts(c *gin.Context) {
	userID, _ := strconv.Atoi(c.Param("id"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	posts, total, err := h.communityService.GetUserPosts(userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取用户帖子失",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"posts": posts,
			"pagination": gin.H{
				"total":       total,
				"page":        page,
				"page_size":   pageSize,
				"total_pages": (total + pageSize - 1) / pageSize,
			},
		},
	})
}

// CreateCommentReply 创建评论回复
func (h *CommunityHandler) CreateCommentReply(c *gin.Context) {
	commentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		util.Logger.Error("无效的评论ID", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid comment ID",
		})
		return
	}

	util.Logger.Info("开始创建评论回复", zap.Int("comment_id", commentID))

	// 先获取父评论信息
	parentComment, err := h.communityService.GetCommentByID(commentID)
	if err != nil {
		util.Logger.Error("获取父评论失败",
			zap.Error(err),
			zap.Int("comment_id", commentID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to get parent comment",
		})
		return
	}

	util.Logger.Info("获取父评论成功",
		zap.Int("comment_id", commentID),
		zap.Int("post_id", parentComment.PostID))

	var comment model.Comment
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid form data",
		})
		return
	}

	comment.Content = c.PostForm("content")
	if comment.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Content is required",
		})
		return
	}

	comment.ParentID = &commentID
	comment.PostID = parentComment.PostID
	userID, _ := c.Get("user_id")
	comment.UserID = userID.(int)

	// 处理可选的图片上传
	file, err := c.FormFile("image")
	if err == nil {
		filename := util.GenerateUniqueFilename(file.Filename)
		path := fmt.Sprintf("comments/%d/%s", commentID, filename)
		imageURL, err := h.storage.UploadFile(file, path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "Failed to upload image",
			})
			return
		}
		comment.ImageURL = imageURL
	}

	if err := h.communityService.CreateCommentReply(&comment); err != nil {
		util.Logger.Error("创建回复失败",
			zap.Error(err),
			zap.Any("comment", comment))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to create reply",
		})
		return
	}

	util.Logger.Info("回复创建成功",
		zap.Int("comment_id", comment.ID),
		zap.Int("parent_id", *comment.ParentID))

	// 获取用户信息
	user, err := h.communityService.GetUserByID(comment.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to get user info",
		})
		return
	}
	comment.User = user

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": comment,
	})
}

// GetCommentReplies 获取评论回复列表
func (h *CommunityHandler) GetCommentReplies(c *gin.Context) {
	commentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid comment ID",
		})
		return
	}

	replies, err := h.communityService.GetCommentReplies(commentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to get replies",
		})
		return
	}

	// 如果没有回复，返回空数组而不是 null
	if replies == nil {
		replies = []*model.Comment{}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"replies": replies,
			"total":   len(replies),
		},
	})
}

// GetFollowingPosts 获取关注者的帖子列表
func (h *CommunityHandler) GetFollowingPosts(c *gin.Context) {
	userID, _ := c.Get("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	posts, total, err := h.communityService.GetFollowingPosts(userID.(int), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to get following posts",
		})
		return
	}

	// 如果没有帖子，返回空数组而不是 null
	if posts == nil {
		posts = []*model.Post{}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"posts": posts,
			"pagination": gin.H{
				"current_page": page,
				"page_size":    pageSize,
				"total":        total,
				"total_pages":  (total + pageSize - 1) / pageSize,
			},
		},
	})
}

// GetFollowersPosts 获取粉丝的帖子列表
func (h *CommunityHandler) GetFollowersPosts(c *gin.Context) {
	userID, _ := c.Get("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	posts, total, err := h.communityService.GetFollowersPosts(userID.(int), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to get followers' posts",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"posts": posts,
			"pagination": gin.H{
				"current_page": page,
				"page_size":    pageSize,
				"total":        total,
				"total_pages":  (total + pageSize - 1) / pageSize,
			},
		},
	})
}

// GetFollowStatus 获取关注状态
func (h *CommunityHandler) GetFollowStatus(c *gin.Context) {
	// 获取目标用户ID
	targetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid user ID",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, _ := c.Get("user_id")

	// 检查是否已关注
	isFollowing, err := h.communityService.IsFollowing(currentUserID.(int), targetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to check follow status",
		})
		return
	}

	// 获取关注者数量
	followerCount, err := h.communityService.GetFollowerCount(targetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to get follower count",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"is_following":   isFollowing,
			"follower_count": followerCount,
		},
	})
}

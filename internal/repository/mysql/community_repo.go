package mysql

import (
	"crowdfunding-backend/config"
	"crowdfunding-backend/internal/model"
	"crowdfunding-backend/internal/util"
	"database/sql"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

type communityRepository struct {
	db *sql.DB
}

func NewCommunityRepository(db *sql.DB) *communityRepository {
	return &communityRepository{db: db}
}

func (r *communityRepository) CreatePost(post *model.Post, images []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 插入帖子
	query := `INSERT INTO posts (user_id, content, created_at, updated_at) 
              VALUES (?, ?, NOW(), NOW())`
	result, err := tx.Exec(query, post.UserID, post.Content)
	if err != nil {
		util.Logger.Error("创建帖子失败", zap.Error(err))
		return err
	}

	postID, err := result.LastInsertId()
	if err != nil {
		util.Logger.Error("获取新帖子ID失败", zap.Error(err))
		return err
	}
	post.ID = int(postID)

	// 插入图片
	if len(images) > 0 {
		query = `INSERT INTO post_images (post_id, image_url, created_at) VALUES (?, ?, NOW())`
		for _, imageURL := range images {
			_, err = tx.Exec(query, postID, imageURL)
			if err != nil {
				util.Logger.Error("插入帖子图片失败", zap.Error(err))
				return err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		util.Logger.Error("提交事务失败", zap.Error(err))
		return err
	}

	post.Images = images
	util.Logger.Info("帖子创建成功", zap.Int("post_id", post.ID))
	return nil
}

func (r *communityRepository) GetPostByID(id int) (*model.Post, error) {
	// 获取帖子基本信息
	query := `
        SELECT p.id, p.user_id, p.content, p.created_at, p.updated_at,
               u.username, u.email, u.avatar_url, u.bio
        FROM posts p
        LEFT JOIN users u ON p.user_id = u.id
        WHERE p.id = ?`

	var post model.Post
	var user model.User
	err := r.db.QueryRow(query, id).Scan(
		&post.ID, &post.UserID, &post.Content,
		&post.CreatedAt, &post.UpdatedAt,
		&user.Username, &user.Email, &user.AvatarURL, &user.Bio,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// 获取帖子图片
	query = `SELECT image_url FROM post_images WHERE post_id = ? ORDER BY created_at ASC`
	rows, err := r.db.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []string
	for rows.Next() {
		var imageURL string
		if err := rows.Scan(&imageURL); err != nil {
			return nil, err
		}
		images = append(images, config.AppConfig.BackendURL+"/uploads/"+imageURL)
	}

	post.Images = images
	user.ID = post.UserID
	post.User = &user

	// 获取点赞数
	var likeCount int
	err = r.db.QueryRow(`
        SELECT COUNT(*) 
        FROM likes 
        WHERE post_id = ?
    `, id).Scan(&likeCount)
	if err != nil {
		return nil, err
	}
	post.LikeCount = likeCount

	return &post, nil
}

func (r *communityRepository) UpdatePost(post *model.Post) error {
	query := `UPDATE posts SET content = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.db.Exec(query, post.Content, post.ID)
	if err != nil {
		util.Logger.Error("更新帖子失败", zap.Error(err), zap.Int("post_id", post.ID))
		return err
	}
	return nil
}

func (r *communityRepository) DeletePost(id int) error {
	util.Logger.Info("开始删除帖子", zap.Int("post_id", id))

	query := `DELETE FROM posts WHERE id = ?`
	_, err := r.db.Exec(query, id)
	if err != nil {
		util.Logger.Error("删除帖子失败", zap.Error(err), zap.Int("post_id", id))
		return err
	}

	util.Logger.Info("帖子删除成功", zap.Int("post_id", id))
	return nil
}

func (r *communityRepository) ListPosts(page, pageSize int) ([]*model.Post, int, error) {
	// 首先取总数
	var total int
	err := r.db.QueryRow("SELECT COUNT(*) FROM posts").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query := `SELECT id, user_id, content, created_at, updated_at 
              FROM posts 
              ORDER BY created_at DESC 
              LIMIT ? OFFSET ?`

	rows, err := r.db.Query(query, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var posts []*model.Post
	for rows.Next() {
		var post model.Post
		err := rows.Scan(
			&post.ID, &post.UserID, &post.Content,
			&post.CreatedAt, &post.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		posts = append(posts, &post)
	}

	return posts, total, nil
}

func (r *communityRepository) CreateComment(comment *model.Comment) error {
	// 添加日志
	util.Logger.Info("开始创建评论",
		zap.Int("user_id", comment.UserID),
		zap.Int("post_id", comment.PostID),
		zap.Any("parent_id", comment.ParentID))

	query := `INSERT INTO comments 
		(user_id, post_id, parent_id, content, image_url, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, NOW(), NOW())`

	result, err := r.db.Exec(query,
		comment.UserID,
		comment.PostID,
		comment.ParentID,
		comment.Content,
		comment.ImageURL)

	if err != nil {
		util.Logger.Error("创建评论失败",
			zap.Error(err),
			zap.Any("comment", comment))
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		util.Logger.Error("获取新评论ID失败", zap.Error(err))
		return err
	}
	comment.ID = int(id)

	util.Logger.Info("评论创建成功",
		zap.Int("comment_id", comment.ID),
		zap.Any("parent_id", comment.ParentID))
	return nil
}

func (r *communityRepository) GetCommentsByPostID(postID, page, pageSize int) ([]*model.Comment, error) {
	offset := (page - 1) * pageSize
	query := `
        SELECT c.id, c.user_id, c.post_id, c.content, c.image_url, c.created_at, c.updated_at,
               u.username, u.email, u.avatar_url, u.bio
        FROM comments c
        LEFT JOIN users u ON c.user_id = u.id
        WHERE c.post_id = ?
        ORDER BY c.created_at DESC
        LIMIT ? OFFSET ?`

	rows, err := r.db.Query(query, postID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*model.Comment
	for rows.Next() {
		var comment model.Comment
		var user model.User
		err := rows.Scan(
			&comment.ID, &comment.UserID, &comment.PostID, &comment.Content, &comment.ImageURL,
			&comment.CreatedAt, &comment.UpdatedAt,
			&user.Username, &user.Email, &user.AvatarURL, &user.Bio,
		)
		if err != nil {
			return nil, err
		}
		user.ID = comment.UserID
		comment.User = &user
		comments = append(comments, &comment)
	}

	return comments, nil
}

func (r *communityRepository) DeleteComment(id int) error {
	util.Logger.Info("开始删除评论", zap.Int("comment_id", id))

	query := `DELETE FROM comments WHERE id = ?`
	_, err := r.db.Exec(query, id)
	if err != nil {
		util.Logger.Error("删除评论失败", zap.Error(err), zap.Int("comment_id", id))
		return err
	}

	util.Logger.Info("评论除成功", zap.Int("comment_id", id))
	return nil
}

func (r *communityRepository) CreateLike(like *model.Like) error {
	// 使用事务确保原子性
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 检查帖子是否存在
	var exists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM posts WHERE id = ?)", like.PostID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("post not found")
	}

	// 尝试插入点赞记录
	query := `INSERT INTO likes (user_id, post_id, created_at) VALUES (?, ?, NOW())`
	_, err = tx.Exec(query, like.UserID, like.PostID)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			return fmt.Errorf("already liked")
		}
		return err
	}

	return tx.Commit()
}

func (r *communityRepository) DeleteLike(userID, postID int) error {
	// 使用事务确保原子性
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 查点赞记录是否存在
	var exists bool
	err = tx.QueryRow(`
        SELECT EXISTS(
            SELECT 1 FROM likes 
            WHERE user_id = ? AND post_id = ?
        )`, userID, postID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("like not found")
	}

	// 删除点赞记录
	_, err = tx.Exec(`DELETE FROM likes WHERE user_id = ? AND post_id = ?`, userID, postID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *communityRepository) GetLikeCount(postID int) (int, error) {
	var count int
	err := r.db.QueryRow(`
        SELECT COUNT(*) 
        FROM likes 
        WHERE post_id = ?
    `, postID).Scan(&count)
	return count, err
}

func (r *communityRepository) CreateFollow(follow *model.Follow) error {
	util.Logger.Info("开始创建关注", zap.Int("follower_id", follow.FollowerID), zap.Int("followed_id", follow.FollowedID))

	query := `INSERT INTO follows (follower_id, followed_id, created_at) 
              VALUES (?, ?, NOW())`
	result, err := r.db.Exec(query, follow.FollowerID, follow.FollowedID)
	if err != nil {
		util.Logger.Error("创建关注失败", zap.Error(err))
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		util.Logger.Error("获取新关注ID失败", zap.Error(err))
		return err
	}
	follow.ID = int(id)

	util.Logger.Info("关注创建成功", zap.Int("follow_id", follow.ID))
	return nil
}

func (r *communityRepository) DeleteFollow(followerID, followedID int) error {
	util.Logger.Info("开始删除关注", zap.Int("follower_id", followerID), zap.Int("followed_id", followedID))

	query := `DELETE FROM follows WHERE follower_id = ? AND followed_id = ?`
	_, err := r.db.Exec(query, followerID, followedID)
	if err != nil {
		util.Logger.Error("删除关注失败", zap.Error(err))
		return err
	}

	util.Logger.Info("关注删除成功")
	return nil
}

func (r *communityRepository) GetFollowers(userID, page, pageSize int) ([]*model.User, error) {
	util.Logger.Info("开始获取关注者列表", zap.Int("user_id", userID))

	offset := (page - 1) * pageSize
	query := `
        SELECT u.id, u.username, u.email, u.avatar_url, u.bio
        FROM users u
        JOIN follows f ON u.id = f.follower_id
        WHERE f.followed_id = ?
        ORDER BY f.created_at DESC
        LIMIT ? OFFSET ?`

	rows, err := r.db.Query(query, userID, pageSize, offset)
	if err != nil {
		util.Logger.Error("获取关注者列表失败", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var followers []*model.User
	for rows.Next() {
		var user model.User
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.AvatarURL, &user.Bio)
		if err != nil {
			util.Logger.Error("扫描关注者数据失败", zap.Error(err))
			return nil, err
		}
		followers = append(followers, &user)
	}

	util.Logger.Info("成功获取关注者列表", zap.Int("count", len(followers)))
	return followers, nil
}

func (r *communityRepository) GetFollowing(userID, page, pageSize int) ([]*model.User, error) {
	util.Logger.Info("开始获取关注列表", zap.Int("user_id", userID))

	offset := (page - 1) * pageSize
	query := `
        SELECT u.id, u.username, u.email, u.avatar_url, u.bio
        FROM users u
        JOIN follows f ON u.id = f.followed_id
        WHERE f.follower_id = ?
        ORDER BY f.created_at DESC
        LIMIT ? OFFSET ?`

	rows, err := r.db.Query(query, userID, pageSize, offset)
	if err != nil {
		util.Logger.Error("获取关注列表失败", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var following []*model.User
	for rows.Next() {
		var user model.User
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.AvatarURL, &user.Bio)
		if err != nil {
			util.Logger.Error("扫描关注数据失败", zap.Error(err))
			return nil, err
		}
		following = append(following, &user)
	}

	util.Logger.Info("成功获取关注列表", zap.Int("count", len(following)))
	return following, nil
}

func (r *communityRepository) ListAllPosts(page, pageSize int) ([]*model.Post, int, error) {
	// 首先获取总数
	var total int
	err := r.db.QueryRow("SELECT COUNT(*) FROM posts").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	offset := (page - 1) * pageSize
	query := `
        SELECT p.id, p.user_id, p.content, p.created_at, p.updated_at,
               u.username, u.email, u.avatar_url, u.bio
        FROM posts p
        LEFT JOIN users u ON p.user_id = u.id
        ORDER BY p.created_at DESC
        LIMIT ? OFFSET ?`

	rows, err := r.db.Query(query, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var posts []*model.Post
	for rows.Next() {
		var post model.Post
		var user model.User
		err := rows.Scan(
			&post.ID, &post.UserID, &post.Content,
			&post.CreatedAt, &post.UpdatedAt,
			&user.Username, &user.Email, &user.AvatarURL, &user.Bio,
		)
		if err != nil {
			return nil, 0, err
		}

		// 获取帖子图片
		query = `SELECT image_url FROM post_images WHERE post_id = ? ORDER BY created_at ASC`
		imageRows, err := r.db.Query(query, post.ID)
		if err != nil {
			return nil, 0, err
		}
		defer imageRows.Close()

		var images []string
		for imageRows.Next() {
			var imageURL string
			if err := imageRows.Scan(&imageURL); err != nil {
				return nil, 0, err
			}
			images = append(images, config.AppConfig.BackendURL+"/uploads/"+imageURL)
		}
		post.Images = images

		user.ID = post.UserID
		post.User = &user
		posts = append(posts, &post)
	}

	// 为每个帖子获取点赞数
	for _, post := range posts {
		var likeCount int
		err = r.db.QueryRow(`
            SELECT COUNT(*) 
            FROM likes 
            WHERE post_id = ?
        `, post.ID).Scan(&likeCount)
		if err != nil {
			return nil, 0, err
		}
		post.LikeCount = likeCount
	}

	return posts, total, nil
}

// 添加其他必要的方法实现
func (r *communityRepository) GetCommentCount(postID int) (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM comments WHERE post_id = ?", postID).Scan(&count)
	return count, err
}

func (r *communityRepository) IsPostLikedByUser(postID, userID int) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`
        SELECT EXISTS(
            SELECT 1 FROM likes 
            WHERE post_id = ? AND user_id = ?
        )
    `, postID, userID).Scan(&exists)
	return exists, err
}

func (r *communityRepository) GetFollowerCount(userID int) (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM follows WHERE followed_id = ?", userID).Scan(&count)
	return count, err
}

func (r *communityRepository) GetUserPosts(userID, page, pageSize int) ([]*model.Post, int, error) {
	// 获取总数
	var total int
	err := r.db.QueryRow("SELECT COUNT(*) FROM posts WHERE user_id = ?", userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	offset := (page - 1) * pageSize
	query := `
        SELECT p.id, p.user_id, p.content, p.created_at, p.updated_at,
               u.username, u.email, u.avatar_url, u.bio
        FROM posts p
        LEFT JOIN users u ON p.user_id = u.id
        WHERE p.user_id = ?
        ORDER BY p.created_at DESC
        LIMIT ? OFFSET ?`

	rows, err := r.db.Query(query, userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var posts []*model.Post
	for rows.Next() {
		var post model.Post
		var user model.User
		err := rows.Scan(
			&post.ID, &post.UserID, &post.Content,
			&post.CreatedAt, &post.UpdatedAt,
			&user.Username, &user.Email, &user.AvatarURL, &user.Bio,
		)
		if err != nil {
			return nil, 0, err
		}

		user.ID = post.UserID
		post.User = &user
		posts = append(posts, &post)
	}

	return posts, total, nil
}

func (r *communityRepository) IsFollowing(followerID, followedID int) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`
        SELECT EXISTS(
            SELECT 1 FROM follows 
            WHERE follower_id = ? AND followed_id = ?
        )
    `, followerID, followedID).Scan(&exists)
	return exists, err
}

// GetCommentReplies 获取评论的回复列表
func (r *communityRepository) GetCommentReplies(commentID int) ([]*model.Comment, error) {
	// 添加日志
	util.Logger.Info("开始获取评论回复",
		zap.Int("comment_id", commentID))

	query := `
        SELECT c.id, c.user_id, c.post_id, c.parent_id, c.content, c.image_url, 
               c.created_at, c.updated_at,
               u.id as user_id, u.username, u.email, u.avatar_url, u.bio
        FROM comments c
        LEFT JOIN users u ON c.user_id = u.id
        WHERE c.parent_id = ?
        ORDER BY c.created_at ASC`

	// 打印SQL查询
	util.Logger.Debug("执行SQL查询",
		zap.String("query", query),
		zap.Int("comment_id", commentID))

	rows, err := r.db.Query(query, commentID)
	if err != nil {
		util.Logger.Error("查询评论回复失败",
			zap.Error(err),
			zap.Int("comment_id", commentID))
		return nil, err
	}
	defer rows.Close()

	var replies []*model.Comment
	for rows.Next() {
		var comment model.Comment
		var user model.User
		err := rows.Scan(
			&comment.ID, &comment.UserID, &comment.PostID, &comment.ParentID,
			&comment.Content, &comment.ImageURL, &comment.CreatedAt, &comment.UpdatedAt,
			&user.ID, &user.Username, &user.Email, &user.AvatarURL, &user.Bio,
		)
		if err != nil {
			util.Logger.Error("扫描评论回复数据失败",
				zap.Error(err))
			return nil, err
		}
		comment.User = &user
		replies = append(replies, &comment)
	}

	if err = rows.Err(); err != nil {
		util.Logger.Error("遍历评论回复数据失败",
			zap.Error(err))
		return nil, err
	}

	util.Logger.Info("成功获取评论回复",
		zap.Int("comment_id", commentID),
		zap.Int("reply_count", len(replies)))

	return replies, nil
}

// GetFollowingPosts 获取关注者的帖子列表
func (r *communityRepository) GetFollowingPosts(userID int, page, pageSize int) ([]*model.Post, int, error) {
	// 先获取总数
	var total int
	countQuery := `
        SELECT COUNT(DISTINCT p.id)
        FROM posts p
        JOIN follows f ON p.user_id = f.followed_id
        WHERE f.follower_id = ?`

	err := r.db.QueryRow(countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 获取帖子列表
	query := `
        SELECT p.id, p.user_id, p.content, p.created_at, p.updated_at,
               u.username, u.email, u.avatar_url, u.bio
        FROM posts p
        JOIN follows f ON p.user_id = f.followed_id
        JOIN users u ON p.user_id = u.id
        WHERE f.follower_id = ?
        ORDER BY p.created_at DESC
        LIMIT ? OFFSET ?`

	offset := (page - 1) * pageSize
	rows, err := r.db.Query(query, userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var posts []*model.Post
	for rows.Next() {
		var post model.Post
		var user model.User
		err := rows.Scan(
			&post.ID, &post.UserID, &post.Content, &post.CreatedAt, &post.UpdatedAt,
			&user.Username, &user.Email, &user.AvatarURL, &user.Bio,
		)
		if err != nil {
			return nil, 0, err
		}

		// 获取帖子图片
		imageQuery := `SELECT image_url FROM post_images WHERE post_id = ? ORDER BY created_at ASC`
		imageRows, err := r.db.Query(imageQuery, post.ID)
		if err != nil {
			return nil, 0, err
		}
		defer imageRows.Close()

		var images []string
		for imageRows.Next() {
			var imageURL string
			if err := imageRows.Scan(&imageURL); err != nil {
				return nil, 0, err
			}
			// 添加完整的图片URL
			images = append(images, config.AppConfig.BackendURL+"/uploads/"+imageURL)
		}
		post.Images = images

		// 获取点赞数
		var likeCount int
		err = r.db.QueryRow(`SELECT COUNT(*) FROM likes WHERE post_id = ?`, post.ID).Scan(&likeCount)
		if err != nil {
			return nil, 0, err
		}
		post.LikeCount = likeCount

		// 获取评论数
		var commentCount int
		err = r.db.QueryRow(`SELECT COUNT(*) FROM comments WHERE post_id = ?`, post.ID).Scan(&commentCount)
		if err != nil {
			return nil, 0, err
		}
		post.CommentCount = commentCount

		// 检查当前用户是否点赞
		var isLiked bool
		err = r.db.QueryRow(`
            SELECT EXISTS(
                SELECT 1 FROM likes 
                WHERE post_id = ? AND user_id = ?
            )
        `, post.ID, userID).Scan(&isLiked)
		if err != nil {
			return nil, 0, err
		}
		post.IsLiked = isLiked

		// 设置用户信息
		user.ID = post.UserID
		post.User = &user
		post.IsFollowing = true // 这里一定是关注的用户的帖子

		posts = append(posts, &post)
	}

	return posts, total, nil
}

// GetFollowersPosts 获取粉丝的帖子列表
func (r *communityRepository) GetFollowersPosts(userID int, page, pageSize int) ([]*model.Post, int, error) {
	// 先获取总数
	var total int
	countQuery := `
        SELECT COUNT(DISTINCT p.id)
        FROM posts p
        JOIN follows f ON p.user_id = f.follower_id
        WHERE f.followed_id = ?`

	err := r.db.QueryRow(countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 获取帖子列表
	query := `
        SELECT p.id, p.user_id, p.content, p.created_at, p.updated_at,
               u.username, u.email, u.avatar_url
        FROM posts p
        JOIN follows f ON p.user_id = f.follower_id
        JOIN users u ON p.user_id = u.id
        WHERE f.followed_id = ?
        ORDER BY p.created_at DESC
        LIMIT ? OFFSET ?`

	offset := (page - 1) * pageSize
	rows, err := r.db.Query(query, userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var posts []*model.Post
	for rows.Next() {
		var post model.Post
		var user model.User
		err := rows.Scan(
			&post.ID, &post.UserID, &post.Content, &post.CreatedAt, &post.UpdatedAt,
			&user.Username, &user.Email, &user.AvatarURL,
		)
		if err != nil {
			return nil, 0, err
		}
		user.ID = post.UserID
		post.User = &user
		posts = append(posts, &post)
	}

	return posts, total, nil
}

func (r *communityRepository) GetCommentByID(id int) (*model.Comment, error) {
	query := `
        SELECT id, user_id, post_id, parent_id, content, image_url, 
               created_at, updated_at
        FROM comments
        WHERE id = ?`

	var comment model.Comment
	err := r.db.QueryRow(query, id).Scan(
		&comment.ID, &comment.UserID, &comment.PostID, &comment.ParentID,
		&comment.Content, &comment.ImageURL, &comment.CreatedAt, &comment.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &comment, nil
}

// GetUserByID 获取用户信息
func (r *communityRepository) GetUserByID(id int) (*model.User, error) {
	query := `
        SELECT id, username, email, avatar_url, bio
        FROM users
        WHERE id = ?`

	var user model.User
	err := r.db.QueryRow(query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.AvatarURL, &user.Bio,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

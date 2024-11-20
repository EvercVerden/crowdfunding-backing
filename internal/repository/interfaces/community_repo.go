package interfaces

import "crowdfunding-backend/internal/model"

// CommunityRepository 定义了社区相关的数据库操作接口
type CommunityRepository interface {
	CreatePost(post *model.Post, images []string) error
	GetPostByID(id int) (*model.Post, error)
	UpdatePost(post *model.Post) error
	DeletePost(id int) error
	ListPosts(page, pageSize int) ([]*model.Post, int, error)
	CreateComment(comment *model.Comment) error
	GetCommentsByPostID(postID, page, pageSize int) ([]*model.Comment, error)
	DeleteComment(id int) error
	CreateLike(like *model.Like) error
	DeleteLike(userID, postID int) error
	GetLikeCount(postID int) (int, error)
	CreateFollow(follow *model.Follow) error
	DeleteFollow(followerID, followedID int) error
	GetFollowers(userID, page, pageSize int) ([]*model.User, error)
	GetFollowing(userID, page, pageSize int) ([]*model.User, error)
	ListAllPosts(page, pageSize int) ([]*model.Post, int, error)
	GetCommentCount(postID int) (int, error)
	IsPostLikedByUser(postID, userID int) (bool, error)
	GetFollowerCount(userID int) (int, error)
	GetUserPosts(userID, page, pageSize int) ([]*model.Post, int, error)
	IsFollowing(followerID, followedID int) (bool, error)
	GetCommentReplies(commentID int) ([]*model.Comment, error)
	GetFollowingPosts(userID int, page, pageSize int) ([]*model.Post, int, error)
	GetFollowersPosts(userID int, page, pageSize int) ([]*model.Post, int, error)
	GetCommentByID(id int) (*model.Comment, error)
	GetUserByID(id int) (*model.User, error)
}

package service

import (
	"crowdfunding-backend/internal/model"
	"crowdfunding-backend/internal/repository/interfaces"
)

type CommunityService struct {
	repo interfaces.CommunityRepository
}

func NewCommunityService(repo interfaces.CommunityRepository) *CommunityService {
	return &CommunityService{repo}
}

func (s *CommunityService) CreatePost(post *model.Post, images []string) error {
	return s.repo.CreatePost(post, images)
}

func (s *CommunityService) GetPostByID(id int) (*model.Post, error) {
	return s.repo.GetPostByID(id)
}

func (s *CommunityService) UpdatePost(post *model.Post) error {
	return s.repo.UpdatePost(post)
}

func (s *CommunityService) DeletePost(id int) error {
	return s.repo.DeletePost(id)
}

func (s *CommunityService) ListPosts(page, pageSize int) ([]*model.Post, int, error) {
	return s.repo.ListPosts(page, pageSize)
}

// 实现其他方法...

// 这里需要实现 CreateComment, GetCommentsByPostID, DeleteComment,
// CreateLike, DeleteLike, GetLikeCount, CreateFollow, DeleteFollow,
// GetFollowers, 和 GetFollowing 方法

func (s *CommunityService) CreateComment(comment *model.Comment) error {
	return s.repo.CreateComment(comment)
}

func (s *CommunityService) GetCommentsByPostID(postID, page, pageSize int) ([]*model.Comment, error) {
	return s.repo.GetCommentsByPostID(postID, page, pageSize)
}

func (s *CommunityService) DeleteComment(id int) error {
	return s.repo.DeleteComment(id)
}

func (s *CommunityService) CreateLike(like *model.Like) error {
	return s.repo.CreateLike(like)
}

func (s *CommunityService) DeleteLike(userID, postID int) error {
	return s.repo.DeleteLike(userID, postID)
}

func (s *CommunityService) GetLikeCount(postID int) (int, error) {
	return s.repo.GetLikeCount(postID)
}

func (s *CommunityService) CreateFollow(follow *model.Follow) error {
	return s.repo.CreateFollow(follow)
}

func (s *CommunityService) DeleteFollow(followerID, followedID int) error {
	return s.repo.DeleteFollow(followerID, followedID)
}

func (s *CommunityService) GetFollowers(userID, page, pageSize int) ([]*model.User, error) {
	return s.repo.GetFollowers(userID, page, pageSize)
}

func (s *CommunityService) GetFollowing(userID, page, pageSize int) ([]*model.User, error) {
	return s.repo.GetFollowing(userID, page, pageSize)
}

func (s *CommunityService) ListAllPosts(page, pageSize int) ([]*model.Post, int, error) {
	return s.repo.ListAllPosts(page, pageSize)
}

func (s *CommunityService) GetCommentCount(postID int) (int, error) {
	return s.repo.GetCommentCount(postID)
}

func (s *CommunityService) IsPostLikedByUser(postID, userID int) (bool, error) {
	return s.repo.IsPostLikedByUser(postID, userID)
}

func (s *CommunityService) GetFollowerCount(userID int) (int, error) {
	return s.repo.GetFollowerCount(userID)
}

func (s *CommunityService) GetUserPosts(userID, page, pageSize int) ([]*model.Post, int, error) {
	return s.repo.GetUserPosts(userID, page, pageSize)
}

func (s *CommunityService) IsFollowing(followerID, followedID int) (bool, error) {
	return s.repo.IsFollowing(followerID, followedID)
}

// 添加评论回复
func (s *CommunityService) CreateCommentReply(comment *model.Comment) error {
	return s.repo.CreateComment(comment)
}

// 获取评论的回复列表
func (s *CommunityService) GetCommentReplies(commentID int) ([]*model.Comment, error) {
	return s.repo.GetCommentReplies(commentID)
}

// 获取关注者的帖子列表
func (s *CommunityService) GetFollowingPosts(userID int, page, pageSize int) ([]*model.Post, int, error) {
	return s.repo.GetFollowingPosts(userID, page, pageSize)
}

// 获取粉丝的帖子列表
func (s *CommunityService) GetFollowersPosts(userID int, page, pageSize int) ([]*model.Post, int, error) {
	return s.repo.GetFollowersPosts(userID, page, pageSize)
}

func (s *CommunityService) GetCommentByID(id int) (*model.Comment, error) {
	return s.repo.GetCommentByID(id)
}

func (s *CommunityService) GetUserByID(id int) (*model.User, error) {
	return s.repo.GetUserByID(id)
}

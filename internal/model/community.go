package model

import "time"

type Post struct {
	ID           int       `json:"id"`
	UserID       int       `json:"user_id"`
	Content      string    `json:"content"`
	Images       []string  `json:"images"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	User         *User     `json:"user,omitempty"`
	LikeCount    int       `json:"like_count"`
	CommentCount int       `json:"comment_count"`
	IsLiked      bool      `json:"is_liked"`
	IsFollowing  bool      `json:"is_following"`
}

type PostImage struct {
	ID        int       `json:"id"`
	PostID    int       `json:"post_id"`
	ImageURL  string    `json:"image_url"`
	CreatedAt time.Time `json:"created_at"`
}

type Comment struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id"`
	PostID    int        `json:"post_id"`
	ParentID  *int       `json:"parent_id,omitempty"`
	Content   string     `json:"content"`
	ImageURL  string     `json:"image_url"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	User      *User      `json:"user"`
	Replies   []*Comment `json:"replies,omitempty"`
}

type Like struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	PostID    int       `json:"post_id"`
	CreatedAt time.Time `json:"created_at"`
}

type Follow struct {
	ID         int       `json:"id"`
	FollowerID int       `json:"follower_id"`
	FollowedID int       `json:"followed_id"`
	CreatedAt  time.Time `json:"created_at"`
}

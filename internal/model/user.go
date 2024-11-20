package model

import "time"

// User 结构体表示用户模型
type User struct {
	ID           int        `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"` // 密码哈希不应在JSON中暴露
	AvatarURL    string     `json:"avatar_url"`
	Bio          string     `json:"bio"`
	Role         string     `json:"role"`
	IsVerified   bool       `json:"is_verified"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at"`
}

// UserAddress 用户地址模型
type UserAddress struct {
	ID            int       `json:"id"`
	UserID        int       `json:"user_id"`
	ReceiverName  string    `json:"receiver_name"`
	Phone         string    `json:"phone"`
	Province      string    `json:"province"`
	City          string    `json:"city"`
	District      string    `json:"district"`
	DetailAddress string    `json:"detail_address"`
	IsDefault     bool      `json:"is_default"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

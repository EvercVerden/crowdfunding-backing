package model

import (
	"time"
)

type Project struct {
	ID              int            `json:"id"`
	Title           string         `json:"title"`
	Description     string         `json:"description"`
	CreatorID       int            `json:"creator_id"`
	Status          string         `json:"status"`
	TotalAmount     float64        `json:"total_amount"`      // 已筹集金额
	TotalGoalAmount float64        `json:"total_goal_amount"` // 所有目标金额之和
	Progress        float64        `json:"progress"`          // 筹款进度（百分比）
	MinRewardAmount float64        `json:"min_reward_amount"` // 最低有奖支持金额
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	EndDate         time.Time      `json:"end_date"`
	CategoryID      *int           `json:"category_id,omitempty"`
	PrimaryImage    string         `json:"primary_image"`
	Images          []ProjectImage `json:"images,omitempty"`
	Goals           []ProjectGoal  `json:"goals,omitempty"` // 所有目标
	LongImages      []string       `json:"long_images,omitempty"`
	Creator         *User          `json:"creator,omitempty"`
}

type ProjectGoal struct {
	ID          int            `json:"id"`
	ProjectID   int            `json:"project_id"`
	Amount      float64        `json:"amount"`
	Description string         `json:"description"`
	IsReached   bool           `json:"is_reached"`
	Progress    float64        `json:"progress"` // 该目标的达成进度
	Images      []ProjectImage `json:"images"`
}

type ProjectImage struct {
	ID        int    `json:"id"`
	ProjectID int    `json:"project_id"`
	GoalID    *int   `json:"goal_id,omitempty"`
	ImageURL  string `json:"image_url"`
	IsPrimary bool   `json:"is_primary"`
	ImageType string `json:"image_type"`
}

type Pledge struct {
	ID        int          `json:"id"`
	UserID    int          `json:"user_id"`
	ProjectID int          `json:"project_id"`
	Amount    float64      `json:"amount"`
	Status    string       `json:"status"`
	AddressID *int         `json:"address_id,omitempty"`
	Address   *UserAddress `json:"address,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}

type ProjectCategory struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type ProjectTag struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type ProjectUpdate struct {
	ID        int       `json:"id"`
	ProjectID int       `json:"project_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type ProjectComment struct {
	ID        int       `json:"id"`
	ProjectID int       `json:"project_id"`
	UserID    int       `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProjectFilters struct {
	Keyword   string    `json:"keyword"`
	Category  int       `json:"category"`
	Status    string    `json:"status"`
	MinAmount float64   `json:"min_amount"`
	MaxAmount float64   `json:"max_amount"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Tags      []int     `json:"tags"`
}

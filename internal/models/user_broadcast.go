package models

import (
	"time"

	"github.com/google/uuid"
)

type UserBroadcast struct {
	ID           uuid.UUID             `json:"id" db:"id"`
	Title        string                `json:"title" db:"title"`
	Message      string                `json:"message" db:"message"`
	TargetType   string                `json:"target_type" db:"target_type"` // 'all' or 'selected'
	Priority     string                `json:"priority" db:"priority"`       // 'low', 'normal', 'high'
	Status       string                `json:"status" db:"status"`           // 'draft', 'scheduled', 'sent', 'cancelled'
	CreatedBy    uuid.UUID             `json:"created_by" db:"created_by"`
	ScheduledAt  *time.Time            `json:"scheduled_at" db:"scheduled_at"`
	SentAt       *time.Time            `json:"sent_at" db:"sent_at"`
	EndDate      *time.Time            `json:"end_date" db:"end_date"`
	SuccessCount int                   `json:"success_count" db:"success_count"`
	ErrorCount   int                   `json:"error_count" db:"error_count"`
	TotalTargets int                   `json:"total_targets" db:"total_targets"`
	CreatedAt    time.Time             `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at" db:"updated_at"`
	CreatorName  *string               `json:"creator_name,omitempty"`
	Targets      []UserBroadcastTarget `json:"targets,omitempty"`
}

type UserBroadcastTarget struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	BroadcastID  uuid.UUID  `json:"broadcast_id" db:"broadcast_id"`
	UserID       uuid.UUID  `json:"user_id" db:"user_id"`
	Status       string     `json:"status" db:"status"` // 'pending', 'sent', 'failed'
	SentAt       *time.Time `json:"sent_at" db:"sent_at"`
	ErrorMessage *string    `json:"error_message" db:"error_message"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UserName     *string    `json:"user_name,omitempty"`
	UserEmail    *string    `json:"user_email,omitempty"`
}

// Request models
type UserBroadcastCreateRequest struct {
	Title       string   `json:"title" binding:"required"`
	Message     string   `json:"message" binding:"required"`
	TargetType  string   `json:"target_type" binding:"required,oneof=all selected"`
	Priority    string   `json:"priority" binding:"required,oneof=low normal high"`
	ScheduledAt *string  `json:"scheduled_at,omitempty"`
	EndDate     *string  `json:"end_date,omitempty"`
	UserIDs     []string `json:"user_ids,omitempty"` // For selected targets
}

type UserBroadcastUpdateRequest struct {
	Title       *string  `json:"title,omitempty"`
	Message     *string  `json:"message,omitempty"`
	TargetType  *string  `json:"target_type,omitempty" binding:"omitempty,oneof=all selected"`
	Priority    *string  `json:"priority,omitempty" binding:"omitempty,oneof=low normal high"`
	ScheduledAt *string  `json:"scheduled_at,omitempty"`
	EndDate     *string  `json:"end_date,omitempty"`
	Status      *string  `json:"status,omitempty" binding:"omitempty,oneof=draft scheduled sent cancelled"`
	UserIDs     []string `json:"user_ids,omitempty"` // For selected targets
}

type UserBroadcastListRequest struct {
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
	Status   string `form:"status"`
	Search   string `form:"search"`
}

type UserBroadcastStats struct {
	Total     int `json:"total"`
	Draft     int `json:"draft"`
	Scheduled int `json:"scheduled"`
	Sent      int `json:"sent"`
	Cancelled int `json:"cancelled"`
}

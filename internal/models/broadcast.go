package models

import "time"

type Broadcast struct {
	ID             string     `json:"id"`
	AdminID        string     `json:"admin_id"`
	Title          string     `json:"title"`
	Message        string     `json:"message"`
	TargetType     string     `json:"target_type"` // "all_groups" or "selected_groups"
	TargetGroupIDs []string   `json:"target_group_ids,omitempty"`
	IsActive       bool       `json:"is_active"`
	Priority       int        `json:"priority"` // 1=normal, 2=high, 3=urgent
	StartDate      time.Time  `json:"start_date"`
	EndDate        *time.Time `json:"end_date,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type BroadcastResponse struct {
	ID             string     `json:"id"`
	Title          string     `json:"title"`
	Message        string     `json:"message"`
	TargetType     string     `json:"target_type"`
	TargetGroupIDs []string   `json:"target_group_ids,omitempty"`
	IsActive       bool       `json:"is_active"`
	Priority       int        `json:"priority"`
	StartDate      time.Time  `json:"start_date"`
	EndDate        *time.Time `json:"end_date,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type CreateBroadcastRequest struct {
	Title          string   `json:"title" binding:"required"`
	Message        string   `json:"message" binding:"required"`
	TargetType     string   `json:"target_type" binding:"required,oneof=all_groups selected_groups"`
	TargetGroupIDs []string `json:"target_group_ids,omitempty"`
	Priority       int      `json:"priority,omitempty"`
	EndDate        *string  `json:"end_date,omitempty"`
}

type UpdateBroadcastRequest struct {
	Title          string   `json:"title,omitempty"`
	Message        string   `json:"message,omitempty"`
	TargetType     string   `json:"target_type,omitempty"`
	TargetGroupIDs []string `json:"target_group_ids,omitempty"`
	IsActive       *bool    `json:"is_active,omitempty"`
	Priority       *int     `json:"priority,omitempty"`
	EndDate        *string  `json:"end_date,omitempty"`
}

type GetBroadcastsResponse struct {
	Broadcasts []BroadcastResponse `json:"broadcasts"`
	Total      int                 `json:"total"`
}

package models

import (
	"time"
)

type Notification struct {
	ID         string    `json:"id" db:"id"`
	UserID     string    `json:"user_id" db:"user_id"`
	Type       string    `json:"type" db:"type"` // welcome, admin, payment, group, system
	Title      string    `json:"title" db:"title"`
	Message    string    `json:"message" db:"message"`
	IsRead     bool      `json:"is_read" db:"is_read"`
	ActionURL  *string   `json:"action_url,omitempty" db:"action_url"`
	ActionText *string   `json:"action_text,omitempty" db:"action_text"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

type NotificationResponse struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Title      string    `json:"title"`
	Message    string    `json:"message"`
	IsRead     bool      `json:"is_read"`
	ActionURL  *string   `json:"action_url,omitempty"`
	ActionText *string   `json:"action_text,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateNotificationRequest struct {
	UserID     string  `json:"user_id" binding:"required"`
	Type       string  `json:"type" binding:"required"`
	Title      string  `json:"title" binding:"required"`
	Message    string  `json:"message" binding:"required"`
	ActionURL  *string `json:"action_url,omitempty"`
	ActionText *string `json:"action_text,omitempty"`
}

type MarkAsReadRequest struct {
	NotificationID string `json:"notification_id" binding:"required"`
}

type GetNotificationsResponse struct {
	Notifications []NotificationResponse `json:"notifications"`
	UnreadCount   int                    `json:"unread_count"`
	Total         int                    `json:"total"`
}

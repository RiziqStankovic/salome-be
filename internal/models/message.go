package models

import (
	"time"

	"github.com/google/uuid"
)

type GroupMessage struct {
	ID          string       `json:"id" db:"id"`
	GroupID     string       `json:"group_id" db:"group_id"`
	UserID      uuid.UUID    `json:"user_id" db:"user_id"`
	Message     string       `json:"message" db:"message"`
	MessageType string       `json:"message_type" db:"message_type"`
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	User        UserResponse `json:"user,omitempty"`
}

type GroupMessageCreateRequest struct {
	Message     string `json:"message" binding:"required"`
	MessageType string `json:"message_type"`
}

type GroupMessageResponse struct {
	ID          string       `json:"id"`
	GroupID     string       `json:"group_id"`
	UserID      uuid.UUID    `json:"user_id"`
	Message     string       `json:"message"`
	MessageType string       `json:"message_type"`
	CreatedAt   time.Time    `json:"created_at"`
	User        UserResponse `json:"user,omitempty"`
}

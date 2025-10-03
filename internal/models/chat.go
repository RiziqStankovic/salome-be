package models

import (
	"time"

	"github.com/google/uuid"
)

type Chat struct {
	ID            uuid.UUID  `json:"id"`
	UserID        *uuid.UUID `json:"user_id"` // Nullable for anonymous chats
	AnonymousName *string    `json:"anonymous_name"`
	Status        string     `json:"status"`
	IsRead        bool       `json:"is_read"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type Message struct {
	ID         uuid.UUID  `json:"id"`
	ChatID     uuid.UUID  `json:"chat_id"`
	SenderID   *uuid.UUID `json:"sender_id"`   // Nullable for anonymous or admin messages
	SenderType string     `json:"sender_type"` // 'user', 'anonymous', 'admin'
	Content    string     `json:"content"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ChatCreateRequest for creating a new chat (from user or anonymous)
type ChatCreateRequest struct {
	AnonymousName *string `json:"anonymous_name"` // Required if UserID is not provided
	Content       string  `json:"content" binding:"required"`
}

// MessageSendRequest for sending a message within an existing chat
type MessageSendRequest struct {
	Content string `json:"content" binding:"required"`
}

// ChatResponse for returning chat details
type ChatResponse struct {
	ID            uuid.UUID        `json:"id"`
	UserID        *uuid.UUID       `json:"user_id"`
	AnonymousName *string          `json:"anonymous_name"`
	Status        string           `json:"status"`
	IsRead        bool             `json:"is_read"`
	MessageCount  int              `json:"message_count"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	LastMessage   *MessageResponse `json:"last_message,omitempty"`
	SenderName    *string          `json:"sender_name,omitempty"` // For display purposes
}

// MessageResponse for returning message details
type MessageResponse struct {
	ID         uuid.UUID  `json:"id"`
	ChatID     uuid.UUID  `json:"chat_id"`
	SenderID   *uuid.UUID `json:"sender_id"`
	SenderType string     `json:"sender_type"`
	Content    string     `json:"content"`
	CreatedAt  time.Time  `json:"created_at"`
	SenderName *string    `json:"sender_name,omitempty"` // For display purposes
}

package models

import (
	"time"

	"github.com/google/uuid"
)

type EmailSubmission struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	GroupID     string     `json:"group_id" db:"group_id"`
	AppID       string     `json:"app_id" db:"app_id"`
	Email       string     `json:"email" db:"email"`
	Username    *string    `json:"username" db:"username"`
	FullName    string     `json:"full_name" db:"full_name"`
	Status      string     `json:"status" db:"status"` // pending, approved, rejected
	SubmittedAt time.Time  `json:"submitted_at" db:"submitted_at"`
	ReviewedAt  *time.Time `json:"reviewed_at" db:"reviewed_at"`
	ReviewedBy  *uuid.UUID `json:"reviewed_by" db:"reviewed_by"`
	Notes       *string    `json:"notes" db:"notes"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

type EmailSubmissionRequest struct {
	GroupID  string  `json:"group_id" binding:"required"`
	AppID    string  `json:"app_id" binding:"required"`
	Email    string  `json:"email" binding:"required,email"`
	Username *string `json:"username"`
	FullName string  `json:"full_name" binding:"required"`
}

type EmailSubmissionResponse struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	GroupID     string     `json:"group_id"`
	AppID       string     `json:"app_id"`
	Email       string     `json:"email"`
	Username    *string    `json:"username"`
	FullName    string     `json:"full_name"`
	Status      string     `json:"status"`
	SubmittedAt time.Time  `json:"submitted_at"`
	ReviewedAt  *time.Time `json:"reviewed_at"`
	ReviewedBy  *uuid.UUID `json:"reviewed_by"`
	Notes       *string    `json:"notes"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	User        *User      `json:"user,omitempty"`
	Group       *Group     `json:"group,omitempty"`
	App         *App       `json:"app,omitempty"`
}

type EmailSubmissionStats struct {
	Total    int `json:"total"`
	Pending  int `json:"pending"`
	Approved int `json:"approved"`
	Rejected int `json:"rejected"`
}

type EmailSubmissionStatusUpdate struct {
	Status string  `json:"status" binding:"required,oneof=approved rejected"`
	Notes  *string `json:"notes"`
}

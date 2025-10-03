package models

import (
	"time"

	"github.com/google/uuid"
)

type UserAppCredentials struct {
	ID          uuid.UUID `json:"id" db:"id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	GroupID     uuid.UUID `json:"group_id" db:"group_id"`
	Username    string    `json:"username" db:"username"`
	Email       string    `json:"email" db:"email"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type UserAppCredentialsRequest struct {
	GroupID     string `json:"group_id" binding:"required"`
	Username    string `json:"username" binding:"required"`
	Email       string `json:"email" binding:"required"`
	Description string `json:"description" binding:"required"`
}

type UserAppCredentialsResponse struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	GroupID     uuid.UUID `json:"group_id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Group       *Group    `json:"group,omitempty"`
}

package models

import (
	"time"

	"github.com/google/uuid"
)

type UserAppCredentials struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	AppID     string    `json:"app_id" db:"app_id"`
	Username  *string   `json:"username" db:"username"`
	Email     *string   `json:"email" db:"email"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type UserAppCredentialsRequest struct {
	AppID    string  `json:"app_id" binding:"required"`
	Username *string `json:"username"`
	Email    *string `json:"email"`
}

type UserAppCredentialsResponse struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	AppID     string    `json:"app_id"`
	Username  *string   `json:"username"`
	Email     *string   `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	App       *App      `json:"app,omitempty"`
}

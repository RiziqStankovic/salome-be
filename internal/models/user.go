package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID             uuid.UUID `json:"id" db:"id"`
	Email          string    `json:"email" db:"email"`
	PasswordHash   string    `json:"-" db:"password_hash"`
	FullName       string    `json:"full_name" db:"full_name"`
	WhatsappNumber *string   `json:"whatsapp_number" db:"whatsapp_number"`
	AvatarURL      *string   `json:"avatar_url" db:"avatar_url"`
	Status         string    `json:"status" db:"status"`
	Balance        float64   `json:"balance" db:"balance"`
	TotalSpent     float64   `json:"total_spent" db:"total_spent"`
	IsAdmin        bool      `json:"is_admin" db:"is_admin"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

type UserCreateRequest struct {
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required,min=6"`
	FullName       string `json:"full_name" binding:"required"`
	WhatsappNumber string `json:"whatsapp_number" binding:"required"`
}

type UserLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type UserResponse struct {
	ID             uuid.UUID `json:"id"`
	Email          string    `json:"email"`
	FullName       string    `json:"full_name"`
	WhatsappNumber *string   `json:"whatsapp_number"`
	AvatarURL      *string   `json:"avatar_url"`
	Status         string    `json:"status"`
	Balance        float64   `json:"balance"`
	TotalSpent     float64   `json:"total_spent"`
	IsAdmin        bool      `json:"is_admin"`
	CreatedAt      time.Time `json:"created_at"`
}

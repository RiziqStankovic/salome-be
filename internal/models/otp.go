package models

import (
	"time"

	"github.com/google/uuid"
)

type OTP struct {
	ID               string     `json:"id" db:"id"`
	UserID           uuid.UUID  `json:"user_id" db:"user_id"`
	Email            string     `json:"email" db:"email"`
	OTPCode          string     `json:"otp_code" db:"otp_code"`
	Purpose          string     `json:"purpose" db:"purpose"`
	ExpiresAt        time.Time  `json:"expires_at" db:"expires_at"`
	IsUsed           bool       `json:"is_used" db:"is_used"`
	Attempts         int        `json:"attempts" db:"attempts"`
	RateLimitCount   int        `json:"rate_limit_count" db:"rate_limit_count"`
	RateLimitResetAt *time.Time `json:"rate_limit_reset_at" db:"rate_limit_reset_at"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}

type OTPGenerateRequest struct {
	Email   string `json:"email" binding:"required,email"`
	Purpose string `json:"purpose" binding:"required,oneof=email_verification password_reset login_verification"`
}

type OTPVerifyRequest struct {
	Email   string `json:"email" binding:"required,email"`
	OTPCode string `json:"otp_code" binding:"required,len=6"`
	Purpose string `json:"purpose" binding:"required,oneof=email_verification password_reset login_verification"`
}

type OTPResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Purpose   string    `json:"purpose"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type OTPVerifyResponse struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
}

type OTPData struct {
	Email     string `json:"email"`
	OTPCode   string `json:"otp_code"`
	ExpiresIn int    `json:"expires_in"` // in minutes
}

package models

import (
	"time"

	"github.com/google/uuid"
)

type Subscription struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	GroupID         uuid.UUID  `json:"group_id" db:"group_id"`
	ServiceName     string     `json:"service_name" db:"service_name"`
	ServiceURL      *string    `json:"service_url" db:"service_url"`
	PlanName        string     `json:"plan_name" db:"plan_name"`
	PricePerMonth   float64    `json:"price_per_month" db:"price_per_month"`
	Currency        string     `json:"currency" db:"currency"`
	Status          string     `json:"status" db:"status"`
	NextBillingDate *time.Time `json:"next_billing_date" db:"next_billing_date"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

type SubscriptionCreateRequest struct {
	ServiceName   string  `json:"service_name" binding:"required"`
	ServiceURL    string  `json:"service_url"`
	PlanName      string  `json:"plan_name" binding:"required"`
	PricePerMonth float64 `json:"price_per_month" binding:"required,min=0"`
	Currency      string  `json:"currency" binding:"required"`
}

type AccountCredentials struct {
	ID                uuid.UUID              `json:"id" db:"id"`
	SubscriptionID    uuid.UUID              `json:"subscription_id" db:"subscription_id"`
	Username          *string                `json:"username" db:"username"`
	PasswordEncrypted *string                `json:"-" db:"password_encrypted"`
	AdditionalInfo    map[string]interface{} `json:"additional_info" db:"additional_info"`
	CreatedAt         time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at" db:"updated_at"`
}

type AccountCredentialsRequest struct {
	Username       string                 `json:"username"`
	Password       string                 `json:"password"`
	AdditionalInfo map[string]interface{} `json:"additional_info"`
}

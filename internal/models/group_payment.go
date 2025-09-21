package models

import (
	"time"

	"github.com/google/uuid"
)

type GroupPayment struct {
	ID                  uuid.UUID `json:"id" db:"id"`
	GroupID             uuid.UUID `json:"group_id" db:"group_id"`
	TotalCollected      float64   `json:"total_collected" db:"total_collected"`
	TotalRequired       float64   `json:"total_required" db:"total_required"`
	PaymentStatus       string    `json:"payment_status" db:"payment_status"`
	ProviderPurchaseID  *string   `json:"provider_purchase_id" db:"provider_purchase_id"`
	ProviderCredentials *string   `json:"provider_credentials" db:"provider_credentials"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

type GroupPaymentResponse struct {
	ID                  uuid.UUID `json:"id"`
	GroupID             uuid.UUID `json:"group_id"`
	TotalCollected      float64   `json:"total_collected"`
	TotalRequired       float64   `json:"total_required"`
	PaymentStatus       string    `json:"payment_status"`
	ProviderPurchaseID  *string   `json:"provider_purchase_id"`
	ProviderCredentials *string   `json:"provider_credentials"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

package models

import (
	"time"

	"github.com/google/uuid"
)

type Payment struct {
	ID                    uuid.UUID    `json:"id" db:"id"`
	SubscriptionID        uuid.UUID    `json:"subscription_id" db:"subscription_id"`
	UserID                uuid.UUID    `json:"user_id" db:"user_id"`
	Amount                float64      `json:"amount" db:"amount"`
	Currency              string       `json:"currency" db:"currency"`
	Status                string       `json:"status" db:"status"`
	MidtransTransactionID *string      `json:"midtrans_transaction_id" db:"midtrans_transaction_id"`
	PaymentMethod         *string      `json:"payment_method" db:"payment_method"`
	CreatedAt             time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time    `json:"updated_at" db:"updated_at"`
	User                  UserResponse `json:"user,omitempty"`
}

type PaymentRequest struct {
	SubscriptionID uuid.UUID `json:"subscription_id" binding:"required"`
	Amount         float64   `json:"amount" binding:"required,min=0"`
	PaymentMethod  string    `json:"payment_method" binding:"required"`
}

type MidtransResponse struct {
	Token       string `json:"token"`
	RedirectURL string `json:"redirect_url"`
}

type PaymentNotification struct {
	TransactionStatus string `json:"transaction_status"`
	OrderID           string `json:"order_id"`
	PaymentType       string `json:"payment_type"`
	TransactionTime   string `json:"transaction_time"`
	GrossAmount       string `json:"gross_amount"`
}

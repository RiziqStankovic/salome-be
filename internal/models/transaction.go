package models

import (
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	UserID           uuid.UUID  `json:"user_id" db:"user_id"`
	GroupID          *uuid.UUID `json:"group_id" db:"group_id"`
	Type             string     `json:"type" db:"type"`
	Amount           float64    `json:"amount" db:"amount"`
	BalanceBefore    float64    `json:"balance_before" db:"balance_before"`
	BalanceAfter     float64    `json:"balance_after" db:"balance_after"`
	Description      string     `json:"description" db:"description"`
	PaymentMethod    *string    `json:"payment_method" db:"payment_method"`
	PaymentReference *string    `json:"payment_reference" db:"payment_reference"`
	Status           string     `json:"status" db:"status"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

type TransactionCreateRequest struct {
	GroupID          *uuid.UUID `json:"group_id"`
	Type             string     `json:"type" binding:"required"`
	Amount           float64    `json:"amount" binding:"required"`
	Description      string     `json:"description" binding:"required"`
	PaymentMethod    string     `json:"payment_method"`
	PaymentReference string     `json:"payment_reference"`
}

type TransactionResponse struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	GroupID          *uuid.UUID `json:"group_id"`
	Type             string     `json:"type"`
	Amount           float64    `json:"amount"`
	BalanceBefore    float64    `json:"balance_before"`
	BalanceAfter     float64    `json:"balance_after"`
	Description      string     `json:"description"`
	PaymentMethod    *string    `json:"payment_method"`
	PaymentReference *string    `json:"payment_reference"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

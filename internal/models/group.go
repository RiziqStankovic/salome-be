package models

import (
	"time"

	"github.com/google/uuid"
)

type Group struct {
	ID             uuid.UUID     `json:"id" db:"id"`
	Name           string        `json:"name" db:"name"`
	Description    *string       `json:"description" db:"description"`
	AppID          string        `json:"app_id" db:"app_id"`
	MaxMembers     int           `json:"max_members" db:"max_members"`
	CurrentMembers int           `json:"current_members" db:"current_members"`
	PricePerMember float64       `json:"price_per_member" db:"price_per_member"`
	AdminFee       float64       `json:"admin_fee" db:"admin_fee"`
	TotalPrice     float64       `json:"total_price" db:"total_price"`
	Status         string        `json:"status" db:"status"`
	GroupStatus    string        `json:"group_status" db:"group_status"`
	InviteCode     string        `json:"invite_code" db:"invite_code"`
	OwnerID        uuid.UUID     `json:"owner_id" db:"owner_id"`
	ExpiresAt      *time.Time    `json:"expires_at" db:"expires_at"`
	AllPaidAt      *time.Time    `json:"all_paid_at" db:"all_paid_at"`
	CreatedAt      time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at" db:"updated_at"`
	Members        []GroupMember `json:"members,omitempty"`
	App            *App          `json:"app,omitempty"`
}

type GroupMember struct {
	ID                      string       `json:"id" db:"id"`
	GroupID                 string       `json:"group_id" db:"group_id"`
	UserID                  uuid.UUID    `json:"user_id" db:"user_id"`
	JoinedAt                time.Time    `json:"joined_at" db:"joined_at"`
	Status                  string       `json:"status" db:"status"`
	UserStatus              string       `json:"user_status" db:"user_status"`
	PaymentAmount           int          `json:"payment_amount" db:"payment_amount"`
	PricePerMember          float64      `json:"price_per_member" db:"price_per_member"`
	PaymentDeadline         *time.Time   `json:"payment_deadline" db:"payment_deadline"`
	PaidAt                  *time.Time   `json:"paid_at" db:"paid_at"`
	ActivatedAt             *time.Time   `json:"activated_at" db:"activated_at"`
	ExpiredAt               *time.Time   `json:"expired_at" db:"expired_at"`
	RemovedAt               *time.Time   `json:"removed_at" db:"removed_at"`
	RemovedReason           *string      `json:"removed_reason" db:"removed_reason"`
	SubscriptionPeriodStart *time.Time   `json:"subscription_period_start" db:"subscription_period_start"`
	SubscriptionPeriodEnd   *time.Time   `json:"subscription_period_end" db:"subscription_period_end"`
	User                    UserResponse `json:"user,omitempty"`
}

type GroupCreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	AppID       string `json:"app_id" binding:"required"`
	MaxMembers  int    `json:"max_members" binding:"min=2,max=50"`
}

type GroupJoinRequest struct {
	InviteCode string `json:"invite_code" binding:"required"`
}

type GroupResponse struct {
	ID             uuid.UUID     `json:"id"`
	Name           string        `json:"name"`
	Description    *string       `json:"description"`
	AppID          string        `json:"app_id"`
	MaxMembers     int           `json:"max_members"`
	CurrentMembers int           `json:"current_members"`
	MemberCount    int           `json:"member_count"`
	PricePerMember float64       `json:"price_per_member"`
	AdminFee       float64       `json:"admin_fee"`
	TotalPrice     float64       `json:"total_price"`
	Status         string        `json:"status"`
	GroupStatus    string        `json:"group_status"`
	InviteCode     string        `json:"invite_code"`
	OwnerID        uuid.UUID     `json:"owner_id"`
	ExpiresAt      *time.Time    `json:"expires_at"`
	AllPaidAt      *time.Time    `json:"all_paid_at"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	Members        []GroupMember `json:"members,omitempty"`
	App            *App          `json:"app,omitempty"`
	Owner          *UserResponse `json:"owner,omitempty"`
}

// State Machine Constants
const (
	// User Status Constants
	UserStatusPending = "pending"
	UserStatusPaid    = "paid"
	UserStatusActive  = "active"
	UserStatusExpired = "expired"
	UserStatusRemoved = "removed"

	// Group Status Constants
	GroupStatusOpen      = "open"
	GroupStatusPrivate   = "private"
	GroupStatusFull      = "full"
	GroupStatusPaidGroup = "paid_group"
	GroupStatusClosed    = "closed"

	// Payment timeout (24 hours)
	PaymentTimeoutHours = 24
)

// Admin Request/Response
type AdminUserStatusRequest struct {
	UserID        string `json:"user_id" binding:"required"`
	GroupID       string `json:"group_id" binding:"required"`
	NewStatus     string `json:"new_status" binding:"required"`
	RemovedReason string `json:"removed_reason,omitempty"`
}

type AdminGroupStatusRequest struct {
	GroupID   string `json:"group_id" binding:"required"`
	NewStatus string `json:"new_status" binding:"required"`
}

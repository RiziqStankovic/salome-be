package models

import "time"

type PaidGroupWithCredentials struct {
	ID                 string                      `json:"id"`
	Name               string                      `json:"name"`
	Description        string                      `json:"description"`
	AppID              string                      `json:"app_id"`
	MaxMembers         int                         `json:"max_members"`
	MemberCount        int                         `json:"member_count"`
	PricePerMember     int                         `json:"price_per_member"`
	TotalPrice         int                         `json:"total_price"`
	GroupStatus        string                      `json:"group_status"`
	AllPaidAt          time.Time                   `json:"all_paid_at"`
	CreatedAt          time.Time                   `json:"created_at"`
	App                App                         `json:"app"`
	Members            []GroupMemberWithUser       `json:"members"`
	AccountCredentials []AccountCredentialWithUser `json:"account_credentials"`
}

type GroupMemberWithUser struct {
	ID         string    `json:"id"`
	GroupID    string    `json:"group_id"`
	UserID     string    `json:"user_id"`
	UserStatus string    `json:"user_status"`
	JoinedAt   time.Time `json:"joined_at"`
	User       User      `json:"user"`
}

type AccountCredentialWithUser struct {
	ID          string    `json:"id"`
	GroupID     string    `json:"group_id"`
	UserID      string    `json:"user_id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	User        User      `json:"user"`
}

type GetPaidGroupsResponse struct {
	Groups   []PaidGroupWithCredentials `json:"groups"`
	Total    int                        `json:"total"`
	Page     int                        `json:"page"`
	PageSize int                        `json:"page_size"`
}

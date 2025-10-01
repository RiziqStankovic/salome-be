package models

import (
	"time"
)

type App struct {
	ID                 string    `json:"id" db:"id"`
	Name               string    `json:"name" db:"name"`
	Description        string    `json:"description" db:"description"`
	Category           string    `json:"category" db:"category"`
	IconURL            string    `json:"icon_url" db:"icon_url"`
	WebsiteURL         string    `json:"website_url" db:"website_url"`
	MaxGroupMembers    int       `json:"max_group_members" db:"max_group_members"`
	TotalPrice         float64   `json:"total_price" db:"total_price"`
	IsPopular          bool      `json:"is_popular" db:"is_popular"`
	IsActive           bool      `json:"is_active" db:"is_active"`
	AdminFeePercentage float64   `json:"admin_fee_percentage" db:"admin_fee_percentage"`
	HowItWorks         *string   `json:"how_it_works" db:"how_it_works"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`

	// Admin view fields
	GroupsCount  int     `json:"groups_count,omitempty"`
	TotalRevenue float64 `json:"total_revenue,omitempty"`
	AvgPrice     float64 `json:"avg_price,omitempty"`
}

type AppResponse struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Category        string  `json:"category"`
	IconURL         string  `json:"icon_url"`
	WebsiteURL      string  `json:"website_url"`
	MaxGroupMembers int     `json:"max_group_members"`
	TotalPrice      float64 `json:"total_price"`
	PricePerUser    float64 `json:"price_per_user"`
	IsPopular       bool    `json:"is_popular"`
	IsActive        bool    `json:"is_active"`
	HowItWorks      *string `json:"how_it_works"`
}

type AppListResponse struct {
	Apps       []AppResponse `json:"apps"`
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
}

type AppSearchRequest struct {
	Query    string `json:"query"`
	Category string `json:"category"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

type AppDetailResponse struct {
	App          App     `json:"app"`
	PricePerUser float64 `json:"price_per_user"`
}

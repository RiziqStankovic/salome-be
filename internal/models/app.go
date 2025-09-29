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
	TotalMembers       int       `json:"total_members" db:"total_members"`
	TotalPrice         int       `json:"total_price" db:"total_price"`
	IsPopular          bool      `json:"is_popular" db:"is_popular"`
	IsActive           bool      `json:"is_active" db:"is_active"`
	IsAvailable        bool      `json:"is_available" db:"is_available"`
	MaxGroupMembers    int       `json:"max_group_members" db:"max_group_members"`
	BasePrice          float64   `json:"base_price" db:"base_price"`
	AdminFeePercentage float64   `json:"admin_fee_percentage" db:"admin_fee_percentage"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

type AppResponse struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Category        string  `json:"category"`
	IconURL         string  `json:"icon_url"`
	WebsiteURL      string  `json:"website_url"`
	TotalMembers    int     `json:"total_members"`
	TotalPrice      int     `json:"total_price"`
	PricePerUser    float64 `json:"price_per_user"`
	IsPopular       bool    `json:"is_popular"`
	IsActive        bool    `json:"is_active"`
	IsAvailable     bool    `json:"is_available"`
	MaxGroupMembers int     `json:"max_group_members"`
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

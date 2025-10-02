package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"salome-be/internal/models"

	"github.com/gin-gonic/gin"
)

type AppHandler struct {
	db *sql.DB
}

func NewAppHandler(db *sql.DB) *AppHandler {
	return &AppHandler{db: db}
}

func (h *AppHandler) GetApps(c *gin.Context) {
	// Parse query parameters
	page := 1
	pageSize := 20
	category := c.Query("category")
	query := c.Query("q")
	popular := c.Query("popular")

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	offset := (page - 1) * pageSize

	// Build query
	whereClause := "WHERE is_active = true"
	args := []interface{}{}
	argIndex := 1

	if category != "" {
		whereClause += " AND category = $" + strconv.Itoa(argIndex)
		args = append(args, category)
		argIndex++
	}

	if query != "" {
		whereClause += " AND (name ILIKE $" + strconv.Itoa(argIndex) + " OR description ILIKE $" + strconv.Itoa(argIndex+1) + ")"
		args = append(args, "%"+query+"%", "%"+query+"%")
		argIndex += 2
	}

	if popular == "true" {
		whereClause += " AND is_popular = true"
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM apps " + whereClause
	err := h.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count apps"})
		return
	}

	// Get apps
	orderBy := "ORDER BY is_popular DESC, name ASC"
	if query != "" {
		orderBy = "ORDER BY is_popular DESC, name ILIKE $" + strconv.Itoa(argIndex) + " DESC, name ASC"
		args = append(args, query+"%")
		argIndex++
	}

	appsQuery := `
		SELECT id, name, description, category, icon_url, website_url, max_group_members, total_price, is_popular, is_active, how_it_works
		FROM apps 
		` + whereClause + `
		` + orderBy + `
		LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)

	args = append(args, pageSize, offset)

	// fmt.Printf("Executing query: %s\n", appsQuery)
	// fmt.Printf("With args: %v\n", args)

	rows, err := h.db.Query(appsQuery, args...)
	if err != nil {
		fmt.Printf("Query error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch apps", "details": err.Error()})
		return
	}
	defer rows.Close()

	var apps []models.AppResponse
	for rows.Next() {
		var app models.AppResponse
		err := rows.Scan(&app.ID, &app.Name, &app.Description, &app.Category, &app.IconURL, &app.WebsiteURL, &app.MaxGroupMembers, &app.TotalPrice, &app.IsPopular, &app.IsActive, &app.HowItWorks)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan app"})
			return
		}

		// Calculate price per user: (total_price / max_group_members) + 3500 (admin fee)
		if app.MaxGroupMembers > 0 {
			app.PricePerUser = app.TotalPrice/float64(app.MaxGroupMembers) + 3500
		} else {
			app.PricePerUser = app.TotalPrice + 3500
		}

		apps = append(apps, app)
	}

	totalPages := (total + pageSize - 1) / pageSize

	response := models.AppListResponse{
		Apps:       apps,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	c.JSON(http.StatusOK, response)
}

func (h *AppHandler) GetAppCategories(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT DISTINCT category 
		FROM apps 
		WHERE category IS NOT NULL AND category != ''
		ORDER BY category
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		err := rows.Scan(&category)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan category"})
			return
		}
		categories = append(categories, category)
	}

	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

func (h *AppHandler) GetPopularApps(c *gin.Context) {
	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	rows, err := h.db.Query(`
		SELECT id, name, description, category, icon_url, website_url, max_group_members, total_price, is_popular, is_active, how_it_works
		FROM apps 
		WHERE is_popular = true
		ORDER BY name ASC
		LIMIT $1
	`, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch popular apps"})
		return
	}
	defer rows.Close()

	var apps []models.AppResponse
	for rows.Next() {
		var app models.AppResponse
		err := rows.Scan(&app.ID, &app.Name, &app.Description, &app.Category, &app.IconURL, &app.WebsiteURL, &app.MaxGroupMembers, &app.TotalPrice, &app.IsPopular, &app.IsActive, &app.HowItWorks)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan app"})
			return
		}

		// Calculate price per user: (total_price / max_group_members) + 3500 (admin fee)
		if app.MaxGroupMembers > 0 {
			app.PricePerUser = app.TotalPrice/float64(app.MaxGroupMembers) + 3500
		} else {
			app.PricePerUser = app.TotalPrice + 3500
		}

		apps = append(apps, app)
	}

	c.JSON(http.StatusOK, gin.H{"apps": apps})
}

func (h *AppHandler) GetAppByID(c *gin.Context) {
	appID := c.Param("id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "App ID is required"})
		return
	}

	var app models.App
	err := h.db.QueryRow(`
		SELECT id, name, description, category, icon_url, website_url, max_group_members, total_price, is_popular, is_active, admin_fee_percentage, how_it_works
		FROM apps 
		WHERE id = $1 AND is_active = true
	`, appID).Scan(
		&app.ID, &app.Name, &app.Description, &app.Category, &app.IconURL, &app.WebsiteURL,
		&app.MaxGroupMembers, &app.TotalPrice, &app.IsPopular, &app.IsActive,
		&app.AdminFeePercentage, &app.HowItWorks,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch app"})
		return
	}

	// Calculate pricing details
	var pricePerUser float64
	if app.MaxGroupMembers > 0 {
		pricePerUser = app.TotalPrice/float64(app.MaxGroupMembers) + 3500
	} else {
		pricePerUser = app.TotalPrice + 3500
	}

	response := models.AppDetailResponse{
		App:          app,
		PricePerUser: pricePerUser,
	}

	c.JSON(http.StatusOK, response)
}

func (h *AppHandler) SeedApps(c *gin.Context) {
	// This is a development endpoint to seed apps data
	apps := []models.App{
		{
			ID:              "netflix",
			Name:            "Netflix",
			Description:     "Streaming film dan serial TV terpopuler",
			Category:        "Entertainment",
			IconURL:         "https://cdn.jsdelivr.net/gh/simple-icons/simple-icons@develop/icons/netflix.svg",
			WebsiteURL:      "https://netflix.com",
			MaxGroupMembers: 4,
			TotalPrice:      186000,
			IsPopular:       true,
		},
		{
			ID:              "spotify",
			Name:            "Spotify",
			Description:     "Platform musik streaming terbesar di dunia",
			Category:        "Music",
			IconURL:         "https://cdn.jsdelivr.net/gh/simple-icons/simple-icons@develop/icons/spotify.svg",
			WebsiteURL:      "https://spotify.com",
			MaxGroupMembers: 6,
			TotalPrice:      78000,
			IsPopular:       true,
		},
		{
			ID:              "youtube-premium",
			Name:            "YouTube Premium",
			Description:     "YouTube tanpa iklan dengan fitur tambahan",
			Category:        "Entertainment",
			IconURL:         "https://cdn.jsdelivr.net/gh/simple-icons/simple-icons@develop/icons/youtube.svg",
			WebsiteURL:      "https://youtube.com/premium",
			MaxGroupMembers: 6,
			TotalPrice:      139000,
			IsPopular:       true,
		},
		{
			ID:              "adobe-creative-cloud",
			Name:            "Adobe Creative Cloud",
			Description:     "Suite aplikasi kreatif untuk desain dan editing",
			Category:        "Productivity",
			IconURL:         "https://cdn.jsdelivr.net/gh/simple-icons/simple-icons@develop/icons/adobe.svg",
			WebsiteURL:      "https://adobe.com/creativecloud.html",
			MaxGroupMembers: 2,
			TotalPrice:      680000,
			IsPopular:       true,
		},
		{
			ID:              "microsoft-365",
			Name:            "Microsoft 365",
			Description:     "Office suite lengkap dengan cloud storage",
			Category:        "Productivity",
			IconURL:         "https://cdn.jsdelivr.net/gh/simple-icons/simple-icons@develop/icons/microsoft.svg",
			WebsiteURL:      "https://microsoft.com/microsoft-365",
			MaxGroupMembers: 6,
			TotalPrice:      120000,
			IsPopular:       true,
		},
		{
			ID:              "canva-pro",
			Name:            "Canva Pro",
			Description:     "Platform desain grafis online",
			Category:        "Design",
			IconURL:         "https://cdn.jsdelivr.net/gh/simple-icons/simple-icons@develop/icons/canva.svg",
			WebsiteURL:      "https://canva.com",
			MaxGroupMembers: 5,
			TotalPrice:      150000,
			IsPopular:       true,
		},
		{
			ID:              "notion",
			Name:            "Notion",
			Description:     "All-in-one workspace untuk notes dan project management",
			Category:        "Productivity",
			IconURL:         "https://cdn.jsdelivr.net/gh/simple-icons/simple-icons@develop/icons/notion.svg",
			WebsiteURL:      "https://notion.so",
			MaxGroupMembers: 4,
			TotalPrice:      80000,
			IsPopular:       true,
		},
		{
			ID:              "figma",
			Name:            "Figma",
			Description:     "Collaborative design tool untuk UI/UX",
			Category:        "Design",
			IconURL:         "https://cdn.jsdelivr.net/gh/simple-icons/simple-icons@develop/icons/figma.svg",
			WebsiteURL:      "https://figma.com",
			MaxGroupMembers: 3,
			TotalPrice:      150000,
			IsPopular:       true,
		},
		{
			ID:              "discord-nitro",
			Name:            "Discord Nitro",
			Description:     "Premium features untuk Discord",
			Category:        "Communication",
			IconURL:         "https://cdn.jsdelivr.net/gh/simple-icons/simple-icons@develop/icons/discord.svg",
			WebsiteURL:      "https://discord.com/nitro",
			MaxGroupMembers: 2,
			TotalPrice:      100000,
			IsPopular:       false,
		},
		{
			ID:              "github-copilot",
			Name:            "GitHub Copilot",
			Description:     "AI pair programmer untuk coding",
			Category:        "Development",
			IconURL:         "https://cdn.jsdelivr.net/gh/simple-icons/simple-icons@develop/icons/github.svg",
			WebsiteURL:      "https://github.com/features/copilot",
			MaxGroupMembers: 1,
			TotalPrice:      100000,
			IsPopular:       false,
		},
	}

	// Clear existing apps
	_, err := h.db.Exec("DELETE FROM apps")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear existing apps"})
		return
	}

	// Insert new apps
	for _, app := range apps {
		_, err := h.db.Exec(`
			INSERT INTO apps (id, name, description, category, icon_url, website_url, max_group_members, total_price, is_popular, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		`, app.ID, app.Name, app.Description, app.Category, app.IconURL, app.WebsiteURL, app.MaxGroupMembers, app.TotalPrice, app.IsPopular)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert app: " + app.Name})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Apps seeded successfully",
		"count":   len(apps),
	})
}

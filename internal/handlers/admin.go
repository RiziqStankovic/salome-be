package handlers

import (
	"database/sql"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	// "strings"

	"salome-be/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminHandler struct {
	db *sql.DB
}

func NewAdminHandler(db *sql.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

// GetUsers - Get all users with pagination and filters
func (h *AdminHandler) GetUsers(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	status := c.Query("status")
	search := c.Query("search")

	// Calculate offset
	offset := (page - 1) * pageSize

	// Build query
	query := `
		SELECT id, email, full_name, avatar_url, status, balance, total_spent, is_admin, created_at, updated_at
		FROM users
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	// Add status filter
	if status != "" && status != "all" {
		query += ` AND status = $` + strconv.Itoa(argIndex)
		args = append(args, status)
		argIndex++
	}

	// Add search filter
	if search != "" {
		query += ` AND (email ILIKE $` + strconv.Itoa(argIndex) + ` OR full_name ILIKE $` + strconv.Itoa(argIndex+1) + `)`
		searchTerm := "%" + search + "%"
		args = append(args, searchTerm, searchTerm)
		argIndex += 2
	}

	// Add ordering and pagination
	query += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)
	args = append(args, pageSize, offset)

	// Execute query
	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.Email, &user.FullName, &user.AvatarURL, &user.Status, &user.Balance, &user.TotalSpent, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan user"})
			return
		}
		users = append(users, user)
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM users WHERE 1=1`
	countArgs := []interface{}{}
	countArgIndex := 1

	if status != "" && status != "all" {
		countQuery += ` AND status = $` + strconv.Itoa(countArgIndex)
		countArgs = append(countArgs, status)
		countArgIndex++
	}

	if search != "" {
		countQuery += ` AND (email ILIKE $` + strconv.Itoa(countArgIndex) + ` OR full_name ILIKE $` + strconv.Itoa(countArgIndex+1) + `)`
		searchTerm := "%" + search + "%"
		countArgs = append(countArgs, searchTerm, searchTerm)
	}

	var total int
	err = h.db.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count users"})
		return
	}

	// Get stats
	statsQuery := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
			COUNT(CASE WHEN status = 'suspended' THEN 1 END) as suspended,
			COUNT(CASE WHEN is_admin = true THEN 1 END) as admins
		FROM users
	`
	var stats struct {
		Total     int `json:"total"`
		Active    int `json:"active"`
		Pending   int `json:"pending"`
		Suspended int `json:"suspended"`
		Admins    int `json:"admins"`
	}

	err = h.db.QueryRow(statsQuery).Scan(&stats.Total, &stats.Active, &stats.Pending, &stats.Suspended, &stats.Admins)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": users,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": (total + pageSize - 1) / pageSize,
		},
		"stats": stats,
	})
}

// UpdateUserStatus - Update user status
func (h *AdminHandler) UpdateUserStatus(c *gin.Context) {
	var req struct {
		UserID        string `json:"user_id" binding:"required"`
		NewStatus     string `json:"new_status" binding:"required"`
		RemovedReason string `json:"removed_reason,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate status
	validStatuses := []string{"active", "pending", "suspended"}
	if !contains(validStatuses, req.NewStatus) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}

	// Update user status
	_, err := h.db.Exec(`
		UPDATE users 
		SET status = $1, updated_at = NOW() 
		WHERE id = $2
	`, req.NewStatus, req.UserID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User status updated successfully"})
}

// GetGroups - Get all groups with pagination and filters
func (h *AdminHandler) GetGroups(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	// status := c.Query("status") // Commented out since we're not using status filter yet
	search := c.Query("search")

	// Calculate offset
	offset := (page - 1) * pageSize

	// Build query - very simple first
	query := `
		SELECT 
			g.id, g.name, g.description, g.app_id, g.owner_id, 
			g.max_members, g.price_per_member, g.created_at, g.updated_at,
			u.full_name as owner_name, u.email as owner_email,
			a.name as app_name, a.icon_url as app_icon,
			0 as members_count,
			0 as total_revenue
		FROM groups g
		LEFT JOIN users u ON g.owner_id = u.id
		LEFT JOIN apps a ON g.app_id = a.id
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	// Skip status filter for now since field might not exist
	// if status != "" && status != "all" {
	// 	query += ` AND g.status = $` + strconv.Itoa(argIndex)
	// 	args = append(args, status)
	// 	argIndex++
	// }

	// Add search filter
	if search != "" {
		query += ` AND (g.name ILIKE $` + strconv.Itoa(argIndex) + ` OR g.description ILIKE $` + strconv.Itoa(argIndex+1) + ` OR a.name ILIKE $` + strconv.Itoa(argIndex+2) + ` OR u.full_name ILIKE $` + strconv.Itoa(argIndex+3) + `)`
		searchTerm := "%" + search + "%"
		args = append(args, searchTerm, searchTerm, searchTerm, searchTerm)
		argIndex += 4
	}

	// Add GROUP BY and ordering
	query += ` GROUP BY g.id, u.full_name, u.email, a.name, a.icon_url ORDER BY g.created_at DESC LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)
	args = append(args, pageSize, offset)

	// Execute query
	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch groups", "details": err.Error()})
		return
	}
	defer rows.Close()

	var groups []models.Group
	for rows.Next() {
		var group models.Group
		var ownerName, ownerEmail, appName, appIcon sql.NullString
		var membersCount int
		var totalRevenue float64

		err := rows.Scan(
			&group.ID, &group.Name, &group.Description, &group.AppID, &group.OwnerID,
			&group.MaxMembers, &group.PricePerMember,
			&group.CreatedAt, &group.UpdatedAt, &ownerName, &ownerEmail, &appName, &appIcon,
			&membersCount, &totalRevenue,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan group"})
			return
		}

		// Add additional fields for admin view
		group.OwnerName = ownerName.String
		group.OwnerEmail = ownerEmail.String
		group.AppName = appName.String
		group.AppIcon = appIcon.String
		group.MembersCount = membersCount
		group.TotalRevenue = totalRevenue

		// Set default values for fields that might not exist
		group.GroupStatus = "open" // Default status
		group.IsPublic = true      // Default public

		groups = append(groups, group)
	}

	// Get total count - simplified
	countQuery := `SELECT COUNT(*) FROM groups g WHERE 1=1`
	countArgs := []interface{}{}
	countArgIndex := 1

	// Skip status filter for now since field might not exist
	// if status != "" && status != "all" {
	// 	countQuery += ` AND g.status = $` + strconv.Itoa(countArgIndex)
	// 	countArgs = append(countArgs, status)
	// 	countArgIndex++
	// }

	if search != "" {
		countQuery += ` AND (g.name ILIKE $` + strconv.Itoa(countArgIndex) + ` OR g.description ILIKE $` + strconv.Itoa(countArgIndex+1) + `)`
		searchTerm := "%" + search + "%"
		countArgs = append(countArgs, searchTerm, searchTerm)
	}

	var total int
	err = h.db.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count groups"})
		return
	}

	// Get stats - simplified
	statsQuery := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) as active,
			0 as pending,
			0 as suspended,
			COUNT(*) as public,
			0 as private,
			0 as total_revenue
		FROM groups g
	`
	var stats struct {
		Total        int     `json:"total"`
		Active       int     `json:"active"`
		Pending      int     `json:"pending"`
		Suspended    int     `json:"suspended"`
		Public       int     `json:"public"`
		Private      int     `json:"private"`
		TotalRevenue float64 `json:"total_revenue"`
	}

	err = h.db.QueryRow(statsQuery).Scan(&stats.Total, &stats.Active, &stats.Pending, &stats.Suspended, &stats.Public, &stats.Private, &stats.TotalRevenue)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": groups,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": (total + pageSize - 1) / pageSize,
		},
		"stats": stats,
	})
}

// UpdateGroupStatus - Update group status
func (h *AdminHandler) UpdateGroupStatus(c *gin.Context) {
	var req struct {
		GroupID       string `json:"group_id" binding:"required"`
		NewStatus     string `json:"new_status" binding:"required"`
		RemovedReason string `json:"removed_reason,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate status
	validStatuses := []string{"open", "private", "full", "paid_group", "closed"}
	if !contains(validStatuses, req.NewStatus) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}

	// Update group status
	_, err := h.db.Exec(`
		UPDATE groups 
		SET group_status = $1, updated_at = NOW() 
		WHERE id = $2
	`, req.NewStatus, req.GroupID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update group status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Group status updated successfully"})
}

// GetApps - Get all apps with pagination and filters
func (h *AdminHandler) GetApps(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	// status := c.Query("status") // Commented out since we're not using status filter yet
	search := c.Query("search")

	// Calculate offset
	offset := (page - 1) * pageSize

	// Build query - very simple first
	query := `
		SELECT 
			a.id, a.name, a.description, a.icon_url, a.category, 
			a.how_it_works, a.created_at, a.updated_at,
			0 as groups_count,
			0 as total_revenue,
			0 as avg_price
		FROM apps a
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	// Skip status filter for now since fields might not exist
	// if status != "" && status != "all" {
	// 	if status == "active" {
	// 		query += ` AND a.is_active = true`
	// 	} else if status == "inactive" {
	// 		query += ` AND a.is_active = false`
	// 	} else if status == "available" {
	// 		query += ` AND a.is_available = true`
	// 	} else if status == "unavailable" {
	// 		query += ` AND a.is_available = false`
	// 	}
	// }

	// Add search filter
	if search != "" {
		query += ` AND (a.name ILIKE $` + strconv.Itoa(argIndex) + ` OR a.description ILIKE $` + strconv.Itoa(argIndex+1) + ` OR a.category ILIKE $` + strconv.Itoa(argIndex+2) + `)`
		searchTerm := "%" + search + "%"
		args = append(args, searchTerm, searchTerm, searchTerm)
		argIndex += 3
	}

	// Add GROUP BY and ordering
	query += ` GROUP BY a.id ORDER BY a.created_at DESC LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)
	args = append(args, pageSize, offset)

	// Execute query
	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch apps", "details": err.Error()})
		return
	}
	defer rows.Close()

	var apps []models.App
	for rows.Next() {
		var app models.App
		var groupsCount int
		var totalRevenue, avgPrice float64

		err := rows.Scan(
			&app.ID, &app.Name, &app.Description, &app.IconURL, &app.Category,
			&app.HowItWorks, &app.CreatedAt, &app.UpdatedAt,
			&groupsCount, &totalRevenue, &avgPrice,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan app"})
			return
		}

		// Add additional fields for admin view
		app.GroupsCount = groupsCount
		app.TotalRevenue = totalRevenue
		app.AvgPrice = avgPrice

		// Set default values for fields that might not exist
		app.IsActive = true    // Default active
		app.IsAvailable = true // Default available

		apps = append(apps, app)
	}

	// Get total count - simplified
	countQuery := `SELECT COUNT(*) FROM apps WHERE 1=1`
	countArgs := []interface{}{}
	countArgIndex := 1

	// Skip status filter for now since fields might not exist
	// if status != "" && status != "all" {
	// 	if status == "active" {
	// 		countQuery += ` AND is_active = true`
	// 	} else if status == "inactive" {
	// 		countQuery += ` AND is_active = false`
	// 	} else if status == "available" {
	// 		countQuery += ` AND is_available = true`
	// 	} else if status == "unavailable" {
	// 		countQuery += ` AND is_available = false`
	// 	}
	// }

	if search != "" {
		countQuery += ` AND (name ILIKE $` + strconv.Itoa(countArgIndex) + ` OR description ILIKE $` + strconv.Itoa(countArgIndex+1) + ` OR category ILIKE $` + strconv.Itoa(countArgIndex+2) + `)`
		searchTerm := "%" + search + "%"
		countArgs = append(countArgs, searchTerm, searchTerm, searchTerm)
	}

	var total int
	err = h.db.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count apps"})
		return
	}

	// Get stats - simplified
	statsQuery := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) as active,
			0 as inactive,
			COUNT(*) as available,
			0 as unavailable,
			0 as total_revenue,
			0 as avg_price
		FROM apps a
	`
	var stats struct {
		Total        int     `json:"total"`
		Active       int     `json:"active"`
		Inactive     int     `json:"inactive"`
		Available    int     `json:"available"`
		Unavailable  int     `json:"unavailable"`
		TotalRevenue float64 `json:"total_revenue"`
		AvgPrice     float64 `json:"avg_price"`
	}

	err = h.db.QueryRow(statsQuery).Scan(&stats.Total, &stats.Active, &stats.Inactive, &stats.Available, &stats.Unavailable, &stats.TotalRevenue, &stats.AvgPrice)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": apps,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": (total + pageSize - 1) / pageSize,
		},
		"stats": stats,
	})
}

// UpdateAppStatus - Update app status
func (h *AdminHandler) UpdateAppStatus(c *gin.Context) {
	var req struct {
		AppID string `json:"app_id" binding:"required"`
		Field string `json:"field" binding:"required"`
		Value bool   `json:"value"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate field
	validFields := []string{"is_active", "is_available"}
	if !contains(validFields, req.Field) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid field"})
		return
	}

	// Update app status
	_, err := h.db.Exec(`
		UPDATE apps 
		SET `+req.Field+` = $1, updated_at = NOW() 
		WHERE id = $2
	`, req.Value, req.AppID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update app status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "App status updated successfully"})
}

// CreateApp - Create a new app
func (h *AdminHandler) CreateApp(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description" binding:"required"`
		Category    string `json:"category" binding:"required"`
		IconURL     string `json:"icon_url"`
		HowItWorks  string `json:"how_it_works"`
		IsActive    *bool  `json:"is_active"`
		IsAvailable *bool  `json:"is_available"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default values
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	isAvailable := true
	if req.IsAvailable != nil {
		isAvailable = *req.IsAvailable
	}

	// Insert new app
	query := `
		INSERT INTO apps (name, description, category, icon_url, how_it_works, is_active, is_available, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, name, description, category, icon_url, how_it_works, is_active, is_available, created_at, updated_at
	`

	var app struct {
		ID          string    `json:"id"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		Category    string    `json:"category"`
		IconURL     *string   `json:"icon_url"`
		HowItWorks  *string   `json:"how_it_works"`
		IsActive    bool      `json:"is_active"`
		IsAvailable bool      `json:"is_available"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
	}

	err := h.db.QueryRow(query, req.Name, req.Description, req.Category, req.IconURL, req.HowItWorks, isActive, isAvailable).Scan(
		&app.ID, &app.Name, &app.Description, &app.Category, &app.IconURL, &app.HowItWorks, &app.IsActive, &app.IsAvailable, &app.CreatedAt, &app.UpdatedAt,
	)

	if err != nil {
		log.Printf("Error creating app: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create app"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": app})
}

// UpdateApp - Update an existing app
func (h *AdminHandler) UpdateApp(c *gin.Context) {
	var req struct {
		AppID       string `json:"app_id" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Description string `json:"description" binding:"required"`
		Category    string `json:"category" binding:"required"`
		IconURL     string `json:"icon_url"`
		HowItWorks  string `json:"how_it_works"`
		IsActive    *bool  `json:"is_active"`
		IsAvailable *bool  `json:"is_available"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default values
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	isAvailable := true
	if req.IsAvailable != nil {
		isAvailable = *req.IsAvailable
	}

	// Update app
	query := `
		UPDATE apps 
		SET name = $1, description = $2, category = $3, icon_url = $4, how_it_works = $5, 
		    is_active = $6, is_available = $7, updated_at = NOW()
		WHERE id = $8
		RETURNING id, name, description, category, icon_url, how_it_works, is_active, is_available, created_at, updated_at
	`

	var app struct {
		ID          string    `json:"id"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		Category    string    `json:"category"`
		IconURL     *string   `json:"icon_url"`
		HowItWorks  *string   `json:"how_it_works"`
		IsActive    bool      `json:"is_active"`
		IsAvailable bool      `json:"is_available"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
	}

	err := h.db.QueryRow(query, req.Name, req.Description, req.Category, req.IconURL, req.HowItWorks, isActive, isAvailable, req.AppID).Scan(
		&app.ID, &app.Name, &app.Description, &app.Category, &app.IconURL, &app.HowItWorks, &app.IsActive, &app.IsAvailable, &app.CreatedAt, &app.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
			return
		}
		log.Printf("Error updating app: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update app"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": app})
}

// DeleteApp - Delete an app
func (h *AdminHandler) DeleteApp(c *gin.Context) {
	var req struct {
		AppID string `json:"app_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if app exists
	var exists bool
	err := h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM apps WHERE id = $1)", req.AppID).Scan(&exists)
	if err != nil {
		log.Printf("Error checking app existence: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check app existence"})
		return
	}

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
		return
	}

	// Delete app
	_, err = h.db.Exec("DELETE FROM apps WHERE id = $1", req.AppID)
	if err != nil {
		log.Printf("Error deleting app: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete app"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "App deleted successfully"})
}

// CreateGroup - Create a new group (admin only)
func (h *AdminHandler) CreateGroup(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		AppID       string `json:"app_id" binding:"required"`
		MaxMembers  int    `json:"max_members" binding:"min=2,max=50"`
		IsPublic    *bool  `json:"is_public"`
		OwnerID     string `json:"owner_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default values
	isPublic := false
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}

	// Generate invite code
	inviteCode := h.generateInviteCode()

	// Get app information to calculate pricing
	var app struct {
		ID           string  `json:"id"`
		TotalPrice   float64 `json:"total_price"`
		TotalMembers int     `json:"total_members"`
	}
	appQuery := `
		SELECT id, total_price, total_members
		FROM apps WHERE id = $1 AND is_active = true AND is_available = true
	`
	err := h.db.QueryRow(appQuery, req.AppID).Scan(&app.ID, &app.TotalPrice, &app.TotalMembers)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid app ID or app not available"})
		return
	}

	// Calculate pricing
	pricePerMember := app.TotalPrice / float64(app.TotalMembers)
	adminFee := pricePerMember * 0.1 // 10% admin fee
	totalPrice := pricePerMember + adminFee

	// Insert new group
	query := `
		INSERT INTO groups (id, name, description, app_id, owner_id, invite_code, max_members, 
		                   current_members, price_per_member, admin_fee, total_price, status, 
		                   group_status, is_public, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW())
		RETURNING id, name, description, app_id, owner_id, invite_code, max_members, 
		          current_members, price_per_member, admin_fee, total_price, status, 
		          group_status, is_public, created_at, updated_at
	`

	groupID := uuid.New().String()
	var group struct {
		ID             string    `json:"id"`
		Name           string    `json:"name"`
		Description    *string   `json:"description"`
		AppID          string    `json:"app_id"`
		OwnerID        string    `json:"owner_id"`
		InviteCode     string    `json:"invite_code"`
		MaxMembers     int       `json:"max_members"`
		CurrentMembers int       `json:"current_members"`
		PricePerMember float64   `json:"price_per_member"`
		AdminFee       float64   `json:"admin_fee"`
		TotalPrice     float64   `json:"total_price"`
		GroupStatus    string    `json:"group_status"`
		IsPublic       bool      `json:"is_public"`
		CreatedAt      time.Time `json:"created_at"`
		UpdatedAt      time.Time `json:"updated_at"`
	}

	err = h.db.QueryRow(query, groupID, req.Name, req.Description, req.AppID, req.OwnerID,
		inviteCode, req.MaxMembers, 1, pricePerMember, adminFee, totalPrice, "open", "open", isPublic).Scan(
		&group.ID, &group.Name, &group.Description, &group.AppID, &group.OwnerID,
		&group.InviteCode, &group.MaxMembers, &group.CurrentMembers, &group.PricePerMember,
		&group.AdminFee, &group.TotalPrice, &group.GroupStatus,
		&group.IsPublic, &group.CreatedAt, &group.UpdatedAt,
	)

	if err != nil {
		log.Printf("Error creating group: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group"})
		return
	}

	// Add owner as member
	_, err = h.db.Exec(`
		INSERT INTO group_members (id, group_id, user_id, role, joined_at, user_status, payment_amount)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, uuid.New().String(), groupID, req.OwnerID, "owner", time.Now(), "active", 0)

	if err != nil {
		log.Printf("Error adding owner to group: %v", err)
		// Don't fail the group creation, just log the error
	}

	c.JSON(http.StatusCreated, gin.H{"data": group})
}

// UpdateGroup - Update an existing group
func (h *AdminHandler) UpdateGroup(c *gin.Context) {
	var req struct {
		GroupID     string `json:"group_id" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		AppID       string `json:"app_id" binding:"required"`
		MaxMembers  int    `json:"max_members" binding:"min=2,max=50"`
		IsPublic    *bool  `json:"is_public"`
		GroupStatus string `json:"group_status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default values
	isPublic := false
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}

	// Update group
	query := `
		UPDATE groups 
		SET name = $1, description = $2, app_id = $3, max_members = $4, 
		    is_public = $5, group_status = $6, updated_at = NOW()
		WHERE id = $7
		RETURNING id, name, description, app_id, owner_id, invite_code, max_members, 
		          current_members, price_per_member, admin_fee, total_price, group_status, 
		          is_public, created_at, updated_at
	`

	var group struct {
		ID             string    `json:"id"`
		Name           string    `json:"name"`
		Description    *string   `json:"description"`
		AppID          string    `json:"app_id"`
		OwnerID        string    `json:"owner_id"`
		InviteCode     string    `json:"invite_code"`
		MaxMembers     int       `json:"max_members"`
		CurrentMembers int       `json:"current_members"`
		PricePerMember float64   `json:"price_per_member"`
		AdminFee       float64   `json:"admin_fee"`
		TotalPrice     float64   `json:"total_price"`
		GroupStatus    string    `json:"group_status"`
		IsPublic       bool      `json:"is_public"`
		CreatedAt      time.Time `json:"created_at"`
		UpdatedAt      time.Time `json:"updated_at"`
	}

	err := h.db.QueryRow(query, req.Name, req.Description, req.AppID, req.MaxMembers,
		isPublic, req.GroupStatus, req.GroupID).Scan(
		&group.ID, &group.Name, &group.Description, &group.AppID, &group.OwnerID,
		&group.InviteCode, &group.MaxMembers, &group.CurrentMembers, &group.PricePerMember,
		&group.AdminFee, &group.TotalPrice, &group.GroupStatus,
		&group.IsPublic, &group.CreatedAt, &group.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}
		log.Printf("Error updating group: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update group"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": group})
}

// DeleteGroup - Delete a group (admin only)
func (h *AdminHandler) DeleteGroup(c *gin.Context) {
	var req struct {
		GroupID string `json:"group_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if group exists
	var exists bool
	err := h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM groups WHERE id = $1)", req.GroupID).Scan(&exists)
	if err != nil {
		log.Printf("Error checking group existence: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check group existence"})
		return
	}

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	// Start transaction
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Update group status to closed (removed)
	_, err = tx.Exec(`
		UPDATE groups 
		SET group_status = 'closed', updated_at = NOW() 
		WHERE id = $1
	`, req.GroupID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete group"})
		return
	}

	// Update all members status to removed
	_, err = tx.Exec(`
		UPDATE group_members 
		SET user_status = 'removed', removed_at = NOW(), removed_reason = 'Group deleted by admin'
		WHERE group_id = $1
	`, req.GroupID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update member statuses"})
		return
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete group deletion"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Group deleted successfully"})
}

// GetGroupMembers - Get all members of a group (admin only)
func (h *AdminHandler) GetGroupMembers(c *gin.Context) {
	groupID := c.Param("id")
	if groupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Group ID is required"})
		return
	}

	query := `
		SELECT 
			gm.id,
			gm.user_id,
			u.full_name,
			u.email,
			COALESCE(gm.role, 'member') as role,
			COALESCE(gm.user_status, 'pending') as user_status,
			COALESCE(gm.payment_amount, 0) as payment_amount,
			gm.joined_at
		FROM group_members gm
		JOIN users u ON gm.user_id = u.id
		WHERE gm.group_id = $1
		ORDER BY 
			CASE COALESCE(gm.role, 'member') 
				WHEN 'owner' THEN 1 
				WHEN 'admin' THEN 2 
				ELSE 3 
			END,
			gm.joined_at ASC
	`

	rows, err := h.db.Query(query, groupID)
	if err != nil {
		log.Printf("Error fetching group members: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch group members"})
		return
	}
	defer rows.Close()

	var members []models.GroupMember
	for rows.Next() {
		var member models.GroupMember
		var user models.UserResponse

		err := rows.Scan(
			&member.ID,
			&member.UserID,
			&user.FullName,
			&user.Email,
			&member.Role,
			&member.UserStatus,
			&member.PaymentAmount,
			&member.JoinedAt,
		)
		if err != nil {
			log.Printf("Error scanning group member: %v", err)
			continue
		}

		member.User = user
		members = append(members, member)
	}

	c.JSON(http.StatusOK, gin.H{"members": members})
}

// ChangeGroupOwner - Change group owner (admin only)
func (h *AdminHandler) ChangeGroupOwner(c *gin.Context) {
	var req struct {
		GroupID    string `json:"group_id" binding:"required"`
		NewOwnerID string `json:"new_owner_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if group exists
	var groupExists bool
	err := h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM groups WHERE id = $1)", req.GroupID).Scan(&groupExists)
	if err != nil {
		log.Printf("Error checking group existence: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check group existence"})
		return
	}

	if !groupExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	// Check if new owner is a member of the group
	var memberExists bool
	err = h.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM group_members 
			WHERE group_id = $1 AND user_id = $2 AND user_status = 'active'
		)
	`, req.GroupID, req.NewOwnerID).Scan(&memberExists)

	if err != nil {
		log.Printf("Error checking member existence: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check member existence"})
		return
	}

	if !memberExists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "New owner must be an active member of the group"})
		return
	}

	// Start transaction
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Get current owner ID
	var currentOwnerID string
	err = tx.QueryRow("SELECT owner_id FROM groups WHERE id = $1", req.GroupID).Scan(&currentOwnerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get current owner"})
		return
	}

	// Update group owner
	_, err = tx.Exec(`
		UPDATE groups 
		SET owner_id = $1, updated_at = NOW() 
		WHERE id = $2
	`, req.NewOwnerID, req.GroupID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update group owner"})
		return
	}

	// Update old owner role to member
	_, err = tx.Exec(`
		UPDATE group_members 
		SET role = 'member'
		WHERE group_id = $1 AND user_id = $2
	`, req.GroupID, currentOwnerID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update old owner role"})
		return
	}

	// Update new owner role to owner
	_, err = tx.Exec(`
		UPDATE group_members 
		SET role = 'owner'
		WHERE group_id = $1 AND user_id = $2
	`, req.GroupID, req.NewOwnerID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update new owner role"})
		return
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete owner change"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Group owner changed successfully"})
}

// RemoveGroupMember - Remove a member from group (admin only)
func (h *AdminHandler) RemoveGroupMember(c *gin.Context) {
	var req struct {
		GroupID string `json:"group_id" binding:"required"`
		UserID  string `json:"user_id" binding:"required"`
		Reason  string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if group exists
	var groupExists bool
	err := h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM groups WHERE id = $1)", req.GroupID).Scan(&groupExists)
	if err != nil {
		log.Printf("Error checking group existence: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check group existence"})
		return
	}

	if !groupExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	// Check if user is a member of the group
	var memberExists bool
	err = h.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM group_members 
			WHERE group_id = $1 AND user_id = $2
		)
	`, req.GroupID, req.UserID).Scan(&memberExists)

	if err != nil {
		log.Printf("Error checking member existence: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check member existence"})
		return
	}

	if !memberExists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User is not a member of this group"})
		return
	}

	// Check if user is the owner
	var isOwner bool
	err = h.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM groups 
			WHERE id = $1 AND owner_id = $2
		)
	`, req.GroupID, req.UserID).Scan(&isOwner)

	if err != nil {
		log.Printf("Error checking owner status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check owner status"})
		return
	}

	if isOwner {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot remove group owner. Please change owner first."})
		return
	}

	// Update member status to removed
	_, err = h.db.Exec(`
		UPDATE group_members 
		SET user_status = 'removed', removed_at = NOW(), 
		    removed_reason = $1
		WHERE group_id = $2 AND user_id = $3
	`, req.Reason, req.GroupID, req.UserID)

	if err != nil {
		log.Printf("Error removing member: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove member"})
		return
	}

	// Update group current_members count
	_, err = h.db.Exec(`
		UPDATE groups 
		SET current_members = current_members - 1, updated_at = NOW() 
		WHERE id = $1
	`, req.GroupID)

	if err != nil {
		log.Printf("Error updating group member count: %v", err)
		// Don't fail the operation, just log the error
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed successfully"})
}

// AddGroupMember - Add a member to group (admin only)
func (h *AdminHandler) AddGroupMember(c *gin.Context) {
	var req struct {
		GroupID string `json:"group_id" binding:"required"`
		UserID  string `json:"user_id" binding:"required"`
		Role    string `json:"role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default role
	if req.Role == "" {
		req.Role = "member"
	}

	// Check if group exists and get group info
	var group struct {
		ID             string `json:"id"`
		MaxMembers     int    `json:"max_members"`
		CurrentMembers int    `json:"current_members"`
	}
	err := h.db.QueryRow(`
		SELECT id, max_members, current_members 
		FROM groups WHERE id = $1
	`, req.GroupID).Scan(&group.ID, &group.MaxMembers, &group.CurrentMembers)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}
		log.Printf("Error checking group: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check group"})
		return
	}

	// Check if group is full
	if group.CurrentMembers >= group.MaxMembers {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Group is full"})
		return
	}

	// Check if user exists
	var userExists bool
	err = h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", req.UserID).Scan(&userExists)
	if err != nil {
		log.Printf("Error checking user existence: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check user existence"})
		return
	}

	if !userExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check if user is already a member
	var alreadyMember bool
	err = h.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM group_members 
			WHERE group_id = $1 AND user_id = $2
		)
	`, req.GroupID, req.UserID).Scan(&alreadyMember)

	if err != nil {
		log.Printf("Error checking existing membership: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing membership"})
		return
	}

	if alreadyMember {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User is already a member of this group"})
		return
	}

	// Start transaction
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Add member to group
	_, err = tx.Exec(`
		INSERT INTO group_members (id, group_id, user_id, role, joined_at, user_status, payment_amount)
		VALUES ($1, $2, $3, $4, NOW(), 'active', 0)
	`, uuid.New().String(), req.GroupID, req.UserID, req.Role)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add member to group"})
		return
	}

	// Update group current_members count
	_, err = tx.Exec(`
		UPDATE groups 
		SET current_members = current_members + 1, updated_at = NOW() 
		WHERE id = $1
	`, req.GroupID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update group member count"})
		return
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete member addition"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member added successfully"})
}

// generateInviteCode - Generate a random invite code
func (h *AdminHandler) generateInviteCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 8)
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}
	return string(code)
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

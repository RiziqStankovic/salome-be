package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"salome-be/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type BroadcastHandler struct {
	db *sql.DB
}

func NewBroadcastHandler(db *sql.DB) *BroadcastHandler {
	return &BroadcastHandler{db: db}
}

// CreateBroadcast membuat broadcast baru (admin only)
func (h *BroadcastHandler) CreateBroadcast(c *gin.Context) {
	// User ID is already checked by middleware. Just get it.
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userID, ok := userIDInterface.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Debug logging
	fmt.Printf("CreateBroadcast - User ID: %s\n", userID.String())

	var req models.CreateBroadcastRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate target_group_ids if target_type is selected_groups
	if req.TargetType == "selected_groups" && len(req.TargetGroupIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_group_ids is required for selected_groups type"})
		return
	}

	// Set default priority
	if req.Priority == 0 {
		req.Priority = 1
	}

	// Parse end_date if provided
	var endDate *time.Time
	if req.EndDate != nil && *req.EndDate != "" {
		parsed, err := time.Parse(time.RFC3339, *req.EndDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format. Use RFC3339 format"})
			return
		}
		endDate = &parsed
	}

	broadcastID := uuid.New().String()
	now := time.Now()

	// Debug logging
	fmt.Printf("Creating broadcast with data:\n")
	fmt.Printf("  ID: %s\n", broadcastID)
	fmt.Printf("  Admin ID: %s\n", userID.String())
	fmt.Printf("  Title: %s\n", req.Title)
	fmt.Printf("  Message: %s\n", req.Message)
	fmt.Printf("  Target Type: %s\n", req.TargetType)
	fmt.Printf("  Target Group IDs: %v\n", req.TargetGroupIDs)
	fmt.Printf("  Priority: %d\n", req.Priority)
	fmt.Printf("  End Date: %v\n", endDate)

	_, err := h.db.Exec(`
		INSERT INTO broadcasts (id, admin_id, title, message, target_type, target_group_ids, is_active, priority, start_date, end_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, broadcastID, userID, req.Title, req.Message, req.TargetType, pq.Array(req.TargetGroupIDs), true, req.Priority, now, endDate, now, now)

	if err != nil {
		fmt.Printf("Database error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create broadcast", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Broadcast created successfully",
		"data": gin.H{
			"id": broadcastID,
		},
	})
}

// GetBroadcasts mendapatkan daftar broadcast (admin only)
func (h *BroadcastHandler) GetBroadcasts(c *gin.Context) {
	// User ID is already checked by middleware

	// Get query parameters
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	active := c.DefaultQuery("active", "")

	pageInt := 1
	if p, err := strconv.Atoi(page); err == nil && p > 0 {
		pageInt = p
	}
	pageSizeInt := 20
	if ps, err := strconv.Atoi(pageSize); err == nil && ps > 0 && ps <= 100 {
		pageSizeInt = ps
	}

	offset := (pageInt - 1) * pageSizeInt

	// Build query
	query := `
		SELECT id, title, message, target_type, target_group_ids, is_active, priority, start_date, end_date, created_at
		FROM broadcasts
	`
	args := []interface{}{}
	argIndex := 1

	if active != "" {
		if active == "true" {
			query += " WHERE is_active = true AND (end_date IS NULL OR end_date > NOW())"
		} else if active == "false" {
			query += " WHERE is_active = false OR (end_date IS NOT NULL AND end_date <= NOW())"
		}
	}

	query += fmt.Sprintf(" ORDER BY priority DESC, created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, pageSizeInt, offset)

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch broadcasts"})
		return
	}
	defer rows.Close()

	var broadcasts []models.BroadcastResponse
	for rows.Next() {
		var broadcast models.BroadcastResponse
		var targetGroupIDs pq.StringArray

		err := rows.Scan(
			&broadcast.ID,
			&broadcast.Title,
			&broadcast.Message,
			&broadcast.TargetType,
			&targetGroupIDs,
			&broadcast.IsActive,
			&broadcast.Priority,
			&broadcast.StartDate,
			&broadcast.EndDate,
			&broadcast.CreatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan broadcast"})
			return
		}

		broadcast.TargetGroupIDs = []string(targetGroupIDs)
		broadcasts = append(broadcasts, broadcast)
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM broadcasts"
	if active != "" {
		if active == "true" {
			countQuery += " WHERE is_active = true AND (end_date IS NULL OR end_date > NOW())"
		} else if active == "false" {
			countQuery += " WHERE is_active = false OR (end_date IS NOT NULL AND end_date <= NOW())"
		}
	}

	err = h.db.QueryRow(countQuery).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count broadcasts"})
		return
	}

	response := models.GetBroadcastsResponse{
		Broadcasts: broadcasts,
		Total:      total,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Broadcasts retrieved successfully",
		"data":    response,
	})
}

// GetGroupBroadcast mendapatkan broadcast aktif untuk group tertentu
func (h *BroadcastHandler) GetGroupBroadcast(c *gin.Context) {
	groupID := c.Param("group_id")
	if groupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Group ID is required"})
		return
	}

	// Get all active broadcasts for this group
	query := `
		SELECT id, title, message, priority, start_date, end_date, created_at
		FROM broadcasts
		WHERE is_active = true 
		AND (end_date IS NULL OR end_date > NOW())
		AND (
			target_type = 'all_groups' 
			OR (target_type = 'selected_groups' AND $1 = ANY(target_group_ids))
		)
		ORDER BY priority DESC, created_at DESC
	`

	rows, err := h.db.Query(query, groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch group broadcasts"})
		return
	}
	defer rows.Close()

	var broadcasts []models.BroadcastResponse
	for rows.Next() {
		var broadcast models.BroadcastResponse
		var endDate sql.NullTime

		err := rows.Scan(
			&broadcast.ID,
			&broadcast.Title,
			&broadcast.Message,
			&broadcast.Priority,
			&broadcast.StartDate,
			&endDate,
			&broadcast.CreatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan broadcast"})
			return
		}

		if endDate.Valid {
			broadcast.EndDate = &endDate.Time
		}

		broadcasts = append(broadcasts, broadcast)
	}

	if len(broadcasts) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "No active broadcasts for this group",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Group broadcasts retrieved successfully",
		"data":    broadcasts,
	})
}

// UpdateBroadcast mengupdate broadcast (admin only)
func (h *BroadcastHandler) UpdateBroadcast(c *gin.Context) {
	// User ID is already checked by middleware

	broadcastID := c.Param("id")
	if broadcastID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Broadcast ID is required"})
		return
	}

	var req models.UpdateBroadcastRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build update query dynamically
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Title != "" {
		setParts = append(setParts, fmt.Sprintf("title = $%d", argIndex))
		args = append(args, req.Title)
		argIndex++
	}

	if req.Message != "" {
		setParts = append(setParts, fmt.Sprintf("message = $%d", argIndex))
		args = append(args, req.Message)
		argIndex++
	}

	if req.TargetType != "" {
		setParts = append(setParts, fmt.Sprintf("target_type = $%d", argIndex))
		args = append(args, req.TargetType)
		argIndex++

		if req.TargetType == "selected_groups" {
			setParts = append(setParts, fmt.Sprintf("target_group_ids = $%d", argIndex))
			args = append(args, pq.Array(req.TargetGroupIDs))
			argIndex++
		} else if req.TargetType == "all_groups" {
			setParts = append(setParts, "target_group_ids = NULL")
		}
	}

	// Handle target_group_ids update when target_type is not changed but target_group_ids is provided
	if req.TargetGroupIDs != nil && req.TargetType == "" {
		// Check current target_type first
		var currentTargetType string
		err := h.db.QueryRow("SELECT target_type FROM broadcasts WHERE id = $1", broadcastID).Scan(&currentTargetType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check current target type"})
			return
		}

		if currentTargetType == "selected_groups" {
			setParts = append(setParts, fmt.Sprintf("target_group_ids = $%d", argIndex))
			args = append(args, pq.Array(req.TargetGroupIDs))
			argIndex++
		}
	}

	if req.IsActive != nil {
		setParts = append(setParts, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *req.IsActive)
		argIndex++
	}

	if req.Priority != nil {
		setParts = append(setParts, fmt.Sprintf("priority = $%d", argIndex))
		args = append(args, *req.Priority)
		argIndex++
	}

	if req.EndDate != nil {
		if *req.EndDate == "" {
			setParts = append(setParts, "end_date = NULL")
		} else {
			parsed, err := time.Parse(time.RFC3339, *req.EndDate)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format. Use RFC3339 format"})
				return
			}
			setParts = append(setParts, fmt.Sprintf("end_date = $%d", argIndex))
			args = append(args, parsed)
			argIndex++
		}
	}

	if len(setParts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	setParts = append(setParts, fmt.Sprintf("id = $%d", argIndex))
	args = append(args, broadcastID)

	query := fmt.Sprintf("UPDATE broadcasts SET %s WHERE id = $%d", strings.Join(setParts, ", "), argIndex)

	_, err := h.db.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update broadcast"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Broadcast updated successfully",
	})
}

// DeleteBroadcast menghapus broadcast (admin only)
func (h *BroadcastHandler) DeleteBroadcast(c *gin.Context) {
	// User ID is already checked by middleware

	broadcastID := c.Param("id")
	if broadcastID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Broadcast ID is required"})
		return
	}

	_, err := h.db.Exec("DELETE FROM broadcasts WHERE id = $1", broadcastID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete broadcast"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Broadcast deleted successfully",
	})
}

package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"salome-be/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserBroadcastHandler struct {
	db *sql.DB
}

func NewUserBroadcastHandler(db *sql.DB) *UserBroadcastHandler {
	return &UserBroadcastHandler{db: db}
}

// GetUserBroadcasts - Get list of user broadcasts with pagination and filters
func (h *UserBroadcastHandler) GetUserBroadcasts(c *gin.Context) {
	var req models.UserBroadcastListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	offset := (req.Page - 1) * req.PageSize

	// Build query
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if req.Status != "" {
		whereClause += fmt.Sprintf(" AND ub.status = $%d", argIndex)
		args = append(args, req.Status)
		argIndex++
	}

	if req.Search != "" {
		whereClause += fmt.Sprintf(" AND (ub.title ILIKE $%d OR ub.message ILIKE $%d)", argIndex, argIndex+1)
		searchTerm := "%" + req.Search + "%"
		args = append(args, searchTerm, searchTerm)
		argIndex += 2
	}

	// Get broadcasts
	query := fmt.Sprintf(`
		SELECT ub.id, ub.title, ub.message, ub.target_type, ub.priority, ub.status,
		       ub.created_by, ub.scheduled_at, ub.sent_at, ub.end_date,
		       ub.success_count, ub.error_count, ub.total_targets,
		       ub.created_at, ub.updated_at, u.full_name
		FROM user_broadcast ub
		LEFT JOIN users u ON ub.created_by = u.id
		%s
		ORDER BY ub.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, req.PageSize, offset)

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch broadcasts"})
		return
	}
	defer rows.Close()

	var broadcasts []models.UserBroadcast
	for rows.Next() {
		var broadcast models.UserBroadcast
		var creatorName sql.NullString

		err := rows.Scan(
			&broadcast.ID, &broadcast.Title, &broadcast.Message, &broadcast.TargetType,
			&broadcast.Priority, &broadcast.Status, &broadcast.CreatedBy,
			&broadcast.ScheduledAt, &broadcast.SentAt, &broadcast.EndDate,
			&broadcast.SuccessCount, &broadcast.ErrorCount, &broadcast.TotalTargets,
			&broadcast.CreatedAt, &broadcast.UpdatedAt, &creatorName,
		)
		if err != nil {
			fmt.Printf("Error scanning broadcast row: %v\n", err)
			continue
		}

		if creatorName.Valid {
			broadcast.CreatorName = &creatorName.String
		}

		// Get targets if target_type is 'selected'
		fmt.Printf("DEBUG: GetUserBroadcasts - Broadcast ID: %s, TargetType: %s\n", broadcast.ID, broadcast.TargetType)
		if broadcast.TargetType == "selected" {
			targetRows, err := h.db.Query(`
				SELECT ubt.id, ubt.broadcast_id, ubt.user_id, ubt.status, ubt.sent_at, ubt.error_message,
				       ubt.created_at, u.full_name, u.email
				FROM user_broadcast_targets ubt
				LEFT JOIN users u ON ubt.user_id = u.id
				WHERE ubt.broadcast_id = $1
				ORDER BY ubt.created_at ASC
			`, broadcast.ID)

			if err == nil {
				targetCount := 0
				for targetRows.Next() {
					targetCount++
					var target models.UserBroadcastTarget
					var userName, userEmail sql.NullString

					err := targetRows.Scan(
						&target.ID, &target.BroadcastID, &target.UserID, &target.Status,
						&target.SentAt, &target.ErrorMessage, &target.CreatedAt,
						&userName, &userEmail,
					)
					if err != nil {
						continue
					}

					if userName.Valid {
						target.UserName = &userName.String
					}
					if userEmail.Valid {
						target.UserEmail = &userEmail.String
					}

					broadcast.Targets = append(broadcast.Targets, target)
				}
				targetRows.Close()
				fmt.Printf("DEBUG: Found %d targets for broadcast %s\n", targetCount, broadcast.ID)
			} else {
				fmt.Printf("DEBUG: Error querying targets for broadcast %s: %v\n", broadcast.ID, err)
			}
		}

		broadcasts = append(broadcasts, broadcast)
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM user_broadcast ub %s", whereClause)
	var total int
	err = h.db.QueryRow(countQuery, args[:len(args)-2]...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count broadcasts"})
		return
	}

	// Get stats
	statsQuery := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'draft') as draft,
			COUNT(*) FILTER (WHERE status = 'scheduled') as scheduled,
			COUNT(*) FILTER (WHERE status = 'sent') as sent,
			COUNT(*) FILTER (WHERE status = 'cancelled') as cancelled
		FROM user_broadcast
	`
	var stats models.UserBroadcastStats
	err = h.db.QueryRow(statsQuery).Scan(&stats.Total, &stats.Draft, &stats.Scheduled, &stats.Sent, &stats.Cancelled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"broadcasts": broadcasts,
		"pagination": gin.H{
			"page":        req.Page,
			"page_size":   req.PageSize,
			"total":       total,
			"total_pages": (total + req.PageSize - 1) / req.PageSize,
		},
		"stats": stats,
	})
}

// CreateUserBroadcast - Create a new user broadcast
func (h *UserBroadcastHandler) CreateUserBroadcast(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req models.UserBroadcastCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse scheduled_at if provided
	var scheduledAt *time.Time
	if req.ScheduledAt != nil && *req.ScheduledAt != "" {
		// Try multiple date formats
		formats := []string{
			time.RFC3339,           // "2006-01-02T15:04:05Z07:00"
			"2006-01-02T15:04:05Z", // "2006-01-02T15:04:05Z"
			"2006-01-02T15:04:05",  // "2006-01-02T15:04:05"
			"2006-01-02T15:04",     // "2006-01-02T15:04"
			"2006-01-02 15:04:05",  // "2006-01-02 15:04:05"
			"2006-01-02 15:04",     // "2006-01-02 15:04"
		}

		var parsed time.Time
		var err error
		for _, format := range formats {
			parsed, err = time.Parse(format, *req.ScheduledAt)
			if err == nil {
				break
			}
		}

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scheduled_at format. Expected format: YYYY-MM-DDTHH:mm or YYYY-MM-DDTHH:mm:ss"})
			return
		}
		scheduledAt = &parsed
	}

	// Parse end_date if provided
	var endDate *time.Time
	if req.EndDate != nil && *req.EndDate != "" {
		// Try multiple date formats
		formats := []string{
			time.RFC3339,           // "2006-01-02T15:04:05Z07:00"
			"2006-01-02T15:04:05Z", // "2006-01-02T15:04:05Z"
			"2006-01-02T15:04:05",  // "2006-01-02T15:04:05"
			"2006-01-02T15:04",     // "2006-01-02T15:04"
			"2006-01-02 15:04:05",  // "2006-01-02 15:04:05"
			"2006-01-02 15:04",     // "2006-01-02 15:04"
		}

		var parsed time.Time
		var err error
		for _, format := range formats {
			parsed, err = time.Parse(format, *req.EndDate)
			if err == nil {
				break
			}
		}

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format. Expected format: YYYY-MM-DDTHH:mm or YYYY-MM-DDTHH:mm:ss"})
			return
		}
		endDate = &parsed
	}

	// Determine status
	status := "draft"
	if scheduledAt != nil && scheduledAt.After(time.Now()) {
		status = "scheduled"
	}

	// Start transaction
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Create broadcast
	broadcastID := uuid.New()
	_, err = tx.Exec(`
		INSERT INTO user_broadcast (id, title, message, target_type, priority, status, created_by, scheduled_at, end_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, broadcastID, req.Title, req.Message, req.TargetType, req.Priority, status, userID, scheduledAt, endDate, time.Now(), time.Now())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create broadcast"})
		return
	}

	// Handle targets
	var totalTargets int
	if req.TargetType == "all" {
		// Get all users count
		err = tx.QueryRow("SELECT COUNT(*) FROM users WHERE is_admin = false").Scan(&totalTargets)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count users"})
			return
		}
	} else if req.TargetType == "selected" {
		// Create targets for selected users
		totalTargets = len(req.UserIDs)
		for _, userIDStr := range req.UserIDs {
			userUUID, err := uuid.Parse(userIDStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID: " + userIDStr})
				return
			}

			_, err = tx.Exec(`
				INSERT INTO user_broadcast_targets (id, broadcast_id, user_id, status, created_at)
				VALUES ($1, $2, $3, $4, $5)
			`, uuid.New(), broadcastID, userUUID, "pending", time.Now())

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create target"})
				return
			}
		}
	}

	// Update total_targets
	_, err = tx.Exec("UPDATE user_broadcast SET total_targets = $1 WHERE id = $2", totalTargets, broadcastID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update total targets"})
		return
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Get created broadcast
	var broadcast models.UserBroadcast
	var creatorName sql.NullString
	err = h.db.QueryRow(`
		SELECT ub.id, ub.title, ub.message, ub.target_type, ub.priority, ub.status,
		       ub.created_by, ub.scheduled_at, ub.sent_at, ub.end_date,
		       ub.success_count, ub.error_count, ub.total_targets,
		       ub.created_at, ub.updated_at, u.full_name
		FROM user_broadcast ub
		LEFT JOIN users u ON ub.created_by = u.id
		WHERE ub.id = $1
	`, broadcastID).Scan(
		&broadcast.ID, &broadcast.Title, &broadcast.Message, &broadcast.TargetType,
		&broadcast.Priority, &broadcast.Status, &broadcast.CreatedBy,
		&broadcast.ScheduledAt, &broadcast.SentAt, &broadcast.EndDate,
		&broadcast.SuccessCount, &broadcast.ErrorCount, &broadcast.TotalTargets,
		&broadcast.CreatedAt, &broadcast.UpdatedAt, &creatorName,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch created broadcast"})
		return
	}

	if creatorName.Valid {
		broadcast.CreatorName = &creatorName.String
	}

	c.JSON(http.StatusCreated, gin.H{"broadcast": broadcast})
}

// GetUserBroadcast - Get single user broadcast by ID
func (h *UserBroadcastHandler) GetUserBroadcast(c *gin.Context) {
	broadcastIDStr := c.Param("id")
	broadcastID, err := uuid.Parse(broadcastIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid broadcast ID"})
		return
	}

	// Get broadcast
	var broadcast models.UserBroadcast
	var creatorName sql.NullString
	err = h.db.QueryRow(`
		SELECT ub.id, ub.title, ub.message, ub.target_type, ub.priority, ub.status,
		       ub.created_by, ub.scheduled_at, ub.sent_at, ub.end_date,
		       ub.success_count, ub.error_count, ub.total_targets,
		       ub.created_at, ub.updated_at, u.full_name
		FROM user_broadcast ub
		LEFT JOIN users u ON ub.created_by = u.id
		WHERE ub.id = $1
	`, broadcastID).Scan(
		&broadcast.ID, &broadcast.Title, &broadcast.Message, &broadcast.TargetType,
		&broadcast.Priority, &broadcast.Status, &broadcast.CreatedBy,
		&broadcast.ScheduledAt, &broadcast.SentAt, &broadcast.EndDate,
		&broadcast.SuccessCount, &broadcast.ErrorCount, &broadcast.TotalTargets,
		&broadcast.CreatedAt, &broadcast.UpdatedAt, &creatorName,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Broadcast not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch broadcast"})
		return
	}

	if creatorName.Valid {
		broadcast.CreatorName = &creatorName.String
	}

	// Get targets if target_type is 'selected'
	if broadcast.TargetType == "selected" {
		rows, err := h.db.Query(`
			SELECT ubt.id, ubt.broadcast_id, ubt.user_id, ubt.status, ubt.sent_at, ubt.error_message,
			       ubt.created_at, u.full_name, u.email
			FROM user_broadcast_targets ubt
			LEFT JOIN users u ON ubt.user_id = u.id
			WHERE ubt.broadcast_id = $1
			ORDER BY ubt.created_at ASC
		`, broadcastID)

		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var target models.UserBroadcastTarget
				var userName, userEmail sql.NullString

				err := rows.Scan(
					&target.ID, &target.BroadcastID, &target.UserID, &target.Status,
					&target.SentAt, &target.ErrorMessage, &target.CreatedAt,
					&userName, &userEmail,
				)
				if err != nil {
					continue
				}

				if userName.Valid {
					target.UserName = &userName.String
				}
				if userEmail.Valid {
					target.UserEmail = &userEmail.String
				}

				broadcast.Targets = append(broadcast.Targets, target)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"broadcast": broadcast})
}

// UpdateUserBroadcast - Update user broadcast
func (h *UserBroadcastHandler) UpdateUserBroadcast(c *gin.Context) {
	broadcastIDStr := c.Param("id")
	broadcastID, err := uuid.Parse(broadcastIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid broadcast ID"})
		return
	}

	var req models.UserBroadcastUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build update query dynamically
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Title != nil {
		setParts = append(setParts, fmt.Sprintf("title = $%d", argIndex))
		args = append(args, *req.Title)
		argIndex++
	}

	if req.Message != nil {
		setParts = append(setParts, fmt.Sprintf("message = $%d", argIndex))
		args = append(args, *req.Message)
		argIndex++
	}

	if req.TargetType != nil {
		setParts = append(setParts, fmt.Sprintf("target_type = $%d", argIndex))
		args = append(args, *req.TargetType)
		argIndex++
	}

	if req.Priority != nil {
		setParts = append(setParts, fmt.Sprintf("priority = $%d", argIndex))
		args = append(args, *req.Priority)
		argIndex++
	}

	if req.Status != nil {
		setParts = append(setParts, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *req.Status)
		argIndex++
	}

	if req.ScheduledAt != nil {
		if *req.ScheduledAt == "" {
			setParts = append(setParts, "scheduled_at = NULL")
		} else {
			// Try multiple date formats
			formats := []string{
				time.RFC3339,           // "2006-01-02T15:04:05Z07:00"
				"2006-01-02T15:04:05Z", // "2006-01-02T15:04:05Z"
				"2006-01-02T15:04:05",  // "2006-01-02T15:04:05"
				"2006-01-02T15:04",     // "2006-01-02T15:04"
				"2006-01-02 15:04:05",  // "2006-01-02 15:04:05"
				"2006-01-02 15:04",     // "2006-01-02 15:04"
			}

			var parsed time.Time
			var err error
			for _, format := range formats {
				parsed, err = time.Parse(format, *req.ScheduledAt)
				if err == nil {
					break
				}
			}

			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scheduled_at format. Expected format: YYYY-MM-DDTHH:mm or YYYY-MM-DDTHH:mm:ss"})
				return
			}
			setParts = append(setParts, fmt.Sprintf("scheduled_at = $%d", argIndex))
			args = append(args, parsed)
			argIndex++
		}
	}

	if req.EndDate != nil {
		if *req.EndDate == "" {
			setParts = append(setParts, "end_date = NULL")
		} else {
			// Try multiple date formats
			formats := []string{
				time.RFC3339,           // "2006-01-02T15:04:05Z07:00"
				"2006-01-02T15:04:05Z", // "2006-01-02T15:04:05Z"
				"2006-01-02T15:04:05",  // "2006-01-02T15:04:05"
				"2006-01-02T15:04",     // "2006-01-02T15:04"
				"2006-01-02 15:04:05",  // "2006-01-02 15:04:05"
				"2006-01-02 15:04",     // "2006-01-02 15:04"
			}

			var parsed time.Time
			var err error
			for _, format := range formats {
				parsed, err = time.Parse(format, *req.EndDate)
				if err == nil {
					break
				}
			}

			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format. Expected format: YYYY-MM-DDTHH:mm or YYYY-MM-DDTHH:mm:ss"})
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

	// Add updated_at
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	// Add WHERE clause
	args = append(args, broadcastID)

	// Build the SET clause
	setClause := setParts[0]
	for i := 1; i < len(setParts); i++ {
		setClause += ", " + setParts[i]
	}

	query := fmt.Sprintf("UPDATE user_broadcast SET %s WHERE id = $%d", setClause, argIndex)

	_, err = h.db.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update broadcast"})
		return
	}

	// Handle user targets if target_type is being updated
	fmt.Printf("DEBUG: UpdateUserBroadcast - TargetType: %v, UserIDs: %v\n", req.TargetType, req.UserIDs)
	if req.TargetType != nil {
		// Delete existing targets
		_, err = h.db.Exec("DELETE FROM user_broadcast_targets WHERE broadcast_id = $1", broadcastID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete existing targets"})
			return
		}

		// Insert new targets if target_type is 'selected'
		if *req.TargetType == "selected" && req.UserIDs != nil && len(req.UserIDs) > 0 {
			// Convert string IDs to UUIDs
			var userUUIDs []uuid.UUID
			for _, userIDStr := range req.UserIDs {
				userID, err := uuid.Parse(userIDStr)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID: " + userIDStr})
					return
				}
				userUUIDs = append(userUUIDs, userID)
			}

			// Insert new targets
			for _, userID := range userUUIDs {
				_, err = h.db.Exec(
					"INSERT INTO user_broadcast_targets (id, broadcast_id, user_id, status, created_at) VALUES ($1, $2, $3, $4, $5)",
					uuid.New(), broadcastID, userID, "pending", time.Now(),
				)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert user target"})
					return
				}
			}

			// Update total_targets count
			_, err = h.db.Exec("UPDATE user_broadcast SET total_targets = $1 WHERE id = $2", len(userUUIDs), broadcastID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update total targets count"})
				return
			}
		} else if *req.TargetType == "all" {
			// For 'all' target type, set total_targets to 0 (will be calculated when sending)
			_, err = h.db.Exec("UPDATE user_broadcast SET total_targets = 0 WHERE id = $1", broadcastID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update total targets count"})
				return
			}
		}
	} else if req.UserIDs != nil {
		// If only user_ids is provided (target_type not changed), update targets for existing 'selected' broadcasts
		// First check if current broadcast is 'selected' type
		var currentTargetType string
		err = h.db.QueryRow("SELECT target_type FROM user_broadcast WHERE id = $1", broadcastID).Scan(&currentTargetType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get current target type"})
			return
		}

		if currentTargetType == "selected" {
			// Delete existing targets
			_, err = h.db.Exec("DELETE FROM user_broadcast_targets WHERE broadcast_id = $1", broadcastID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete existing targets"})
				return
			}

			// Convert string IDs to UUIDs
			var userUUIDs []uuid.UUID
			for _, userIDStr := range req.UserIDs {
				userID, err := uuid.Parse(userIDStr)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID: " + userIDStr})
					return
				}
				userUUIDs = append(userUUIDs, userID)
			}

			// Insert new targets
			for _, userID := range userUUIDs {
				_, err = h.db.Exec(
					"INSERT INTO user_broadcast_targets (id, broadcast_id, user_id, status, created_at) VALUES ($1, $2, $3, $4, $5)",
					uuid.New(), broadcastID, userID, "pending", time.Now(),
				)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert user target"})
					return
				}
			}

			// Update total_targets count
			_, err = h.db.Exec("UPDATE user_broadcast SET total_targets = $1 WHERE id = $2", len(userUUIDs), broadcastID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update total targets count"})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Broadcast updated successfully"})
}

// DeleteUserBroadcast - Delete user broadcast
func (h *UserBroadcastHandler) DeleteUserBroadcast(c *gin.Context) {
	broadcastIDStr := c.Param("id")
	broadcastID, err := uuid.Parse(broadcastIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid broadcast ID"})
		return
	}

	// Check if broadcast exists and get status
	var status string
	err = h.db.QueryRow("SELECT status FROM user_broadcast WHERE id = $1", broadcastID).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Broadcast not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check broadcast status"})
		return
	}

	// Only allow deletion of draft or cancelled broadcasts
	if status != "draft" && status != "cancelled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Can only delete draft or cancelled broadcasts"})
		return
	}

	// Delete broadcast (targets will be deleted by CASCADE)
	_, err = h.db.Exec("DELETE FROM user_broadcast WHERE id = $1", broadcastID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete broadcast"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Broadcast deleted successfully"})
}

// SendUserBroadcast - Send broadcast to users
func (h *UserBroadcastHandler) SendUserBroadcast(c *gin.Context) {
	broadcastIDStr := c.Param("id")
	broadcastID, err := uuid.Parse(broadcastIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid broadcast ID"})
		return
	}

	// Check if broadcast exists and get details
	var broadcast models.UserBroadcast
	var targetType string
	var status string
	err = h.db.QueryRow(`
		SELECT id, title, message, target_type, status, total_targets
		FROM user_broadcast 
		WHERE id = $1
	`, broadcastID).Scan(
		&broadcast.ID, &broadcast.Title, &broadcast.Message,
		&targetType, &status, &broadcast.TotalTargets,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Broadcast not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch broadcast"})
		return
	}

	// Check if broadcast can be sent
	if status != "draft" && status != "scheduled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Broadcast can only be sent if status is draft or scheduled"})
		return
	}

	// Start transaction
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Update broadcast status to sent
	_, err = tx.Exec(`
		UPDATE user_broadcast 
		SET status = 'sent', sent_at = $1, updated_at = $1
		WHERE id = $2
	`, time.Now(), broadcastID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update broadcast status"})
		return
	}

	// Get target users
	var userIDs []uuid.UUID
	if targetType == "all" {
		rows, err := tx.Query("SELECT id FROM users WHERE is_admin = false")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
			return
		}
		defer rows.Close()

		for rows.Next() {
			var userID uuid.UUID
			if err := rows.Scan(&userID); err != nil {
				continue
			}
			userIDs = append(userIDs, userID)
		}
	} else {
		// Get selected users from targets table
		rows, err := tx.Query(`
			SELECT user_id FROM user_broadcast_targets 
			WHERE broadcast_id = $1
		`, broadcastID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch target users"})
			return
		}
		defer rows.Close()

		for rows.Next() {
			var userID uuid.UUID
			if err := rows.Scan(&userID); err != nil {
				continue
			}
			userIDs = append(userIDs, userID)
		}
	}

	// Send broadcast to each user (simulate sending)
	successCount := 0
	errorCount := 0

	for _, userID := range userIDs {
		// Here you would implement actual sending logic (email, push notification, etc.)
		// For now, we'll just simulate success

		// Update target status if it's a selected broadcast
		if targetType == "selected" {
			_, err = tx.Exec(`
				UPDATE user_broadcast_targets 
				SET status = 'sent', sent_at = $1
				WHERE broadcast_id = $2 AND user_id = $3
			`, time.Now(), broadcastID, userID)
			if err != nil {
				errorCount++
				continue
			}
		}

		// Create notification or store in user's broadcast inbox
		// This is where you would integrate with your notification system
		// For now, we'll just count as success
		successCount++
	}

	// Update broadcast with results
	_, err = tx.Exec(`
		UPDATE user_broadcast 
		SET success_count = $1, error_count = $2, updated_at = $3
		WHERE id = $4
	`, successCount, errorCount, time.Now(), broadcastID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update broadcast results"})
		return
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Broadcast sent successfully",
		"success_count": successCount,
		"error_count":   errorCount,
		"total_targets": len(userIDs),
	})
}

// GetUserBroadcastsForDashboard - Get broadcasts for user dashboard
func (h *UserBroadcastHandler) GetUserBroadcastsForDashboard(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Get broadcasts that are sent and visible to users
	rows, err := h.db.Query(`
		SELECT ub.id, ub.title, ub.message, ub.priority, ub.sent_at, ub.end_date,
		       ub.created_at, u.full_name as creator_name
		FROM user_broadcast ub
		LEFT JOIN users u ON ub.created_by = u.id
		WHERE ub.status = 'sent'
		  AND (ub.end_date IS NULL OR ub.end_date > NOW())
		ORDER BY ub.sent_at DESC
		LIMIT 20
	`)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch broadcasts"})
		return
	}
	defer rows.Close()

	var broadcasts []models.UserBroadcast
	for rows.Next() {
		var broadcast models.UserBroadcast
		var creatorName sql.NullString
		var sentAt, endDate sql.NullTime

		err := rows.Scan(
			&broadcast.ID, &broadcast.Title, &broadcast.Message, &broadcast.Priority,
			&sentAt, &endDate, &broadcast.CreatedAt, &creatorName,
		)
		if err != nil {
			continue
		}

		if sentAt.Valid {
			broadcast.SentAt = &sentAt.Time
		}
		if endDate.Valid {
			broadcast.EndDate = &endDate.Time
		}
		if creatorName.Valid {
			broadcast.CreatorName = &creatorName.String
		}

		broadcasts = append(broadcasts, broadcast)
	}

	c.JSON(http.StatusOK, gin.H{"broadcasts": broadcasts})
}

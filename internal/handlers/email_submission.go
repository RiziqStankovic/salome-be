package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"salome-be/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type EmailSubmissionHandler struct {
	db *sql.DB
}

func NewEmailSubmissionHandler(db *sql.DB) *EmailSubmissionHandler {
	return &EmailSubmissionHandler{db: db}
}

// GetEmailSubmissions - Get all email submissions with filters (admin only)
func (h *EmailSubmissionHandler) GetEmailSubmissions(c *gin.Context) {
	// Get query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	search := c.Query("search")

	// Calculate offset
	offset := (page - 1) * pageSize

	// Build query
	query := `
		SELECT 
			es.id, es.user_id, es.group_id, es.app_id, es.email, es.username, es.full_name,
			es.status, es.submitted_at, es.reviewed_at, es.reviewed_by, es.notes,
			es.created_at, es.updated_at,
			u.id, u.full_name, u.email,
			g.id, g.name, g.app_id,
			a.id, a.name, a.icon_url
		FROM email_submissions es
		LEFT JOIN users u ON es.user_id = u.id
		LEFT JOIN groups g ON es.group_id = g.id
		LEFT JOIN apps a ON es.app_id = a.id
		WHERE 1=1
	`

	args := []interface{}{}
	argIndex := 1

	// Add status filter
	if status != "" && status != "all" {
		query += fmt.Sprintf(" AND es.status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	// Add search filter
	if search != "" {
		query += fmt.Sprintf(" AND (es.email ILIKE $%d OR es.full_name ILIKE $%d OR g.name ILIKE $%d OR a.name ILIKE $%d)",
			argIndex, argIndex+1, argIndex+2, argIndex+3)
		searchTerm := "%" + search + "%"
		args = append(args, searchTerm, searchTerm, searchTerm, searchTerm)
		argIndex += 4
	}

	// Add ordering and pagination
	query += fmt.Sprintf(" ORDER BY es.submitted_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, pageSize, offset)

	// Execute query
	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch email submissions"})
		return
	}
	defer rows.Close()

	var submissions []models.EmailSubmissionResponse
	for rows.Next() {
		var submission models.EmailSubmissionResponse
		var user models.User
		var group models.Group
		var app models.App

		err := rows.Scan(
			&submission.ID, &submission.UserID, &submission.GroupID, &submission.AppID,
			&submission.Email, &submission.Username, &submission.FullName,
			&submission.Status, &submission.SubmittedAt, &submission.ReviewedAt, &submission.ReviewedBy, &submission.Notes,
			&submission.CreatedAt, &submission.UpdatedAt,
			&user.ID, &user.FullName, &user.Email,
			&group.ID, &group.Name, &group.AppID,
			&app.ID, &app.Name, &app.IconURL,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan email submission"})
			return
		}

		submission.User = &user
		submission.Group = &group
		submission.App = &app
		submissions = append(submissions, submission)
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM email_submissions es WHERE 1=1"
	countArgs := []interface{}{}
	countArgIndex := 1

	if status != "" && status != "all" {
		countQuery += fmt.Sprintf(" AND es.status = $%d", countArgIndex)
		countArgs = append(countArgs, status)
		countArgIndex++
	}

	if search != "" {
		countQuery += fmt.Sprintf(" AND (es.email ILIKE $%d OR es.full_name ILIKE $%d OR g.name ILIKE $%d OR a.name ILIKE $%d)",
			countArgIndex, countArgIndex+1, countArgIndex+2, countArgIndex+3)
		searchTerm := "%" + search + "%"
		countArgs = append(countArgs, searchTerm, searchTerm, searchTerm, searchTerm)
	}

	var total int
	err = h.db.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count email submissions"})
		return
	}

	// Get stats
	statsQuery := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
			COUNT(CASE WHEN status = 'approved' THEN 1 END) as approved,
			COUNT(CASE WHEN status = 'rejected' THEN 1 END) as rejected
		FROM email_submissions
	`

	var stats models.EmailSubmissionStats
	err = h.db.QueryRow(statsQuery).Scan(&stats.Total, &stats.Pending, &stats.Approved, &stats.Rejected)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"submissions": submissions,
			"pagination": gin.H{
				"page":        page,
				"page_size":   pageSize,
				"total":       total,
				"total_pages": (total + pageSize - 1) / pageSize,
			},
			"stats": stats,
		},
	})
}

// CreateEmailSubmission - Create new email submission
func (h *EmailSubmissionHandler) CreateEmailSubmission(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req models.EmailSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user is member of the group
	var isMember bool
	err := h.db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)",
		req.GroupID, userID,
	).Scan(&isMember)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check group membership"})
		return
	}

	if !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this group"})
		return
	}

	// Check if submission already exists for this user and group
	var existingID uuid.UUID
	err = h.db.QueryRow(
		"SELECT id FROM email_submissions WHERE user_id = $1 AND group_id = $2",
		userID, req.GroupID,
	).Scan(&existingID)

	if err == nil {
		// Update existing submission
		_, err = h.db.Exec(`
			UPDATE email_submissions 
			SET email = $1, username = $2, full_name = $3, app_id = $4, status = 'pending', updated_at = CURRENT_TIMESTAMP
			WHERE id = $5
		`, req.Email, req.Username, req.FullName, req.AppID, existingID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update email submission"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Email submission updated successfully",
			"data":    gin.H{"id": existingID},
		})
		return
	}

	// Create new submission
	submissionID := uuid.New()
	_, err = h.db.Exec(`
		INSERT INTO email_submissions (id, user_id, group_id, app_id, email, username, full_name, status, submitted_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, submissionID, userID, req.GroupID, req.AppID, req.Email, req.Username, req.FullName)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create email submission"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Email submission created successfully",
		"data":    gin.H{"id": submissionID},
	})
}

// UpdateEmailSubmissionStatus - Update email submission status (admin only)
func (h *EmailSubmissionHandler) UpdateEmailSubmissionStatus(c *gin.Context) {
	submissionID := c.Param("id")
	reviewerID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req models.EmailSubmissionStatusUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update submission status
	_, err := h.db.Exec(`
		UPDATE email_submissions 
		SET status = $1, reviewed_at = CURRENT_TIMESTAMP, reviewed_by = $2, notes = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4
	`, req.Status, reviewerID, req.Notes, submissionID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update email submission status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Email submission status updated successfully",
	})
}

// GetEmailSubmission - Get single email submission by ID
func (h *EmailSubmissionHandler) GetEmailSubmission(c *gin.Context) {
	submissionID := c.Param("id")

	query := `
		SELECT 
			es.id, es.user_id, es.group_id, es.app_id, es.email, es.username, es.full_name,
			es.status, es.submitted_at, es.reviewed_at, es.reviewed_by, es.notes,
			es.created_at, es.updated_at,
			u.id, u.full_name, u.email,
			g.id, g.name, g.app_id,
			a.id, a.name, a.icon_url
		FROM email_submissions es
		LEFT JOIN users u ON es.user_id = u.id
		LEFT JOIN groups g ON es.group_id = g.id
		LEFT JOIN apps a ON es.app_id = a.id
		WHERE es.id = $1
	`

	var submission models.EmailSubmissionResponse
	var user models.User
	var group models.Group
	var app models.App

	err := h.db.QueryRow(query, submissionID).Scan(
		&submission.ID, &submission.UserID, &submission.GroupID, &submission.AppID,
		&submission.Email, &submission.Username, &submission.FullName,
		&submission.Status, &submission.SubmittedAt, &submission.ReviewedAt, &submission.ReviewedBy, &submission.Notes,
		&submission.CreatedAt, &submission.UpdatedAt,
		&user.ID, &user.FullName, &user.Email,
		&group.ID, &group.Name, &group.AppID,
		&app.ID, &app.Name, &app.IconURL,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Email submission not found"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch email submission"})
		return
	}

	submission.User = &user
	submission.Group = &group
	submission.App = &app

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    submission,
	})
}

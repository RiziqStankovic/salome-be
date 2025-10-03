package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"salome-be/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type NotificationHandler struct {
	db *sql.DB
}

func NewNotificationHandler(db *sql.DB) *NotificationHandler {
	return &NotificationHandler{db: db}
}

// GetUserNotifications mendapatkan notifikasi user
func (h *NotificationHandler) GetUserNotifications(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	// Get query parameters
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	// Build query - always get all notifications for frontend filtering
	query := `
		SELECT id, type, title, message, is_read, action_url, action_text, created_at
		FROM notifications 
		WHERE user_id = $1
	`
	args := []interface{}{userID}
	argIndex := 2

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)

	pageInt := 1
	if p, err := strconv.Atoi(page); err == nil && p > 0 {
		pageInt = p
	}
	pageSizeInt := 20
	if ps, err := strconv.Atoi(pageSize); err == nil && ps > 0 && ps <= 100 {
		pageSizeInt = ps
	}

	offset := (pageInt - 1) * pageSizeInt
	args = append(args, pageSizeInt, offset)

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}
	defer rows.Close()

	var notifications []models.NotificationResponse
	for rows.Next() {
		var notification models.NotificationResponse
		var actionURL, actionText sql.NullString

		err := rows.Scan(
			&notification.ID,
			&notification.Type,
			&notification.Title,
			&notification.Message,
			&notification.IsRead,
			&actionURL,
			&actionText,
			&notification.CreatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan notification"})
			return
		}

		if actionURL.Valid {
			notification.ActionURL = &actionURL.String
		}
		if actionText.Valid {
			notification.ActionText = &actionText.String
		}

		// Debug logging for date
		fmt.Printf("Notification %s - CreatedAt: %v (Type: %T)\n", notification.ID, notification.CreatedAt, notification.CreatedAt)

		notifications = append(notifications, notification)
	}

	// Get unread count
	var unreadCount int
	err = h.db.QueryRow("SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false", userID).Scan(&unreadCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count unread notifications"})
		return
	}

	// Get total count
	var total int
	err = h.db.QueryRow("SELECT COUNT(*) FROM notifications WHERE user_id = $1", userID).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count total notifications"})
		return
	}

	response := models.GetNotificationsResponse{
		Notifications: notifications,
		UnreadCount:   unreadCount,
		Total:         total,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Notifications retrieved successfully",
		"data":    response,
	})
}

// MarkAsRead menandai notifikasi sebagai sudah dibaca
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	var req models.MarkAsReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.db.Exec(`
		UPDATE notifications 
		SET is_read = true, updated_at = $1 
		WHERE id = $2
	`, time.Now(), req.NotificationID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notification as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Notification marked as read",
	})
}

// MarkAllAsRead menandai semua notifikasi user sebagai sudah dibaca
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	_, err := h.db.Exec(`
		UPDATE notifications 
		SET is_read = true, updated_at = $1 
		WHERE user_id = $2 AND is_read = false
	`, time.Now(), userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark all notifications as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "All notifications marked as read",
	})
}

// CreateNotification membuat notifikasi baru (untuk admin)
func (h *NotificationHandler) CreateNotification(c *gin.Context) {
	var req models.CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notificationID := uuid.New().String()
	now := time.Now()

	_, err := h.db.Exec(`
		INSERT INTO notifications (id, user_id, type, title, message, action_url, action_text, is_read, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, notificationID, req.UserID, req.Type, req.Title, req.Message, req.ActionURL, req.ActionText, false, now, now)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create notification"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Notification created successfully",
		"data": gin.H{
			"id": notificationID,
		},
	})
}

// DeleteNotification menghapus notifikasi
func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
	notificationID := c.Param("notification_id")
	if notificationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Notification ID is required"})
		return
	}

	// Check if notification exists and belongs to user
	var userID string
	err := h.db.QueryRow("SELECT user_id FROM notifications WHERE id = $1", notificationID).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check notification"})
		return
	}

	// Delete notification
	_, err = h.db.Exec("DELETE FROM notifications WHERE id = $1", notificationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete notification"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Notification deleted successfully",
	})
}

// CreateWelcomeNotification membuat notifikasi welcome untuk user baru
func (h *NotificationHandler) CreateWelcomeNotification(userID string) error {
	notificationID := uuid.New().String()
	now := time.Now()
	actionURL := "/browse"
	actionText := "Jelajahi Apps"

	_, err := h.db.Exec(`
		INSERT INTO notifications (id, user_id, type, title, message, action_url, action_text, is_read, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, notificationID, userID, "welcome", "Selamat datang di Salome! 🎉",
		"Terima kasih telah bergabung dengan kami. Mulai jelajahi grup patungan yang tersedia.",
		actionURL, actionText, false, now, now)

	return err
}

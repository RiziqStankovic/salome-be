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

type MessageHandler struct {
	db *sql.DB
}

func NewMessageHandler(db *sql.DB) *MessageHandler {
	return &MessageHandler{db: db}
}

// GetGroupMessages retrieves messages for a specific group
func (h *MessageHandler) GetGroupMessages(c *gin.Context) {
	groupID := c.Param("groupId")
	if groupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Group ID is required"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Check if user is admin
	var isAdmin bool
	err := h.db.QueryRow(`
		SELECT is_admin FROM users WHERE id = $1
	`, userID.(uuid.UUID)).Scan(&isAdmin)
	if err != nil {
		fmt.Printf("Error checking admin role for user %v in messages: %v\n", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check user role"})
		return
	}

	fmt.Printf("User %v is admin for messages: %v\n", userID, isAdmin)

	// If not admin, check if user is member of the group
	if !isAdmin {
		var isMember bool
		err := h.db.QueryRow(`
			SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)
		`, groupID, userID.(uuid.UUID)).Scan(&isMember)

		if err != nil {
			fmt.Printf("Error checking membership for user %v in group %s for messages: %v\n", userID, groupID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check membership"})
			return
		}

		if !isMember {
			fmt.Printf("User %v is not a member of group %s for messages\n", userID, groupID)
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
	}

	// Get pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	offset := (page - 1) * pageSize

	// Query messages with user information
	query := `
		SELECT 
			gm.id, gm.group_id, gm.user_id, gm.message, gm.message_type, gm.created_at,
			u.full_name, u.avatar_url
		FROM group_messages gm
		JOIN users u ON gm.user_id = u.id
		WHERE gm.group_id = $1
		ORDER BY gm.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := h.db.Query(query, groupID, pageSize, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}
	defer rows.Close()

	var messages []models.GroupMessageResponse
	for rows.Next() {
		var msg models.GroupMessageResponse
		var user models.UserResponse
		err := rows.Scan(
			&msg.ID, &msg.GroupID, &msg.UserID, &msg.Message, &msg.MessageType, &msg.CreatedAt,
			&user.FullName, &user.AvatarURL,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan message"})
			return
		}
		msg.User = user
		messages = append(messages, msg)
	}

	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM group_messages WHERE group_id = $1`
	err = h.db.QueryRow(countQuery, groupID).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count messages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"messages":    messages,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + pageSize - 1) / pageSize,
	})
}

// CreateGroupMessage creates a new message in a group
func (h *MessageHandler) CreateGroupMessage(c *gin.Context) {
	groupID := c.Param("groupId")
	if groupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Group ID is required"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req models.GroupMessageCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user is admin
	var isAdmin bool
	err := h.db.QueryRow(`
		SELECT is_admin FROM users WHERE id = $1
	`, userID.(uuid.UUID)).Scan(&isAdmin)
	if err != nil {
		fmt.Printf("Error checking admin role for user %v in CreateGroupMessage: %v\n", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check user role"})
		return
	}

	fmt.Printf("User %v is admin for CreateGroupMessage: %v\n", userID, isAdmin)

	// If not admin, verify user is a member of the group
	if !isAdmin {
		var isMember bool
		memberQuery := `SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)`
		err := h.db.QueryRow(memberQuery, groupID, userID.(uuid.UUID)).Scan(&isMember)
		if err != nil {
			fmt.Printf("Error checking membership for user %v in group %s for CreateGroupMessage: %v\n", userID, groupID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check membership"})
			return
		}

		if !isMember {
			fmt.Printf("User %v is not a member of group %s for CreateGroupMessage\n", userID, groupID)
			c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this group"})
			return
		}
	}

	// Set default message type if not provided
	if req.MessageType == "" {
		req.MessageType = "text"
	}

	// Insert message
	query := `
		INSERT INTO group_messages (id, group_id, user_id, message, message_type, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	messageID := uuid.New().String()
	var createdAt time.Time
	err = h.db.QueryRow(query, messageID, groupID, userID.(uuid.UUID), req.Message, req.MessageType, time.Now()).Scan(&messageID, &createdAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create message"})
		return
	}

	// Get user information for response
	var user models.UserResponse
	userQuery := `SELECT id, full_name, avatar_url FROM users WHERE id = $1`
	err = h.db.QueryRow(userQuery, userID.(uuid.UUID)).Scan(&user.ID, &user.FullName, &user.AvatarURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user information"})
		return
	}

	response := models.GroupMessageResponse{
		ID:          messageID,
		GroupID:     groupID,
		UserID:      userID.(uuid.UUID),
		Message:     req.Message,
		MessageType: req.MessageType,
		CreatedAt:   createdAt,
		User:        user,
	}

	c.JSON(http.StatusCreated, response)
}

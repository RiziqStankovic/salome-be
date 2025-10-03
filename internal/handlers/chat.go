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

type ChatHandler struct {
	db *sql.DB
}

func NewChatHandler(db *sql.DB) *ChatHandler {
	return &ChatHandler{db: db}
}

// CreateChat - Create a new chat (for anonymous or logged-in user)
func (h *ChatHandler) CreateChat(c *gin.Context) {
	var req models.ChatCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var userID *uuid.UUID
	var senderType string
	var anonymousName *string

	// Check if user is authenticated
	if id, exists := c.Get("user_id"); exists {
		uID := id.(uuid.UUID)
		userID = &uID
		senderType = "user"
	} else {
		if req.AnonymousName == nil || *req.AnonymousName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Anonymous name is required for anonymous chat"})
			return
		}
		anonymousName = req.AnonymousName
		senderType = "anonymous"
	}

	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin transaction"})
		return
	}
	defer tx.Rollback()

	// Create new chat
	newChatID := uuid.New()
	_, err = tx.Exec(`
		INSERT INTO chats (id, user_id, anonymous_name, status, is_read, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, newChatID, userID, anonymousName, "open", false, time.Now(), time.Now())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chat"})
		return
	}

	// Create initial message
	newMessageID := uuid.New()
	_, err = tx.Exec(`
		INSERT INTO messages (id, chat_id, sender_id, sender_type, content, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, newMessageID, newChatID, userID, senderType, req.Content, time.Now())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create initial message"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Fetch the created chat and its first message for response
	var chat models.Chat
	var message models.Message

	chatRow := h.db.QueryRow(`
		SELECT id, user_id, anonymous_name, status, created_at, updated_at FROM chats WHERE id = $1
	`, newChatID)
	err = chatRow.Scan(&chat.ID, &chat.UserID, &chat.AnonymousName, &chat.Status, &chat.CreatedAt, &chat.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch created chat"})
		return
	}

	messageRow := h.db.QueryRow(`
		SELECT id, chat_id, sender_id, sender_type, content, created_at FROM messages WHERE id = $1
	`, newMessageID)
	err = messageRow.Scan(&message.ID, &message.ChatID, &message.SenderID, &message.SenderType, &message.Content, &message.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch initial message"})
		return
	}

	chatResponse := models.ChatResponse{
		ID:            chat.ID,
		UserID:        chat.UserID,
		AnonymousName: chat.AnonymousName,
		Status:        chat.Status,
		CreatedAt:     chat.CreatedAt,
		UpdatedAt:     chat.UpdatedAt,
		LastMessage: &models.MessageResponse{
			ID:         message.ID,
			ChatID:     message.ChatID,
			SenderID:   message.SenderID,
			SenderType: message.SenderType,
			Content:    message.Content,
			CreatedAt:  message.CreatedAt,
		},
	}

	c.JSON(http.StatusCreated, gin.H{"chat": chatResponse})
}

// GetUserChats - Get all chats for a logged-in user
func (h *ChatHandler) GetUserChats(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	rows, err := h.db.Query(`
		SELECT c.id, c.user_id, c.anonymous_name, c.status, c.is_read, c.created_at, c.updated_at,
			lm.id, lm.chat_id, lm.sender_id, lm.sender_type, lm.content, lm.created_at,
			u.full_name as sender_name,
			(SELECT COUNT(*) FROM messages WHERE chat_id = c.id) as message_count
		FROM chats c
		LEFT JOIN LATERAL (
			SELECT id, chat_id, sender_id, sender_type, content, created_at
			FROM messages 
			WHERE chat_id = c.id 
			ORDER BY created_at DESC 
			LIMIT 1
		) lm ON true
		LEFT JOIN users u ON c.user_id = u.id
		WHERE c.user_id = $1
		ORDER BY lm.created_at DESC, c.created_at DESC
	`, userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user chats"})
		return
	}
	defer rows.Close()

	chatsMap := make(map[uuid.UUID]models.ChatResponse)
	for rows.Next() {
		var chat models.Chat
		var messageID, messageChatID, messageSenderID sql.NullString
		var messageSenderType, messageContent sql.NullString
		var messageCreatedAt sql.NullTime
		var senderName sql.NullString
		var messageCount int

		err := rows.Scan(
			&chat.ID, &chat.UserID, &chat.AnonymousName, &chat.Status, &chat.IsRead, &chat.CreatedAt, &chat.UpdatedAt,
			&messageID, &messageChatID, &messageSenderID, &messageSenderType, &messageContent, &messageCreatedAt,
			&senderName, &messageCount,
		)
		if err != nil {
			fmt.Printf("Error scanning chat row: %v\n", err)
			continue
		}

		chatResponse, exists := chatsMap[chat.ID]
		if !exists {
			// Set display name based on sender_name or anonymous_name
			var displayName *string
			if senderName.Valid && senderName.String != "" {
				displayNameStr := "user - " + senderName.String
				displayName = &displayNameStr
			} else if chat.AnonymousName != nil && *chat.AnonymousName != "" {
				displayNameStr := "anony - " + *chat.AnonymousName
				displayName = &displayNameStr
			}

			chatResponse = models.ChatResponse{
				ID:            chat.ID,
				UserID:        chat.UserID,
				AnonymousName: chat.AnonymousName,
				Status:        chat.Status,
				IsRead:        chat.IsRead,
				MessageCount:  messageCount,
				CreatedAt:     chat.CreatedAt,
				UpdatedAt:     chat.UpdatedAt,
				SenderName:    displayName,
			}
		}

		if messageID.Valid {
			msgSenderID := uuid.Nil
			if messageSenderID.Valid {
				msgSenderID = uuid.MustParse(messageSenderID.String)
			}
			currentMessage := &models.MessageResponse{
				ID:         uuid.MustParse(messageID.String),
				ChatID:     uuid.MustParse(messageChatID.String),
				SenderID:   &msgSenderID,
				SenderType: messageSenderType.String,
				Content:    messageContent.String,
				CreatedAt:  messageCreatedAt.Time,
			}
			if chatResponse.LastMessage == nil || currentMessage.CreatedAt.After(chatResponse.LastMessage.CreatedAt) {
				chatResponse.LastMessage = currentMessage
			}
		}
		chatsMap[chat.ID] = chatResponse
	}

	var chats []models.ChatResponse
	for _, chat := range chatsMap {
		chats = append(chats, chat)
	}

	c.JSON(http.StatusOK, gin.H{"chats": chats})
}

// GetChatMessages - Get messages for a specific chat
func (h *ChatHandler) GetChatMessages(c *gin.Context) {
	chatIDStr := c.Param("id")
	chatID, err := uuid.Parse(chatIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	// Check if the user is authorized to view this chat
	var isAuthorized bool
	var chatUserID *uuid.UUID
	var chatAnonymousName *string

	err = h.db.QueryRow("SELECT user_id, anonymous_name FROM chats WHERE id = $1", chatID).Scan(&chatUserID, &chatAnonymousName)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chat details"})
		return
	}

	// If logged in, check if user_id matches
	if userID, exists := c.Get("user_id"); exists {
		if chatUserID != nil && *chatUserID == userID.(uuid.UUID) {
			isAuthorized = true
		}
	} else { // For anonymous users, we might need a session ID or similar to authorize, for now, assume public if no user_id
		// For simplicity, if no user_id is present, and chat is anonymous, allow access.
		// In a real app, you'd use a session token for anonymous chats.
		if chatUserID == nil {
			isAuthorized = true
		}
	}

	// Admin check
	if !isAuthorized {
		// Check if user is admin
		if userID, exists := c.Get("user_id"); exists {
			var isAdmin bool
			err = h.db.QueryRow("SELECT is_admin FROM users WHERE id = $1", userID).Scan(&isAdmin)
			if err == nil && isAdmin {
				isAuthorized = true
			}
		}
	}

	if !isAuthorized {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to view this chat"})
		return
	}

	rows, err := h.db.Query(`
		SELECT m.id, m.chat_id, m.sender_id, m.sender_type, m.content, m.created_at, u.full_name, c.anonymous_name
		FROM messages m
		LEFT JOIN users u ON m.sender_id = u.id
		LEFT JOIN chats c ON m.chat_id = c.id
		WHERE m.chat_id = $1
		ORDER BY m.created_at ASC
	`, chatID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}
	defer rows.Close()

	var messages []models.MessageResponse
	for rows.Next() {
		var msg models.MessageResponse
		var senderID sql.NullString
		var senderName sql.NullString

		var anonymousName sql.NullString
		err := rows.Scan(&msg.ID, &msg.ChatID, &senderID, &msg.SenderType, &msg.Content, &msg.CreatedAt, &senderName, &anonymousName)
		if err != nil {
			fmt.Printf("Error scanning message row: %v\n", err)
			continue
		}

		if senderID.Valid {
			sID := uuid.MustParse(senderID.String)
			msg.SenderID = &sID
		}
		// Set sender name based on sender type
		if msg.SenderType == "user" && senderName.Valid {
			msg.SenderName = &senderName.String
		} else if msg.SenderType == "anonymous" && anonymousName.Valid {
			msg.SenderName = &anonymousName.String
		} else if msg.SenderType == "admin" {
			adminName := "Admin"
			msg.SenderName = &adminName
		}

		messages = append(messages, msg)
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

// SendMessage - Send a message to an existing chat
func (h *ChatHandler) SendMessage(c *gin.Context) {
	chatIDStr := c.Param("id")
	chatID, err := uuid.Parse(chatIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	var req models.MessageSendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var userID *uuid.UUID
	var senderType string
	var chatUserID *uuid.UUID
	var chatAnonymousName *string

	// Check if user is authenticated
	if id, exists := c.Get("user_id"); exists {
		uID := id.(uuid.UUID)
		userID = &uID
		senderType = "user"
	} else {
		senderType = "anonymous"
	}

	// Check if the user is authorized to send message to this chat
	err = h.db.QueryRow("SELECT user_id, anonymous_name FROM chats WHERE id = $1", chatID).Scan(&chatUserID, &chatAnonymousName)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chat details"})
		return
	}

	// Authorization logic
	isAuthorized := false
	if senderType == "user" && chatUserID != nil && *chatUserID == *userID {
		isAuthorized = true
	} else if senderType == "anonymous" && chatUserID == nil {
		// For anonymous, we need a way to link them to their previous chat.
		// This would typically involve a session ID stored in a cookie/local storage.
		// For now, if it's an anonymous chat, and the sender is anonymous, allow.
		// TODO: Implement robust anonymous session management.
		isAuthorized = true
	}

	// Admin check
	if !isAuthorized {
		// Check if user is admin
		if userID != nil {
			var isAdmin bool
			err = h.db.QueryRow("SELECT is_admin FROM users WHERE id = $1", *userID).Scan(&isAdmin)
			if err == nil && isAdmin {
				isAuthorized = true
				senderType = "admin"
			}
		}
	}

	if !isAuthorized {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to send messages to this chat"})
		return
	}

	newMessageID := uuid.New()
	_, err = h.db.Exec(`
		INSERT INTO messages (id, chat_id, sender_id, sender_type, content, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, newMessageID, chatID, userID, senderType, req.Content, time.Now())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	// Update chat's updated_at timestamp and mark as unread
	_, err = h.db.Exec("UPDATE chats SET updated_at = NOW(), is_read = false WHERE id = $1", chatID)
	if err != nil {
		fmt.Printf("Warning: Failed to update chat updated_at and is_read for chat %s: %v\n", chatID, err)
	}

	var message models.MessageResponse
	var senderName sql.NullString

	err = h.db.QueryRow(`
		SELECT m.id, m.chat_id, m.sender_id, m.sender_type, m.content, m.created_at, u.full_name
		FROM messages m
		LEFT JOIN users u ON m.sender_id = u.id
		WHERE m.id = $1
	`, newMessageID).Scan(&message.ID, &message.ChatID, &userID, &message.SenderType, &message.Content, &message.CreatedAt, &senderName)

	if err != nil {
		fmt.Printf("Error fetching sent message: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sent message"})
		return
	}

	if userID != nil {
		message.SenderID = userID
	}
	if senderName.Valid {
		message.SenderName = &senderName.String
	} else if message.SenderType == "anonymous" && chatAnonymousName != nil {
		message.SenderName = chatAnonymousName
	} else if message.SenderType == "admin" {
		adminName := "Admin"
		message.SenderName = &adminName
	}

	c.JSON(http.StatusCreated, gin.H{"message": message})
}

// GetAllChats - Get all chats (admin only)
func (h *ChatHandler) GetAllChats(c *gin.Context) {
	// TODO: Implement admin authorization check

	rows, err := h.db.Query(`
		SELECT c.id, c.user_id, c.anonymous_name, c.status, c.is_read, c.created_at, c.updated_at,
			lm.id, lm.chat_id, lm.sender_id, lm.sender_type, lm.content, lm.created_at,
			u.full_name as sender_name,
			(SELECT COUNT(*) FROM messages WHERE chat_id = c.id) as message_count
		FROM chats c
		LEFT JOIN LATERAL (
			SELECT id, chat_id, sender_id, sender_type, content, created_at
			FROM messages 
			WHERE chat_id = c.id 
			ORDER BY created_at DESC 
			LIMIT 1
		) lm ON true
		LEFT JOIN users u ON c.user_id = u.id
		ORDER BY lm.created_at DESC, c.created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch all chats"})
		return
	}
	defer rows.Close()

	chatsMap := make(map[uuid.UUID]models.ChatResponse)
	for rows.Next() {
		var chat models.Chat
		var messageID, messageChatID, messageSenderID sql.NullString
		var messageSenderType, messageContent sql.NullString
		var messageCreatedAt sql.NullTime
		var senderName sql.NullString
		var messageCount int

		err := rows.Scan(
			&chat.ID, &chat.UserID, &chat.AnonymousName, &chat.Status, &chat.IsRead, &chat.CreatedAt, &chat.UpdatedAt,
			&messageID, &messageChatID, &messageSenderID, &messageSenderType, &messageContent, &messageCreatedAt,
			&senderName, &messageCount,
		)
		if err != nil {
			fmt.Printf("Error scanning chat row for admin: %v\n", err)
			continue
		}

		chatResponse, exists := chatsMap[chat.ID]
		if !exists {
			// Set display name based on sender_name or anonymous_name
			var displayName *string
			if senderName.Valid && senderName.String != "" {
				displayNameStr := "user - " + senderName.String
				displayName = &displayNameStr
			} else if chat.AnonymousName != nil && *chat.AnonymousName != "" {
				displayNameStr := "anony - " + *chat.AnonymousName
				displayName = &displayNameStr
			}

			chatResponse = models.ChatResponse{
				ID:            chat.ID,
				UserID:        chat.UserID,
				AnonymousName: chat.AnonymousName,
				Status:        chat.Status,
				IsRead:        chat.IsRead,
				MessageCount:  messageCount,
				CreatedAt:     chat.CreatedAt,
				UpdatedAt:     chat.UpdatedAt,
				SenderName:    displayName,
			}
		}

		if messageID.Valid {
			msgSenderID := uuid.Nil
			if messageSenderID.Valid {
				msgSenderID = uuid.MustParse(messageSenderID.String)
			}
			currentMessage := &models.MessageResponse{
				ID:         uuid.MustParse(messageID.String),
				ChatID:     uuid.MustParse(messageChatID.String),
				SenderID:   &msgSenderID,
				SenderType: messageSenderType.String,
				Content:    messageContent.String,
				CreatedAt:  messageCreatedAt.Time,
			}
			if chatResponse.LastMessage == nil || currentMessage.CreatedAt.After(chatResponse.LastMessage.CreatedAt) {
				chatResponse.LastMessage = currentMessage
			}
		}
		chatsMap[chat.ID] = chatResponse
	}

	var chats []models.ChatResponse
	for _, chat := range chatsMap {
		chats = append(chats, chat)
	}

	c.JSON(http.StatusOK, gin.H{"chats": chats})
}

// SendMessageAsAdmin - Send a message to an existing chat as admin
func (h *ChatHandler) SendMessageAsAdmin(c *gin.Context) {
	chatIDStr := c.Param("id")
	chatID, err := uuid.Parse(chatIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	var req models.MessageSendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Admin not authenticated"})
		return
	}

	// Check if chat exists
	var chatExists bool
	err = h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM chats WHERE id = $1)", chatID).Scan(&chatExists)
	if err != nil || !chatExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found"})
		return
	}

	newMessageID := uuid.New()
	_, err = h.db.Exec(`
		INSERT INTO messages (id, chat_id, sender_id, sender_type, content, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, newMessageID, chatID, adminID.(uuid.UUID), "admin", req.Content, time.Now())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message as admin"})
		return
	}

	// Update chat's updated_at timestamp
	_, err = h.db.Exec("UPDATE chats SET updated_at = NOW() WHERE id = $1", chatID)
	if err != nil {
		fmt.Printf("Warning: Failed to update chat updated_at for chat %s: %v\n", chatID, err)
	}

	var message models.MessageResponse
	adminName := "Admin"
	err = h.db.QueryRow(`
		SELECT id, chat_id, sender_id, sender_type, content, created_at FROM messages WHERE id = $1
	`, newMessageID).Scan(&message.ID, &message.ChatID, &message.SenderID, &message.SenderType, &message.Content, &message.CreatedAt)

	if err != nil {
		fmt.Printf("Error fetching sent admin message: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sent admin message"})
		return
	}
	message.SenderName = &adminName

	c.JSON(http.StatusCreated, gin.H{"message": message})
}

// UpdateChatStatus - Update the status of a chat (admin only)
func (h *ChatHandler) UpdateChatStatus(c *gin.Context) {
	chatIDStr := c.Param("id")
	chatID, err := uuid.Parse(chatIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate status
	validStatuses := map[string]bool{"open": true, "closed": true, "pending": true}
	if !validStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat status"})
		return
	}

	_, err = h.db.Exec("UPDATE chats SET status = $1, updated_at = NOW() WHERE id = $2", req.Status, chatID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update chat status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Chat status updated successfully"})
}

// SendBroadcast - Send a broadcast message to all or selected users (admin only)
func (h *ChatHandler) SendBroadcast(c *gin.Context) {
	adminID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Admin not authenticated"})
		return
	}

	var req struct {
		Content string   `json:"content" binding:"required"`
		UserIDs []string `json:"user_ids,omitempty"` // If empty, broadcast to all users
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get target users
	var targetUserIDs []uuid.UUID
	if len(req.UserIDs) > 0 {
		// Broadcast to selected users
		for _, userIDStr := range req.UserIDs {
			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID: " + userIDStr})
				return
			}
			targetUserIDs = append(targetUserIDs, userID)
		}
	} else {
		// Broadcast to all users
		rows, err := h.db.Query("SELECT id FROM users WHERE is_admin = false")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch all users"})
			return
		}
		defer rows.Close()

		for rows.Next() {
			var userID uuid.UUID
			if err := rows.Scan(&userID); err != nil {
				fmt.Printf("Error scanning user ID: %v\n", err)
				continue
			}
			targetUserIDs = append(targetUserIDs, userID)
		}
	}

	// Create broadcast messages for each target user
	successCount := 0
	errorCount := 0

	for _, userID := range targetUserIDs {
		// Find or create a chat for this user
		var chatID uuid.UUID
		err := h.db.QueryRow(`
			SELECT id FROM chats 
			WHERE user_id = $1 AND status = 'open'
			ORDER BY created_at DESC 
			LIMIT 1
		`, userID).Scan(&chatID)

		if err == sql.ErrNoRows {
			// Create new chat for this user
			chatID = uuid.New()
			_, err = h.db.Exec(`
				INSERT INTO chats (id, user_id, status, is_read, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6)
			`, chatID, userID, "open", false, time.Now(), time.Now())
			if err != nil {
				fmt.Printf("Failed to create chat for user %s: %v\n", userID, err)
				errorCount++
				continue
			}
		} else if err != nil {
			fmt.Printf("Failed to find chat for user %s: %v\n", userID, err)
			errorCount++
			continue
		}

		// Send broadcast message
		messageID := uuid.New()
		_, err = h.db.Exec(`
			INSERT INTO messages (id, chat_id, sender_id, sender_type, content, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, messageID, chatID, adminID.(uuid.UUID), "admin", req.Content, time.Now())

		if err != nil {
			fmt.Printf("Failed to send broadcast message to user %s: %v\n", userID, err)
			errorCount++
		} else {
			successCount++
		}

		// Update chat's updated_at timestamp
		_, err = h.db.Exec("UPDATE chats SET updated_at = NOW() WHERE id = $1", chatID)
		if err != nil {
			fmt.Printf("Warning: Failed to update chat updated_at for chat %s: %v\n", chatID, err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Broadcast completed",
		"success_count": successCount,
		"error_count":   errorCount,
		"total_targets": len(targetUserIDs),
	})
}

// MarkChatAsRead - Mark a chat as read by admin
func (h *ChatHandler) MarkChatAsRead(c *gin.Context) {
	chatIDStr := c.Param("id")
	chatID, err := uuid.Parse(chatIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	// Update chat as read
	_, err = h.db.Exec(`
		UPDATE chats 
		SET is_read = true, updated_at = NOW()
		WHERE id = $1
	`, chatID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark chat as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Chat marked as read"})
}

// MarkChatAsUnread - Mark a chat as unread by admin
func (h *ChatHandler) MarkChatAsUnread(c *gin.Context) {
	chatIDStr := c.Param("id")
	chatID, err := uuid.Parse(chatIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	// Update chat as unread
	_, err = h.db.Exec(`
		UPDATE chats 
		SET is_read = false, updated_at = NOW()
		WHERE id = $1
	`, chatID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark chat as unread"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Chat marked as unread"})
}

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

type AccountCredentialsHandler struct {
	db *sql.DB
}

func NewAccountCredentialsHandler(db *sql.DB) *AccountCredentialsHandler {
	return &AccountCredentialsHandler{db: db}
}

// GetUserAccountCredentials gets all account credentials for a user
func (h *AccountCredentialsHandler) GetUserAccountCredentials(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Convert userID to UUID
	var userIDUUID uuid.UUID
	var err error
	switch v := userID.(type) {
	case string:
		userIDUUID, err = uuid.Parse(v)
		if err != nil {
			fmt.Printf("Invalid userID string: %v\n", v)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
			return
		}
	case uuid.UUID:
		userIDUUID = v
	default:
		fmt.Printf("Unexpected userID type in GetUserAccountCredentials: %T, value: %v\n", userID, userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	query := `
		SELECT 
			ac.id, ac.user_id, ac.group_id, ac.username, ac.email, ac.description,
			ac.created_at, ac.updated_at,
			g.name as group_name
		FROM account_credentials ac
		LEFT JOIN groups g ON ac.group_id = g.id
		WHERE ac.user_id = $1
		ORDER BY ac.updated_at DESC
	`

	fmt.Printf("Fetching account credentials for user: %s (type: %T)\n", userIDUUID.String(), userIDUUID)
	fmt.Printf("Query: %s\n", query)
	fmt.Printf("Parameter: %v (type: %T)\n", userIDUUID.String(), userIDUUID.String())
	rows, err := h.db.Query(query, userIDUUID.String())
	if err != nil {
		fmt.Printf("Error fetching account credentials: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch account credentials", "details": err.Error()})
		return
	}
	defer rows.Close()

	var credentials []models.UserAppCredentialsResponse
	for rows.Next() {
		var cred models.UserAppCredentialsResponse
		var group models.Group
		var groupName sql.NullString
		var userIDStr, groupIDStr string

		err := rows.Scan(
			&cred.ID, &userIDStr, &groupIDStr, &cred.Username, &cred.Email, &cred.Description,
			&cred.CreatedAt, &cred.UpdatedAt,
			&groupName,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan account credentials"})
			return
		}

		// Convert string to UUID
		cred.UserID, err = uuid.Parse(userIDStr)
		if err != nil {
			fmt.Printf("Invalid userID in database: %s\n", userIDStr)
			continue
		}

		cred.GroupID, err = uuid.Parse(groupIDStr)
		if err != nil {
			fmt.Printf("Invalid groupID in database: %s\n", groupIDStr)
			continue
		}

		if groupName.Valid {
			group.Name = groupName.String
			cred.Group = &group
		}

		credentials = append(credentials, cred)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    credentials,
	})
}

// CreateOrUpdateAccountCredentials creates or updates account credentials
func (h *AccountCredentialsHandler) CreateOrUpdateAccountCredentials(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req models.UserAppCredentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert GroupID to UUID first
	groupIDUUID, err := uuid.Parse(req.GroupID)
	if err != nil {
		fmt.Printf("Invalid GroupID string: %v\n", req.GroupID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID format"})
		return
	}

	// Validate group exists
	checkGroupQuery := `SELECT id FROM groups WHERE id = $1`
	var groupExists string

	fmt.Printf("Checking if group exists - GroupID: %s\n", groupIDUUID.String())

	err = h.db.QueryRow(checkGroupQuery, groupIDUUID).Scan(&groupExists)
	if err != nil {
		fmt.Printf("Error checking group existence: %v\n", err)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check group existence", "details": err.Error()})
		return
	}

	// Convert userID to UUID
	var userIDUUID uuid.UUID
	switch v := userID.(type) {
	case string:
		userIDUUID, err = uuid.Parse(v)
		if err != nil {
			fmt.Printf("Invalid userID string: %v\n", v)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
			return
		}
	case uuid.UUID:
		userIDUUID = v
	default:
		fmt.Printf("Unexpected userID type: %T, value: %v\n", userID, userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	fmt.Printf("UserID converted to UUID: %s\n", userIDUUID.String())

	// Check if credentials already exist for this group
	var existingID string
	checkQuery := `SELECT id FROM account_credentials WHERE user_id = $1 AND group_id = $2`
	err = h.db.QueryRow(checkQuery, userIDUUID.String(), groupIDUUID.String()).Scan(&existingID)

	if err == sql.ErrNoRows {
		// Create new credentials
		credID := uuid.New()
		insertQuery := `
			INSERT INTO account_credentials (id, user_id, group_id, username, email, description, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		_, err = h.db.Exec(insertQuery, credID, userIDUUID.String(), groupIDUUID.String(), req.Username, req.Email, req.Description, time.Now(), time.Now())
		if err != nil {
			fmt.Printf("Error creating credentials: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create account credentials", "details": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"message": "Account credentials created successfully",
			"data":    gin.H{"id": credID},
		})
	} else if err != nil {
		fmt.Printf("Error checking existing credentials: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing credentials", "details": err.Error()})
		return
	} else {
		// Update existing credentials
		updateQuery := `
			UPDATE account_credentials 
			SET username = $1, email = $2, description = $3, updated_at = $4
			WHERE id = $5
		`
		_, err = h.db.Exec(updateQuery, req.Username, req.Email, req.Description, time.Now(), existingID)
		if err != nil {
			fmt.Printf("Error updating credentials: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update account credentials", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Account credentials updated successfully",
			"data":    gin.H{"id": existingID},
		})
	}
}

// GetAccountCredentialsByGroup gets account credentials for a specific group
func (h *AccountCredentialsHandler) GetAccountCredentialsByGroup(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Convert userID to UUID
	var userIDUUID uuid.UUID
	var err error
	switch v := userID.(type) {
	case string:
		userIDUUID, err = uuid.Parse(v)
		if err != nil {
			fmt.Printf("Invalid userID string: %v\n", v)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
			return
		}
	case uuid.UUID:
		userIDUUID = v
	default:
		fmt.Printf("Unexpected userID type in GetAccountCredentialsByGroup: %T, value: %v\n", userID, userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	groupID := c.Param("groupId")
	if groupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Group ID is required"})
		return
	}

	// Convert groupID to UUID
	groupIDUUID, err := uuid.Parse(groupID)
	if err != nil {
		fmt.Printf("Invalid groupID string: %v\n", groupID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID format"})
		return
	}

	query := `
		SELECT 
			ac.id, ac.user_id, ac.group_id, ac.username, ac.email, ac.description,
			ac.created_at, ac.updated_at,
			g.name as group_name
		FROM account_credentials ac
		LEFT JOIN groups g ON ac.group_id = g.id
		WHERE ac.user_id = $1 AND ac.group_id = $2
	`

	var cred models.UserAppCredentialsResponse
	var group models.Group
	var groupName sql.NullString
	var userIDStr, groupIDStr string

	fmt.Printf("Fetching account credentials for user: %s, group: %s\n", userIDUUID.String(), groupIDUUID.String())
	err = h.db.QueryRow(query, userIDUUID.String(), groupIDUUID.String()).Scan(
		&cred.ID, &userIDStr, &groupIDStr, &cred.Username, &cred.Email, &cred.Description,
		&cred.CreatedAt, &cred.UpdatedAt,
		&groupName,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account credentials not found"})
		return
	} else if err != nil {
		fmt.Printf("Error fetching account credentials by group: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch account credentials", "details": err.Error()})
		return
	}

	// Convert string to UUID
	cred.UserID, err = uuid.Parse(userIDStr)
	if err != nil {
		fmt.Printf("Invalid userID in database: %s\n", userIDStr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID in database"})
		return
	}

	cred.GroupID, err = uuid.Parse(groupIDStr)
	if err != nil {
		fmt.Printf("Invalid groupID in database: %s\n", groupIDStr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid group ID in database"})
		return
	}

	if groupName.Valid {
		group.Name = groupName.String
		cred.Group = &group
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    cred,
	})
}

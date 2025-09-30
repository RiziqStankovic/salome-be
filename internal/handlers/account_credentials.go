package handlers

import (
	"database/sql"
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

	query := `
		SELECT 
			ac.id, ac.user_id, ac.app_id, ac.username, ac.email, 
			ac.created_at, ac.updated_at,
			a.name as app_name, a.icon_url, a.description
		FROM account_credentials ac
		LEFT JOIN apps a ON ac.app_id = a.id
		WHERE ac.user_id = $1
		ORDER BY ac.updated_at DESC
	`

	rows, err := h.db.Query(query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch account credentials"})
		return
	}
	defer rows.Close()

	var credentials []models.UserAppCredentialsResponse
	for rows.Next() {
		var cred models.UserAppCredentialsResponse
		var app models.App
		var username, email sql.NullString
		var appName, iconURL, description sql.NullString

		err := rows.Scan(
			&cred.ID, &cred.UserID, &cred.AppID, &username, &email,
			&cred.CreatedAt, &cred.UpdatedAt,
			&appName, &iconURL, &description,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan account credentials"})
			return
		}

		if username.Valid {
			cred.Username = &username.String
		}
		if email.Valid {
			cred.Email = &email.String
		}

		if appName.Valid {
			app.Name = appName.String
			app.IconURL = iconURL.String
			app.Description = description.String
			cred.App = &app
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

	// Check if credentials already exist
	var existingID uuid.UUID
	checkQuery := `SELECT id FROM account_credentials WHERE user_id = $1 AND app_id = $2`
	err := h.db.QueryRow(checkQuery, userID, req.AppID).Scan(&existingID)

	if err == sql.ErrNoRows {
		// Create new credentials
		credID := uuid.New()
		insertQuery := `
			INSERT INTO account_credentials (id, user_id, app_id, username, email, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`
		_, err = h.db.Exec(insertQuery, credID, userID, req.AppID, req.Username, req.Email, time.Now(), time.Now())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create account credentials"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"message": "Account credentials created successfully",
			"data":    gin.H{"id": credID},
		})
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing credentials"})
		return
	} else {
		// Update existing credentials
		updateQuery := `
			UPDATE account_credentials 
			SET username = $1, email = $2, updated_at = $3
			WHERE id = $4
		`
		_, err = h.db.Exec(updateQuery, req.Username, req.Email, time.Now(), existingID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update account credentials"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Account credentials updated successfully",
			"data":    gin.H{"id": existingID},
		})
	}
}

// GetAccountCredentialsByApp gets account credentials for a specific app
func (h *AccountCredentialsHandler) GetAccountCredentialsByApp(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	appID := c.Param("appId")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "App ID is required"})
		return
	}

	query := `
		SELECT 
			ac.id, ac.user_id, ac.app_id, ac.username, ac.email, 
			ac.created_at, ac.updated_at,
			a.name as app_name, a.icon_url, a.description
		FROM account_credentials ac
		LEFT JOIN apps a ON ac.app_id = a.id
		WHERE ac.user_id = $1 AND ac.app_id = $2
	`

	var cred models.UserAppCredentialsResponse
	var app models.App
	var username, email sql.NullString
	var appName, iconURL, description sql.NullString

	err := h.db.QueryRow(query, userID, appID).Scan(
		&cred.ID, &cred.UserID, &cred.AppID, &username, &email,
		&cred.CreatedAt, &cred.UpdatedAt,
		&appName, &iconURL, &description,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account credentials not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch account credentials"})
		return
	}

	if username.Valid {
		cred.Username = &username.String
	}
	if email.Valid {
		cred.Email = &email.String
	}

	if appName.Valid {
		app.Name = appName.String
		app.IconURL = iconURL.String
		app.Description = description.String
		cred.App = &app
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    cred,
	})
}

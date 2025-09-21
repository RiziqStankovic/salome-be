package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"salome-be/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SubscriptionHandler struct {
	db *sql.DB
}

func NewSubscriptionHandler(db *sql.DB) *SubscriptionHandler {
	return &SubscriptionHandler{db: db}
}

// CreateSubscription creates a new subscription for a group
func (h *SubscriptionHandler) CreateSubscription(c *gin.Context) {
	groupIDStr := c.Param("groupId")
	groupID, err := uuid.Parse(groupIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req models.SubscriptionCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify user is owner of the group
	var ownerID uuid.UUID
	ownerQuery := `SELECT owner_id FROM groups WHERE id = $1`
	err = h.db.QueryRow(ownerQuery, groupID).Scan(&ownerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	if ownerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only group owner can create subscriptions"})
		return
	}

	// Check if subscription already exists for this group
	var existingID uuid.UUID
	checkQuery := `SELECT id FROM subscriptions WHERE group_id = $1`
	err = h.db.QueryRow(checkQuery, groupID).Scan(&existingID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Subscription already exists for this group"})
		return
	}

	// Create subscription
	subscriptionID := uuid.New()
	_, err = h.db.Exec(`
		INSERT INTO subscriptions (id, group_id, service_name, service_url, plan_name, price_per_month, currency, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, subscriptionID, groupID, req.ServiceName, req.ServiceURL, req.PlanName, req.PricePerMonth, req.Currency, "active", time.Now(), time.Now())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subscription"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":         "Subscription created successfully",
		"subscription_id": subscriptionID,
	})
}

// GetGroupSubscriptions retrieves subscriptions for a specific group
func (h *SubscriptionHandler) GetGroupSubscriptions(c *gin.Context) {
	groupIDStr := c.Param("groupId")
	groupID, err := uuid.Parse(groupIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Check if user is member of the group
	var isMember bool
	memberQuery := `SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)`
	err = h.db.QueryRow(memberQuery, groupID, userID).Scan(&isMember)
	if err != nil || !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this group"})
		return
	}

	// Query subscriptions
	query := `
		SELECT id, group_id, service_name, service_url, plan_name, price_per_month, 
			   currency, status, next_billing_date, created_at, updated_at
		FROM subscriptions
		WHERE group_id = $1
		ORDER BY created_at DESC
	`

	rows, err := h.db.Query(query, groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch subscriptions"})
		return
	}
	defer rows.Close()

	var subscriptions []models.Subscription
	for rows.Next() {
		var subscription models.Subscription
		err := rows.Scan(
			&subscription.ID, &subscription.GroupID, &subscription.ServiceName, &subscription.ServiceURL,
			&subscription.PlanName, &subscription.PricePerMonth, &subscription.Currency,
			&subscription.Status, &subscription.NextBillingDate, &subscription.CreatedAt, &subscription.UpdatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan subscription"})
			return
		}
		subscriptions = append(subscriptions, subscription)
	}

	c.JSON(http.StatusOK, gin.H{"subscriptions": subscriptions})
}

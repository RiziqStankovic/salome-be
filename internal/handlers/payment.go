package handlers

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"salome-be/internal/config"
	"salome-be/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PaymentHandler struct {
	db *sql.DB
}

func NewPaymentHandler(db *sql.DB) *PaymentHandler {
	return &PaymentHandler{db: db}
}

func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req models.PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get subscription details
	var subscription models.Subscription
	err := h.db.QueryRow(`
		SELECT id, group_id, service_name, price_per_month, currency
		FROM subscriptions WHERE id = $1
	`, req.SubscriptionID).Scan(&subscription.ID, &subscription.GroupID, &subscription.ServiceName, &subscription.PricePerMonth, &subscription.Currency)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		return
	}

	// Check if user is member of the group
	var isMember bool
	err = h.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)
	`, subscription.GroupID, userID).Scan(&isMember)

	if err != nil || !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Create payment record
	paymentID := uuid.New()
	_, err = h.db.Exec(`
		INSERT INTO payments (id, subscription_id, user_id, amount, currency, status, payment_method, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, paymentID, req.SubscriptionID, userID, req.Amount, subscription.Currency, "pending", req.PaymentMethod, time.Now(), time.Now())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment"})
		return
	}

	// For now, we'll simulate payment processing
	// TODO: Integrate with actual payment gateway (Midtrans, Xendit, etc.)

	// Update payment with mock transaction ID
	_, err = h.db.Exec(`
		UPDATE payments SET midtrans_transaction_id = $1, updated_at = $2 WHERE id = $3
	`, fmt.Sprintf("TXN_%s", paymentID.String()), time.Now(), paymentID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment"})
		return
	}

	response := models.MidtransResponse{
		Token:       fmt.Sprintf("https://payment.salome.id/redirect/%s", paymentID.String()),
		RedirectURL: fmt.Sprintf("https://payment.salome.id/redirect/%s", paymentID.String()),
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment created successfully",
		"payment": response,
	})
}

// CreateGroupPaymentLink creates a Midtrans payment link for group payment
func (h *PaymentHandler) CreateGroupPaymentLink(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		GroupID string  `json:"group_id" binding:"required"`
		Amount  float64 `json:"amount" binding:"required,min=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user is member of the group
	var isMember bool
	err := h.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)
	`, req.GroupID, userID.(uuid.UUID)).Scan(&isMember)

	if err != nil || !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "User is not a member of this group"})
		return
	}

	// Get group and user details
	var groupName, userName, userEmail string
	err = h.db.QueryRow(`
		SELECT g.name, u.full_name, u.email
		FROM groups g
		JOIN users u ON u.id = $2
		WHERE g.id = $1
	`, req.GroupID, userID.(uuid.UUID)).Scan(&groupName, &userName, &userEmail)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch group or user details"})
		return
	}

	// Load Midtrans configuration
	cfg := config.GetConfig()
	midtransConf := cfg.Midtrans

	// Generate order ID (max 36 characters for Midtrans)
	// Use first 8 chars of group ID + timestamp in seconds
	groupIDShort := req.GroupID
	if len(groupIDShort) > 8 {
		groupIDShort = groupIDShort[:8]
	}
	timestamp := time.Now().Unix()
	orderID := fmt.Sprintf("GRP-%s-%d", groupIDShort, timestamp)
	startTime := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	// Prepare payment request body
	body := map[string]interface{}{
		"transaction_details": map[string]interface{}{
			"order_id":     orderID,
			"gross_amount": int64(req.Amount),
		},
		"item_details": []map[string]interface{}{
			{
				"id":       fmt.Sprintf("group-%s", req.GroupID),
				"name":     fmt.Sprintf("Pembayaran Grup: %s", groupName),
				"price":    int64(req.Amount),
				"quantity": 1,
			},
		},
		"customer_details": map[string]interface{}{
			"first_name": userName,
			"last_name":  "User",
			"email":      userEmail,
			"phone":      "081234567890",
		},
		"expiry": map[string]interface{}{
			"start_time": startTime,
			"duration":   24, // 24 hours
			"unit":       "hours",
		},
	}

	// Make request to Midtrans
	jsonBody, _ := json.Marshal(body)
	url := midtransConf.BaseURL + "/v1/payment-links"
	reqHttp, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	reqHttp.Header.Set("Content-Type", "application/json")
	auth := base64.StdEncoding.EncodeToString([]byte(midtransConf.ServerKey + ":"))
	reqHttp.Header.Set("Authorization", "Basic "+auth)

	client := &http.Client{}
	resp, err := client.Do(reqHttp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment link: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	var respData map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respData)

	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Midtrans error",
			"detail": respData,
		})
		return
	}

	// Extract payment link ID from payment URL
	paymentURL, _ := respData["payment_url"].(string)
	paymentLinkID := ""
	if paymentURL != "" {
		parts := strings.Split(paymentURL, "/")
		if len(parts) > 0 {
			paymentLinkID = parts[len(parts)-1]
		}
	}

	// Create transaction record
	transactionID := uuid.New()
	description := fmt.Sprintf("Pembayaran grup %s, order_id: %s", groupName, orderID)
	_, err = h.db.Exec(`
		INSERT INTO transactions (id, user_id, group_id, type, amount, balance_before, balance_after, description, payment_reference, payment_link_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
	`, transactionID, userID.(uuid.UUID), req.GroupID, "group_payment", req.Amount, 0, 0, description, orderID, paymentLinkID, "pending")

	if err != nil {
		fmt.Printf("Warning: Failed to create transaction record: %v\n", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"payment_url":    respData["payment_url"],
		"order_id":       orderID,
		"transaction_id": transactionID.String(),
	})
}

func (h *PaymentHandler) HandlePaymentNotification(c *gin.Context) {
	var notification map[string]interface{}
	if err := c.ShouldBindJSON(&notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification format"})
		return
	}

	// Verify signature (implement proper signature verification)
	orderID, ok := notification["order_id"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	transactionStatus, ok := notification["transaction_status"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction status"})
		return
	}

	// Update payment status
	var status string
	switch transactionStatus {
	case "capture", "settlement":
		status = "paid"
	case "cancel", "deny", "expire":
		status = "cancelled"
	case "pending":
		status = "pending"
	default:
		status = "failed"
	}

	// Update transaction status (for group payments)
	_, err := h.db.Exec(`
		UPDATE transactions 
		SET status = $1, updated_at = $2 
		WHERE payment_reference = $3
	`, status, time.Now(), orderID)

	if err != nil {
		fmt.Printf("Warning: Failed to update transaction status: %v\n", err)
	}

	// Update group_members user_status for group payments
	if status == "paid" {
		_, err = h.db.Exec(`
			UPDATE group_members 
			SET user_status = 'paid', paid_at = $1, updated_at = $2
			FROM transactions t
			WHERE group_members.group_id = t.group_id 
			AND group_members.user_id = t.user_id
			AND t.payment_reference = $3
			AND t.type = 'group_payment'
		`, time.Now(), time.Now(), orderID)

		if err != nil {
			fmt.Printf("Warning: Failed to update group member status: %v\n", err)
		}
	}

	// Also update payments table if exists
	_, err = h.db.Exec(`
		UPDATE payments 
		SET status = $1, updated_at = $2 
		WHERE id = $3
	`, status, time.Now(), orderID)

	if err != nil {
		fmt.Printf("Warning: Failed to update payment status: %v\n", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Payment status updated"})
}

func (h *PaymentHandler) GetUserPayments(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	rows, err := h.db.Query(`
		SELECT p.id, p.subscription_id, p.user_id, p.amount, p.currency, p.status, 
		       p.midtrans_transaction_id, p.payment_method, p.created_at, p.updated_at,
		       s.service_name, s.plan_name
		FROM payments p
		JOIN subscriptions s ON p.subscription_id = s.id
		WHERE p.user_id = $1
		ORDER BY p.created_at DESC
	`, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payments"})
		return
	}
	defer rows.Close()

	var payments []models.Payment
	for rows.Next() {
		var payment models.Payment
		var serviceName, planName string
		err := rows.Scan(&payment.ID, &payment.SubscriptionID, &payment.UserID, &payment.Amount, &payment.Currency, &payment.Status, &payment.MidtransTransactionID, &payment.PaymentMethod, &payment.CreatedAt, &payment.UpdatedAt, &serviceName, &planName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan payment"})
			return
		}
		payments = append(payments, payment)
	}

	c.JSON(http.StatusOK, gin.H{"payments": payments})
}

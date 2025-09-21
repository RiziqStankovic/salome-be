package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	// "salome-be/internal/config"
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

	_, err := h.db.Exec(`
		UPDATE payments 
		SET status = $1, updated_at = $2 
		WHERE id = $3
	`, status, time.Now(), orderID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment status"})
		return
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

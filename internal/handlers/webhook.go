package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type WebhookHandler struct {
	db *sql.DB
}

func NewWebhookHandler(db *sql.DB) *WebhookHandler {
	return &WebhookHandler{db: db}
}

// WebhookRequest represents the webhook payload from Cloudfren Core
type WebhookRequest struct {
	OrderID           string `json:"order_id" binding:"required"`
	TransactionStatus string `json:"transaction_status" binding:"required"`
	PaymentType       string `json:"payment_type"`
	GrossAmount       string `json:"gross_amount"`
	FraudStatus       string `json:"fraud_status"`
	SignatureKey      string `json:"signature_key"`
	StatusMessage     string `json:"status_message"`
	MerchantID        string `json:"merchant_id"`
	TransactionTime   string `json:"transaction_time"`
	TransactionID     string `json:"transaction_id"`
	Bank              string `json:"bank"`
	Channel           string `json:"channel"`
	ApprovalCode      string `json:"approval_code"`
	Currency          string `json:"currency"`
}

// HandleWebhookFromCloudfren handles webhook from Cloudfren Core
func (h *WebhookHandler) HandleWebhookFromCloudfren(c *gin.Context) {
	var webhookReq WebhookRequest
	if err := c.ShouldBindJSON(&webhookReq); err != nil {
		fmt.Printf("[SALOME BE] ERROR: Failed to parse webhook payload: %v\n", err)
		errorResponse := gin.H{
			"error":  "Invalid webhook payload",
			"detail": err.Error(),
		}
		fmt.Printf("[SALOME BE] ERROR RESPONSE: %+v\n", errorResponse)
		c.JSON(http.StatusBadRequest, errorResponse)
		return
	}

	// Log webhook received dengan detail lengkap
	fmt.Printf("[SALOME BE] ===== WEBHOOK RECEIVED FROM CLOUDFREN CORE =====\n")
	fmt.Printf("[SALOME BE] Order ID: %s\n", webhookReq.OrderID)
	fmt.Printf("[SALOME BE] Transaction Status: %s\n", webhookReq.TransactionStatus)
	fmt.Printf("[SALOME BE] Payment Type: %s\n", webhookReq.PaymentType)
	fmt.Printf("[SALOME BE] Gross Amount: %s\n", webhookReq.GrossAmount)
	fmt.Printf("[SALOME BE] Fraud Status: %s\n", webhookReq.FraudStatus)
	fmt.Printf("[SALOME BE] Signature Key: %s\n", webhookReq.SignatureKey)
	fmt.Printf("[SALOME BE] Status Message: %s\n", webhookReq.StatusMessage)
	fmt.Printf("[SALOME BE] Merchant ID: %s\n", webhookReq.MerchantID)
	fmt.Printf("[SALOME BE] Transaction Time: %s\n", webhookReq.TransactionTime)
	fmt.Printf("[SALOME BE] Transaction ID: %s\n", webhookReq.TransactionID)
	fmt.Printf("[SALOME BE] Bank: %s\n", webhookReq.Bank)
	fmt.Printf("[SALOME BE] Channel: %s\n", webhookReq.Channel)
	fmt.Printf("[SALOME BE] Approval Code: %s\n", webhookReq.ApprovalCode)
	fmt.Printf("[SALOME BE] Currency: %s\n", webhookReq.Currency)
	fmt.Printf("[SALOME BE] ================================================\n")

	// Validate required fields
	if webhookReq.OrderID == "" {
		fmt.Printf("[SALOME BE] ERROR: order_id is required\n")
		errorResponse := gin.H{"error": "order_id is required"}
		fmt.Printf("[SALOME BE] ERROR RESPONSE: %+v\n", errorResponse)
		c.JSON(http.StatusBadRequest, errorResponse)
		return
	}

	if webhookReq.TransactionStatus == "" {
		fmt.Printf("[SALOME BE] ERROR: transaction_status is required\n")
		errorResponse := gin.H{"error": "transaction_status is required"}
		fmt.Printf("[SALOME BE] ERROR RESPONSE: %+v\n", errorResponse)
		c.JSON(http.StatusBadRequest, errorResponse)
		return
	}

	// Determine transaction status
	var status string
	switch webhookReq.TransactionStatus {
	case "capture", "settlement":
		status = "success"
	case "cancel", "deny", "expire":
		status = "failed"
	case "pending":
		status = "pending"
	default:
		status = "failed"
	}

	fmt.Printf("Mapped transaction status: %s -> %s\n", webhookReq.TransactionStatus, status)

	// Extract the base order ID (first 6 digits after prefix)
	var baseOrderID string
	if strings.HasPrefix(webhookReq.OrderID, "SALO-TOPUP-") {
		// Extract: SALO-TOPUP-892929-1759555433162 -> SALO-TOPUP-892929
		parts := strings.Split(webhookReq.OrderID, "-")
		if len(parts) >= 3 {
			baseOrderID = fmt.Sprintf("SALO-TOPUP-%s", parts[2])
		} else {
			baseOrderID = webhookReq.OrderID
		}
	} else if strings.HasPrefix(webhookReq.OrderID, "SALO-GRP-") {
		// Extract: SALO-GRP-123456-1759555433162 -> SALO-GRP-123456
		parts := strings.Split(webhookReq.OrderID, "-")
		if len(parts) >= 3 {
			baseOrderID = fmt.Sprintf("SALO-GRP-%s", parts[2])
		} else {
			baseOrderID = webhookReq.OrderID
		}
	} else {
		baseOrderID = webhookReq.OrderID
	}

	fmt.Printf("[SALOME BE] Original Order ID: %s\n", webhookReq.OrderID)
	fmt.Printf("[SALOME BE] Base Order ID: %s\n", baseOrderID)

	// Check if transaction exists using base order ID
	var exists bool
	err := h.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM transactions WHERE payment_reference = $1)
	`, baseOrderID).Scan(&exists)

	if err != nil {
		fmt.Printf("[SALOME BE] ERROR: Failed to check transaction existence: %v\n", err)
		errorResponse := gin.H{"error": "Failed to check transaction"}
		fmt.Printf("[SALOME BE] ERROR RESPONSE: %+v\n", errorResponse)
		c.JSON(http.StatusInternalServerError, errorResponse)
		return
	}

	if !exists {
		fmt.Printf("[SALOME BE] ERROR: Transaction not found: %s (base: %s)\n", webhookReq.OrderID, baseOrderID)
		errorResponse := gin.H{"error": "Transaction not found"}
		fmt.Printf("[SALOME BE] ERROR RESPONSE: %+v\n", errorResponse)
		c.JSON(http.StatusNotFound, errorResponse)
		return
	}

	// Update transaction status and payment method
	_, err = h.db.Exec(`
		UPDATE transactions 
		SET status = $1, payment_method = $2, updated_at = $3 
		WHERE payment_reference = $4
	`, status, webhookReq.PaymentType, time.Now(), baseOrderID)

	if err != nil {
		fmt.Printf("[SALOME BE] ERROR: Failed to update transaction status: %v\n", err)
		errorResponse := gin.H{"error": "Failed to update transaction status"}
		fmt.Printf("[SALOME BE] ERROR RESPONSE: %+v\n", errorResponse)
		c.JSON(http.StatusInternalServerError, errorResponse)
		return
	}

	fmt.Printf("Updated transaction status: %s -> %s, payment_method: %s (base: %s)\n", webhookReq.OrderID, status, webhookReq.PaymentType, baseOrderID)

	// If transaction is success, update user balance for top-up transactions
	if status == "success" {
		err = h.updateUserBalanceForTopUp(baseOrderID)
		if err != nil {
			fmt.Printf("Error updating user balance: %v\n", err)
			// Don't return error here, just log it
		}

		// Update group member status for group payments
		err = h.updateGroupMemberStatus(baseOrderID)
		if err != nil {
			fmt.Printf("Error updating group member status: %v\n", err)
			// Don't return error here, just log it
		}
	}

	// Prepare response
	response := gin.H{
		"success":        true,
		"message":        "Webhook processed successfully",
		"order_id":       webhookReq.OrderID,
		"base_order_id":  baseOrderID,
		"status":         status,
		"payment_method": webhookReq.PaymentType,
	}

	// Log response yang akan dikirim ke Cloudfren Core
	fmt.Printf("[SALOME BE] ===== RESPONSE TO CLOUDFREN CORE =====\n")
	fmt.Printf("[SALOME BE] Status Code: 200 OK\n")
	fmt.Printf("[SALOME BE] Response: %+v\n", response)
	fmt.Printf("[SALOME BE] ======================================\n")

	c.JSON(http.StatusOK, response)
}

// updateUserBalanceForTopUp updates user balance for top-up transactions
func (h *WebhookHandler) updateUserBalanceForTopUp(orderID string) error {
	// Check if this is a top-up transaction based on order_id prefix
	if !strings.HasPrefix(orderID, "SALO-TOPUP-") {
		// Not a top-up transaction, skip
		fmt.Printf("[SALOME BE] Skipping balance update - not a top-up transaction: %s\n", orderID)
		return nil
	}

	// Get transaction details
	var userID string
	var amount int

	err := h.db.QueryRow(`
		SELECT user_id, amount
		FROM transactions
		WHERE payment_reference = $1
	`, orderID).Scan(&userID, &amount)

	if err != nil {
		if err == sql.ErrNoRows {
			// Transaction not found
			fmt.Printf("[SALOME BE] Transaction not found for order_id: %s\n", orderID)
			return nil
		}
		return err
	}

	// Get current balance before update
	var currentBalance int
	err = h.db.QueryRow(`
		SELECT balance FROM users WHERE id = $1
	`, userID).Scan(&currentBalance)

	if err != nil {
		fmt.Printf("[SALOME BE] ERROR: Failed to get current balance for user %s: %v\n", userID, err)
		return err
	}

	fmt.Printf("[SALOME BE] Current balance before update: UserID=%s, CurrentBalance=%d, AmountToAdd=%d\n", userID, currentBalance, amount)

	// Update user balance by adding transaction amount
	_, err = h.db.Exec(`
		UPDATE users
		SET balance = balance + $1, updated_at = $2
		WHERE id = $3
	`, amount, time.Now(), userID)

	if err != nil {
		fmt.Printf("[SALOME BE] ERROR: Failed to update user balance: UserID=%s, Amount=%d, Error=%v\n", userID, amount, err)
		return err
	}

	// Get new balance after update
	var newBalance int
	err = h.db.QueryRow(`
		SELECT balance FROM users WHERE id = $1
	`, userID).Scan(&newBalance)

	if err != nil {
		fmt.Printf("[SALOME BE] WARNING: Failed to get new balance for user %s: %v\n", userID, err)
	} else {
		fmt.Printf("[SALOME BE] Updated user balance: UserID=%s, OldBalance=%d, AmountAdded=%d, NewBalance=%d\n", userID, currentBalance, amount, newBalance)
	}
	return nil
}

// updateGroupMemberStatus updates group member status for group payments
func (h *WebhookHandler) updateGroupMemberStatus(orderID string) error {
	// Check if this is a group payment transaction based on order_id prefix
	if !strings.HasPrefix(orderID, "SALO-GRP-") {
		// Not a group payment transaction, skip
		fmt.Printf("[SALOME BE] Skipping group member update - not a group payment transaction: %s\n", orderID)
		return nil
	}

	fmt.Printf("[SALOME BE] Updating group member status for order_id: %s\n", orderID)

	// Get transaction details
	var userID string
	var groupID *string

	err := h.db.QueryRow(`
		SELECT user_id, group_id
		FROM transactions
		WHERE payment_reference = $1
	`, orderID).Scan(&userID, &groupID)

	if err != nil {
		if err == sql.ErrNoRows {
			// Transaction not found
			fmt.Printf("[SALOME BE] Transaction not found for order_id: %s\n", orderID)
			return nil
		}
		return err
	}

	if groupID == nil {
		return nil
	}

	// Update group member status
	_, err = h.db.Exec(`
		UPDATE group_members
		SET user_status = 'paid', paid_at = $1
		WHERE group_id = $2 AND user_id = $3
	`, time.Now(), *groupID, userID)

	if err != nil {
		fmt.Printf("[SALOME BE] ERROR: Failed to update group member status: GroupID=%s, UserID=%s, Error=%v\n", *groupID, userID, err)
		return err
	}

	fmt.Printf("[SALOME BE] Updated group member status: GroupID=%s, UserID=%s\n", *groupID, userID)
	return nil
}

// HealthCheck endpoint for webhook
func (h *WebhookHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "salome-webhook",
		"timestamp": time.Now().Unix(),
	})
}

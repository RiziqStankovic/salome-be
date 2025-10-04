package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"salome-be/internal/service"

	"github.com/gin-gonic/gin"
)

type PaymentLinkHandler struct {
	db              *sql.DB
	midtransService *service.MidtransService
}

func NewPaymentLinkHandler(db *sql.DB) *PaymentLinkHandler {
	return &PaymentLinkHandler{
		db:              db,
		midtransService: service.NewMidtransService(),
	}
}

type CheckPaymentLinkRequest struct {
	PaymentLinkID string `json:"payment_link_id" binding:"required"`
}

type CheckPaymentLinkResponse struct {
	PaymentLinkID    string `json:"payment_link_id"`
	OrderID          string `json:"order_id"`
	Status           string `json:"status"`
	Amount           int    `json:"amount"`
	Currency         string `json:"currency"`
	ExpiryTime       string `json:"expiry_time"`
	PaymentURL       string `json:"payment_url"`
	IsExpired        bool   `json:"is_expired"`
	IsPaid           bool   `json:"is_paid"`
	IsPending        bool   `json:"is_pending"`
	TransactionCount int    `json:"transaction_count"`
	LastTransaction  *struct {
		TransactionID string `json:"transaction_id"`
		Status        string `json:"status"`
		Method        string `json:"method"`
		Amount        int    `json:"amount"`
		CreatedAt     string `json:"created_at"`
	} `json:"last_transaction,omitempty"`
}

// CheckPaymentLinkStatus mengecek status payment link di Midtrans
func (h *PaymentLinkHandler) CheckPaymentLinkStatus(c *gin.Context) {
	var req CheckPaymentLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	fmt.Printf("🔍 [PAYMENT LINK DEBUG] Checking payment link status for ID: %s\n", req.PaymentLinkID)

	// Cek status payment link di Midtrans
	midtransStatus, err := h.midtransService.GetPaymentLinkStatus(req.PaymentLinkID)
	if err != nil {
		fmt.Printf("❌ [PAYMENT LINK DEBUG] Midtrans API failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to check payment link status",
			"details": err.Error(),
		})
		return
	}

	fmt.Printf("✅ [PAYMENT LINK DEBUG] Payment link status retrieved successfully\n")

	// Cari transaksi terkait di database
	var transactionID string
	var dbStatus string
	var userID string
	var groupID string
	var amount int

	err = h.db.QueryRow(`
		SELECT id, status, user_id, group_id, amount 
		FROM transactions 
		WHERE payment_reference = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, req.PaymentLinkID).Scan(&transactionID, &dbStatus, &userID, &groupID, &amount)

	if err != nil {
		fmt.Printf("⚠️ [PAYMENT LINK DEBUG] No transaction found in database for payment link: %s\n", req.PaymentLinkID)
	}

	// Parse expiry time
	expiryTime, err := time.Parse(time.RFC3339, midtransStatus.ExpiryTime)
	isExpired := false
	if err == nil {
		isExpired = time.Now().After(expiryTime)
	}

	// Tentukan status berdasarkan data Midtrans
	var status string
	var isPaid bool
	var isPending bool

	switch midtransStatus.Status {
	case "ACTIVE":
		if len(midtransStatus.Purchases) > 0 {
			// Ada transaksi, cek status terakhir
			lastPurchase := midtransStatus.Purchases[len(midtransStatus.Purchases)-1]
			switch lastPurchase.PaymentStatus {
			case "SETTLEMENT":
				status = "paid"
				isPaid = true
			case "PENDING":
				status = "pending"
				isPending = true
			case "EXPIRE":
				status = "expired"
			default:
				status = "failed"
			}
		} else {
			status = "active"
			isPending = true
		}
	case "EXPIRED":
		status = "expired"
		isExpired = true
	default:
		status = "unknown"
	}

	// Siapkan response
	response := CheckPaymentLinkResponse{
		PaymentLinkID:    req.PaymentLinkID,
		OrderID:          midtransStatus.OrderID,
		Status:           status,
		Amount:           midtransStatus.GrossAmount,
		Currency:         midtransStatus.Currency,
		ExpiryTime:       midtransStatus.ExpiryTime,
		PaymentURL:       fmt.Sprintf("https://app.sandbox.midtrans.com/payment-links/%s", req.PaymentLinkID),
		IsExpired:        isExpired,
		IsPaid:           isPaid,
		IsPending:        isPending,
		TransactionCount: len(midtransStatus.Purchases),
	}

	// Tambahkan informasi transaksi terakhir jika ada
	if len(midtransStatus.Purchases) > 0 {
		lastPurchase := midtransStatus.Purchases[len(midtransStatus.Purchases)-1]
		response.LastTransaction = &struct {
			TransactionID string `json:"transaction_id"`
			Status        string `json:"status"`
			Method        string `json:"method"`
			Amount        int    `json:"amount"`
			CreatedAt     string `json:"created_at"`
		}{
			TransactionID: lastPurchase.TransactionID,
			Status:        lastPurchase.PaymentStatus,
			Method:        lastPurchase.PaymentMethod,
			Amount:        lastPurchase.AmountValue,
			CreatedAt:     lastPurchase.CreatedAt,
		}
	}

	fmt.Printf("✅ [PAYMENT LINK DEBUG] Response prepared:\n")
	fmt.Printf("   - Status: %s\n", response.Status)
	fmt.Printf("   - IsPaid: %t\n", response.IsPaid)
	fmt.Printf("   - IsPending: %t\n", response.IsPending)
	fmt.Printf("   - IsExpired: %t\n", response.IsExpired)
	fmt.Printf("   - Transaction Count: %d\n", response.TransactionCount)

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment link status retrieved successfully",
		"data":    response,
	})
}

// GetPaymentLinkInfo mengambil informasi payment link tanpa cek status
func (h *PaymentLinkHandler) GetPaymentLinkInfo(c *gin.Context) {
	paymentLinkID := c.Param("payment_link_id")
	if paymentLinkID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment link ID is required"})
		return
	}

	fmt.Printf("🔍 [PAYMENT LINK DEBUG] Getting payment link info for ID: %s\n", paymentLinkID)

	// Cari transaksi terkait di database
	var transactionID string
	var dbStatus string
	var userID string
	var groupID string
	var amount int
	var createdAt time.Time

	err := h.db.QueryRow(`
		SELECT id, status, user_id, group_id, amount, created_at
		FROM transactions 
		WHERE payment_reference = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, paymentLinkID).Scan(&transactionID, &dbStatus, &userID, &groupID, &amount, &createdAt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Payment link not found in database",
			"details": err.Error(),
		})
		return
	}

	// Siapkan response dengan data dari database
	response := gin.H{
		"payment_link_id": paymentLinkID,
		"transaction_id":  transactionID,
		"status":          dbStatus,
		"amount":          amount,
		"user_id":         userID,
		"group_id":        groupID,
		"created_at":      createdAt.Format(time.RFC3339),
		"payment_url":     fmt.Sprintf("https://app.sandbox.midtrans.com/payment-links/%s", paymentLinkID),
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment link info retrieved successfully",
		"data":    response,
	})
}

// ListUserPaymentLinks mengambil daftar payment links milik user
func (h *PaymentLinkHandler) ListUserPaymentLinks(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	fmt.Printf("🔍 [PAYMENT LINK DEBUG] Getting payment links for user: %s\n", userID)

	// Ambil semua transaksi user yang memiliki payment_reference
	rows, err := h.db.Query(`
		SELECT id, payment_reference, status, amount, group_id, created_at, updated_at
		FROM transactions 
		WHERE user_id = $1 AND payment_reference IS NOT NULL
		ORDER BY created_at DESC
	`, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch payment links",
			"details": err.Error(),
		})
		return
	}
	defer rows.Close()

	var paymentLinks []gin.H
	for rows.Next() {
		var transactionID, paymentReference, status, groupID string
		var amount int
		var createdAt, updatedAt time.Time

		err := rows.Scan(&transactionID, &paymentReference, &status, &amount, &groupID, &createdAt, &updatedAt)
		if err != nil {
			fmt.Printf("❌ [PAYMENT LINK DEBUG] Error scanning row: %v\n", err)
			continue
		}

		paymentLinks = append(paymentLinks, gin.H{
			"transaction_id":  transactionID,
			"payment_link_id": paymentReference,
			"status":          status,
			"amount":          amount,
			"group_id":        groupID,
			"created_at":      createdAt.Format(time.RFC3339),
			"updated_at":      updatedAt.Format(time.RFC3339),
			"payment_url":     fmt.Sprintf("https://app.sandbox.midtrans.com/payment-links/%s", paymentReference),
		})
	}

	fmt.Printf("✅ [PAYMENT LINK DEBUG] Found %d payment links for user %s\n", len(paymentLinks), userID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment links retrieved successfully",
		"data":    paymentLinks,
		"count":   len(paymentLinks),
	})
}

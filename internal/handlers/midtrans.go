package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"salome-be/internal/service"

	"github.com/gin-gonic/gin"
)

type MidtransHandler struct {
	db              *sql.DB
	midtransService *service.MidtransService
}

func NewMidtransHandler(db *sql.DB) *MidtransHandler {
	return &MidtransHandler{
		db:              db,
		midtransService: service.NewMidtransService(),
	}
}

type CheckTransactionStatusRequest struct {
	OrderID string `json:"order_id" binding:"required"`
}

type CheckTransactionStatusResponse struct {
	OrderID           string `json:"order_id"`
	TransactionStatus string `json:"transaction_status"`
	PaymentType       string `json:"payment_type"`
	GrossAmount       string `json:"gross_amount"`
	TransactionTime   string `json:"transaction_time"`
	PaymentLinkURL    string `json:"payment_link_url,omitempty"`
	IsSettled         bool   `json:"is_settled"`
	IsPending         bool   `json:"is_pending"`
	IsFailed          bool   `json:"is_failed"`
}

// CheckTransactionStatus mengecek status transaksi di Midtrans dan update database
func (h *MidtransHandler) CheckTransactionStatus(c *gin.Context) {
	var req CheckTransactionStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("🚀 [HANDLER DEBUG] CheckTransactionStatus called with order ID: %s\n", req.OrderID)

	// Cari transaksi di database dulu untuk mendapatkan payment_reference
	var transactionID string
	var currentStatus string
	var paymentReference string

	// Cari transaksi di database dengan order ID tanpa suffix
	err := h.db.QueryRow(`
		SELECT id, status, payment_reference
		FROM transactions 
		WHERE payment_reference = $1
	`, req.OrderID).Scan(&transactionID, &currentStatus, &paymentReference)

	// Jika tidak ditemukan dengan order ID asli, coba cari dengan pattern yang dimulai dengan order ID asli
	if err != nil {
		fmt.Printf("❌ [HANDLER DEBUG] Transaction not found with exact order ID, trying with pattern: %v\n", err)

		// Cari transaksi yang payment_reference dimulai dengan order ID asli
		err = h.db.QueryRow(`
			SELECT id, status, payment_reference
			FROM transactions 
			WHERE payment_reference LIKE $1
			ORDER BY created_at DESC
			LIMIT 1
		`, req.OrderID+"%").Scan(&transactionID, &currentStatus, &paymentReference)

		if err != nil {
			fmt.Printf("❌ [HANDLER DEBUG] Transaction not found with pattern either: %v\n", err)
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Transaction not found in database",
				"details": err.Error(),
			})
			return
		}
		fmt.Printf("✅ [HANDLER DEBUG] Transaction found with pattern matching\n")
	}

	fmt.Printf("✅ [HANDLER DEBUG] Transaction found in database:\n")
	fmt.Printf("   - Transaction ID: %s\n", transactionID)
	fmt.Printf("   - Current Status: %s\n", currentStatus)
	fmt.Printf("   - Payment Reference: %s\n", paymentReference)

	// Jika sudah completed, return status dari database
	if currentStatus == "completed" {
		response := CheckTransactionStatusResponse{
			OrderID:           req.OrderID,
			TransactionStatus: "settlement",
			PaymentType:       "unknown",
			GrossAmount:       "0",
			TransactionTime:   "",
			PaymentLinkURL:    "",
			IsSettled:         true,
			IsPending:         false,
			IsFailed:          false,
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Transaction already completed",
			"data":    response,
		})
		return
	}

	// Cek status di Midtrans menggunakan payment_reference sebagai payment_link_id
	var midtransStatus *service.MidtransTransactionStatus
	var midtransErr error

	fmt.Printf("🔍 [HANDLER DEBUG] Checking Midtrans status...\n")
	fmt.Printf("🔍 [HANDLER DEBUG] Using payment_reference as payment_link_id: %s\n", paymentReference)

	// Gunakan payment_reference sebagai payment_link_id untuk cek ke Midtrans Payment Link API
	midtransStatus, midtransErr = h.midtransService.GetTransactionStatus(paymentReference)

	if midtransErr != nil {
		fmt.Printf("❌ [HANDLER DEBUG] Midtrans API failed: %v\n", midtransErr)
	} else {
		fmt.Printf("✅ [HANDLER DEBUG] Midtrans API success\n")
	}

	// Jika tidak ditemukan di Midtrans, return status dari database
	if midtransErr != nil {
		response := CheckTransactionStatusResponse{
			OrderID:           req.OrderID,
			TransactionStatus: currentStatus,
			PaymentType:       "unknown",
			GrossAmount:       "0",
			TransactionTime:   "",
			PaymentLinkURL:    h.getPaymentLinkURL(paymentReference),
			IsSettled:         currentStatus == "completed",
			IsPending:         currentStatus == "pending",
			IsFailed:          currentStatus == "failed",
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Transaction status from database (Midtrans not found)",
			"data":    response,
		})
		return
	}

	// Cek apakah transaksi sudah settlement
	isSettled := h.midtransService.IsTransactionSettled(midtransStatus.TransactionStatus)
	isPending := h.midtransService.IsTransactionPending(midtransStatus.TransactionStatus)
	isFailed := h.midtransService.IsTransactionFailed(midtransStatus.TransactionStatus)

	fmt.Printf("🔍 [HANDLER DEBUG] Status check results:\n")
	fmt.Printf("   - TransactionStatus: %s\n", midtransStatus.TransactionStatus)
	fmt.Printf("   - IsSettled: %t\n", isSettled)
	fmt.Printf("   - IsPending: %t\n", isPending)
	fmt.Printf("   - IsFailed: %t\n", isFailed)

	// Tentukan status database berdasarkan status Midtrans
	var dbStatus string
	if isSettled {
		dbStatus = "success" // SETTLEMENT -> success di database
	} else if isFailed {
		// Jika expired, set status expired
		if midtransStatus.TransactionStatus == "expire" || midtransStatus.TransactionStatus == "EXPIRE" {
			dbStatus = "expired"
		} else {
			dbStatus = "failed"
		}
	} else {
		dbStatus = "pending"
	}

	fmt.Printf("🔍 [HANDLER DEBUG] Database status: %s\n", dbStatus)

	// Update database dengan status yang sesuai
	if isSettled || isFailed {
		err = h.updateTransactionStatus(req.OrderID, dbStatus, midtransStatus)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to update transaction status",
				"details": err.Error(),
			})
			return
		}
	}

	// Tentukan status response
	var responseStatus string
	if isSettled {
		responseStatus = "success" // SETTLEMENT -> success di response
	} else if midtransStatus.TransactionStatus == "expire" || midtransStatus.TransactionStatus == "EXPIRE" {
		responseStatus = "expired"
	} else if isFailed {
		responseStatus = "failed"
	} else {
		responseStatus = "pending"
	}

	response := CheckTransactionStatusResponse{
		OrderID:           req.OrderID, // Gunakan order ID asli yang diminta user
		TransactionStatus: responseStatus,
		PaymentType:       midtransStatus.PaymentType,
		GrossAmount:       midtransStatus.GrossAmount,
		TransactionTime:   midtransStatus.TransactionTime,
		PaymentLinkURL:    h.getPaymentLinkURL(paymentReference),
		IsSettled:         isSettled,
		IsPending:         isPending,
		IsFailed:          isFailed,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Transaction status retrieved successfully",
		"data":    response,
	})
}

// updateTransactionStatus mengupdate status transaksi di database
func (h *MidtransHandler) updateTransactionStatus(orderID, status string, midtransStatus *service.MidtransTransactionStatus) error {
	// Cari transaksi berdasarkan order_id
	var transactionID string
	var currentStatus string
	var userID string
	var groupID string
	var amount int

	err := h.db.QueryRow(`
		SELECT id, status, user_id, group_id, amount 
		FROM transactions 
		WHERE payment_reference = $1
	`, orderID).Scan(&transactionID, &currentStatus, &userID, &groupID, &amount)

	if err != nil {
		return fmt.Errorf("transaction not found: %v", err)
	}

	// Jangan update jika sudah completed
	if currentStatus == "completed" {
		return nil
	}

	// Update status transaksi dan payment method
	fmt.Printf("🔄 [HANDLER DEBUG] Updating transaction status: %s, payment_method: %s\n", status, midtransStatus.PaymentType)
	_, err = h.db.Exec(`
		UPDATE transactions 
		SET status = $1, payment_method = $2, updated_at = $3
		WHERE id = $4
	`, status, midtransStatus.PaymentType, time.Now(), transactionID)

	if err != nil {
		return fmt.Errorf("failed to update transaction: %v", err)
	}
	fmt.Printf("✅ [HANDLER DEBUG] Transaction updated successfully\n")

	// Jika status success (SETTLEMENT), update saldo user dan group member status
	if status == "success" {
		fmt.Printf("💰 [HANDLER DEBUG] Processing SETTLEMENT - updating user balance and group member status\n")

		// Update saldo user dan total_spent
		fmt.Printf("🔄 [HANDLER DEBUG] Updating user balance: -%d for user_id: %s\n", amount, userID)
		_, err = h.db.Exec(`
			UPDATE users 
			SET balance = balance - $1, total_spent = total_spent + $1, updated_at = $2
			WHERE id = $3
		`, amount, time.Now(), userID)

		if err != nil {
			return fmt.Errorf("failed to update user balance and total_spent: %v", err)
		}
		fmt.Printf("✅ [HANDLER DEBUG] User balance and total_spent updated successfully\n")

		// Update group member status menjadi paid
		if groupID != "" {
			fmt.Printf("🔄 [HANDLER DEBUG] Updating group member status to 'paid' for user_id: %s, group_id: %s\n", userID, groupID)
			_, err = h.db.Exec(`
				UPDATE group_members 
				SET user_status = 'paid'
				WHERE user_id = $1 AND group_id = $2
			`, userID, groupID)

			if err != nil {
				return fmt.Errorf("failed to update group member status: %v", err)
			}
			fmt.Printf("✅ [HANDLER DEBUG] Group member status updated to 'paid' successfully\n")

			// Check apakah semua member dalam group sudah paid
			err = h.checkAndUpdateGroupStatus(groupID)
			if err != nil {
				fmt.Printf("⚠️ [HANDLER DEBUG] Failed to check group status: %v\n", err)
				// Tidak return error karena ini bukan critical
			}
		} else {
			fmt.Printf("⚠️ [HANDLER DEBUG] No group_id found, skipping group member update\n")
		}
	} else if status == "expired" {
		// Untuk expired, tidak perlu update balance atau group member
		// Hanya update status transaksi saja
		fmt.Printf("✅ [HANDLER DEBUG] Transaction expired, no balance or group member update needed\n")
	}

	return nil
}

// GetTransactionPaymentLink mengambil payment link untuk transaksi yang masih pending
func (h *MidtransHandler) GetTransactionPaymentLink(c *gin.Context) {
	orderID := c.Param("order_id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}

	// Cek apakah transaksi ada di database
	var transactionID string
	var status string
	var paymentReference string
	err := h.db.QueryRow(`
		SELECT id, status, payment_reference 
		FROM transactions 
		WHERE payment_reference = $1
	`, orderID).Scan(&transactionID, &status, &paymentReference)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	// Jika sudah completed, tidak perlu payment link
	if status == "completed" {
		c.JSON(http.StatusOK, gin.H{
			"message": "Transaction already completed",
			"data": gin.H{
				"order_id":         orderID,
				"status":           status,
				"payment_link_url": nil,
			},
		})
		return
	}

	// Construct payment link URL dari payment_reference
	var paymentLinkURL string
	if paymentReference != "" {
		paymentLinkURL = fmt.Sprintf("https://app.sandbox.midtrans.com/payment-links/%s", paymentReference)
		fmt.Printf("🔗 [HANDLER DEBUG] Payment link URL: %s\n", paymentLinkURL)
	} else {
		fmt.Printf("⚠️ [HANDLER DEBUG] No payment_reference found for order: %s\n", orderID)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment link retrieved successfully",
		"data": gin.H{
			"order_id":         orderID,
			"status":           status,
			"payment_link_url": paymentLinkURL,
		},
	})
}

// getPaymentLinkURL helper function untuk construct payment link URL
func (h *MidtransHandler) getPaymentLinkURL(paymentLinkID string) string {
	if paymentLinkID == "" {
		return ""
	}
	return fmt.Sprintf("https://app.sandbox.midtrans.com/payment-links/%s", paymentLinkID)
}

// checkAndUpdateGroupStatus mengecek apakah semua member dalam group sudah paid
// dan mengupdate group_status menjadi "paid" jika semua sudah bayar
func (h *MidtransHandler) checkAndUpdateGroupStatus(groupID string) error {
	fmt.Printf("🔍 [GROUP STATUS DEBUG] Checking group completion for group_id: %s\n", groupID)

	// Cek total member dalam group
	var totalMembers int
	err := h.db.QueryRow(`
		SELECT COUNT(*) 
		FROM group_members 
		WHERE group_id = $1
	`, groupID).Scan(&totalMembers)

	if err != nil {
		return fmt.Errorf("failed to count group members: %v", err)
	}

	// Cek berapa banyak member yang sudah paid
	var paidMembers int
	err = h.db.QueryRow(`
		SELECT COUNT(*) 
		FROM group_members 
		WHERE group_id = $1 AND user_status = 'paid'
	`, groupID).Scan(&paidMembers)

	if err != nil {
		return fmt.Errorf("failed to count paid members: %v", err)
	}

	fmt.Printf("📊 [GROUP STATUS DEBUG] Group %s: %d/%d members paid\n", groupID, paidMembers, totalMembers)

	// Jika semua member sudah paid, update group_status menjadi "paid"
	if paidMembers == totalMembers && totalMembers > 0 {
		fmt.Printf("🎉 [GROUP STATUS DEBUG] All members paid! Updating group_status to 'paid_group'\n")

		// Use Asia/Jakarta timezone
		loc, _ := time.LoadLocation("Asia/Jakarta")
		now := time.Now().In(loc)

		_, err = h.db.Exec(`
			UPDATE groups 
			SET group_status = 'paid_group', all_paid_at = $1, updated_at = $1
			WHERE id = $2
		`, now, groupID)

		if err != nil {
			return fmt.Errorf("failed to update group status to paid: %v", err)
		}

		fmt.Printf("✅ [GROUP STATUS DEBUG] Group %s status updated to 'paid_group' successfully\n", groupID)
		fmt.Printf("📅 [GROUP STATUS DEBUG] all_paid_at set to: %s\n", now.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("⏳ [GROUP STATUS DEBUG] Group %s still waiting for %d more members to pay\n", groupID, totalMembers-paidMembers)
	}

	return nil
}

// resetGroupStatusIfNeeded mereset group_status menjadi "pending" jika ada member yang belum paid
func (h *MidtransHandler) resetGroupStatusIfNeeded(groupID string) error {
	fmt.Printf("🔄 [GROUP STATUS DEBUG] Checking if group %s needs status reset\n", groupID)

	// Cek apakah group status sudah "paid"
	var currentStatus string
	err := h.db.QueryRow(`
		SELECT group_status 
		FROM groups 
		WHERE id = $1
	`, groupID).Scan(&currentStatus)

	if err != nil {
		return fmt.Errorf("failed to get current group status: %v", err)
	}

	// Jika group status sudah "paid_group", cek apakah masih ada member yang belum paid
	if currentStatus == "paid_group" {
		var unpaidMembers int
		err = h.db.QueryRow(`
			SELECT COUNT(*) 
			FROM group_members 
			WHERE group_id = $1 AND user_status != 'paid'
		`, groupID).Scan(&unpaidMembers)

		if err != nil {
			return fmt.Errorf("failed to count unpaid members: %v", err)
		}

		// Jika ada member yang belum paid, reset group status ke "pending"
		if unpaidMembers > 0 {
			fmt.Printf("⚠️ [GROUP STATUS DEBUG] Found %d unpaid members, resetting group status to 'pending'\n", unpaidMembers)

			// Use Asia/Jakarta timezone
			loc, _ := time.LoadLocation("Asia/Jakarta")
			now := time.Now().In(loc)

			_, err = h.db.Exec(`
				UPDATE groups 
				SET group_status = 'pending', all_paid_at = NULL, updated_at = $1
				WHERE id = $2
			`, now, groupID)

			if err != nil {
				return fmt.Errorf("failed to reset group status to pending: %v", err)
			}

			fmt.Printf("✅ [GROUP STATUS DEBUG] Group %s status reset to 'pending'\n", groupID)
			fmt.Printf("📅 [GROUP STATUS DEBUG] all_paid_at reset to NULL\n")
		}
	}

	return nil
}

// UpdateAllPendingTransactions mengecek dan update semua transaksi yang statusnya pending
func (h *MidtransHandler) UpdateAllPendingTransactions(c *gin.Context) {
	fmt.Printf("🚀 [HANDLER DEBUG] UpdateAllPendingTransactions called\n")

	// Ambil semua transaksi dengan status pending
	query := `
		SELECT id, payment_reference, payment_link_id, group_id, user_id, amount, status
		FROM transactions 
		WHERE status = 'pending' AND payment_reference IS NOT NULL
		ORDER BY created_at ASC
	`

	rows, err := h.db.Query(query)
	if err != nil {
		fmt.Printf("❌ [ERROR] Failed to fetch pending transactions: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch pending transactions",
			"details": err.Error(),
		})
		return
	}
	defer rows.Close()

	var pendingTransactions []struct {
		ID               string
		PaymentReference string
		PaymentLinkID    *string
		GroupID          string
		UserID           string
		Amount           float64
		Status           string
	}

	for rows.Next() {
		var tx struct {
			ID               string
			PaymentReference string
			PaymentLinkID    *string
			GroupID          string
			UserID           string
			Amount           float64
			Status           string
		}

		err := rows.Scan(
			&tx.ID,
			&tx.PaymentReference,
			&tx.PaymentLinkID,
			&tx.GroupID,
			&tx.UserID,
			&tx.Amount,
			&tx.Status,
		)
		if err != nil {
			fmt.Printf("❌ [ERROR] Failed to scan transaction: %v\n", err)
			continue
		}

		pendingTransactions = append(pendingTransactions, tx)
	}

	fmt.Printf("📊 [STATS] Found %d pending transactions to process\n", len(pendingTransactions))

	if len(pendingTransactions) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message":   "No pending transactions found",
			"processed": 0,
			"updated":   0,
			"failed":    0,
		})
		return
	}

	processed := 0
	updated := 0
	failed := 0

	// Proses setiap transaksi pending
	for _, tx := range pendingTransactions {
		processed++
		fmt.Printf("🔄 [PROCESSING] Transaction %s (Payment Ref: %s)\n", tx.ID, tx.PaymentReference)

		// Cek status di Midtrans menggunakan payment_reference
		fmt.Printf("🔍 [DEBUG] Checking Midtrans status for payment_reference: %s\n", tx.PaymentReference)

		midtransStatus, err := h.midtransService.GetTransactionStatus(tx.PaymentReference)
		if err != nil {
			fmt.Printf("❌ [ERROR] Failed to check Midtrans status for payment_reference %s: %v\n", tx.PaymentReference, err)
			failed++
			continue
		}

		// Map status Midtrans ke status internal
		var newStatus string
		switch midtransStatus.TransactionStatus {
		case "settlement":
			newStatus = "success"
		case "expire":
			newStatus = "expired"
		case "deny", "cancel", "failed":
			newStatus = "failed"
		default:
			newStatus = "pending"
		}

		fmt.Printf("📋 [STATUS] Transaction %s: %s -> %s\n", tx.ID, tx.Status, newStatus)

		// Update status transaksi
		_, err = h.db.Exec(`
			UPDATE transactions 
			SET status = $1, updated_at = $2
			WHERE id = $3
		`, newStatus, time.Now(), tx.ID)

		if err != nil {
			fmt.Printf("❌ [ERROR] Failed to update transaction %s: %v\n", tx.ID, err)
			failed++
			continue
		}

		// Jika status berubah menjadi success, update group members dan balance
		if newStatus == "success" {
			// Update user balance dan total_spent
			_, err = h.db.Exec(`
				UPDATE users 
				SET balance = balance - $1, total_spent = total_spent + $1, updated_at = $2
				WHERE id = $3
			`, tx.Amount, time.Now(), tx.UserID)

			if err != nil {
				fmt.Printf("❌ [ERROR] Failed to update user balance and total_spent for %s: %v\n", tx.UserID, err)
			}

			// Update group member status
			_, err = h.db.Exec(`
				UPDATE group_members 
				SET user_status = 'paid', updated_at = $1
				WHERE group_id = $2 AND user_id = $3
			`, time.Now(), tx.GroupID, tx.UserID)

			if err != nil {
				fmt.Printf("❌ [ERROR] Failed to update group member status: %v\n", err)
			}

			// Cek dan update group status
			err = h.checkAndUpdateGroupStatus(tx.GroupID)
			if err != nil {
				fmt.Printf("❌ [ERROR] Failed to update group status: %v\n", err)
			}
		}

		updated++
		fmt.Printf("✅ [SUCCESS] Transaction %s updated to %s\n", tx.ID, newStatus)
	}

	fmt.Printf("🎉 [COMPLETE] Processed %d transactions: %d updated, %d failed\n", processed, updated, failed)

	c.JSON(http.StatusOK, gin.H{
		"message":   "All pending transactions processed",
		"processed": processed,
		"updated":   updated,
		"failed":    failed,
	})
}

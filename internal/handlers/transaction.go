package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"salome-be/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TransactionHandler struct {
	db *sql.DB
}

func NewTransactionHandler(db *sql.DB) *TransactionHandler {
	return &TransactionHandler{db: db}
}

// GetUserTransactions retrieves transactions for a specific user
func (h *TransactionHandler) GetUserTransactions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Get pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// Query transactions
	query := `
		SELECT 
			id, user_id, group_id, type, amount, balance_before, balance_after,
			description, payment_method, payment_reference, status, created_at, updated_at
		FROM transactions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := h.db.Query(query, userID, pageSize, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
		return
	}
	defer rows.Close()

	var transactions []models.TransactionResponse
	for rows.Next() {
		var txn models.TransactionResponse
		err := rows.Scan(
			&txn.ID, &txn.UserID, &txn.GroupID, &txn.Type, &txn.Amount,
			&txn.BalanceBefore, &txn.BalanceAfter, &txn.Description,
			&txn.PaymentMethod, &txn.PaymentReference, &txn.Status,
			&txn.CreatedAt, &txn.UpdatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan transaction"})
			return
		}
		transactions = append(transactions, txn)
	}

	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM transactions WHERE user_id = $1`
	err = h.db.QueryRow(countQuery, userID).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count transactions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		"total":        total,
		"page":         page,
		"page_size":    pageSize,
		"total_pages":  (total + pageSize - 1) / pageSize,
	})
}

// CreateTransaction creates a new transaction
func (h *TransactionHandler) CreateTransaction(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req models.TransactionCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current user balance
	var currentBalance float64
	balanceQuery := `SELECT balance FROM users WHERE id = $1`
	err := h.db.QueryRow(balanceQuery, userID).Scan(&currentBalance)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user balance"})
		return
	}

	// Calculate new balance based on transaction type
	var newBalance float64
	switch req.Type {
	case "top-up":
		newBalance = currentBalance + req.Amount
	case "group_payment", "withdrawal":
		if currentBalance < req.Amount {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient balance"})
			return
		}
		newBalance = currentBalance - req.Amount
	case "refund":
		newBalance = currentBalance + req.Amount
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction type"})
		return
	}

	// Start transaction
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Insert transaction
	insertQuery := `
		INSERT INTO transactions (
			user_id, group_id, type, amount, balance_before, balance_after,
			description, payment_method, payment_reference, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`

	var transactionID uuid.UUID
	now := time.Now()
	err = tx.QueryRow(
		insertQuery, userID, req.GroupID, req.Type, req.Amount,
		currentBalance, newBalance, req.Description, req.PaymentMethod,
		req.PaymentReference, "completed", now, now,
	).Scan(&transactionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction"})
		return
	}

	// Update user balance
	updateBalanceQuery := `UPDATE users SET balance = $1, updated_at = $2 WHERE id = $3`
	_, err = tx.Exec(updateBalanceQuery, newBalance, now, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user balance"})
		return
	}

	// Update total_spent if it's a payment
	if req.Type == "group_payment" {
		updateSpentQuery := `UPDATE users SET total_spent = total_spent + $1, updated_at = $2 WHERE id = $3`
		_, err = tx.Exec(updateSpentQuery, req.Amount, now, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update total spent"})
			return
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	response := models.TransactionResponse{
		ID:               transactionID,
		UserID:           userID.(uuid.UUID),
		GroupID:          req.GroupID,
		Type:             req.Type,
		Amount:           req.Amount,
		BalanceBefore:    currentBalance,
		BalanceAfter:     newBalance,
		Description:      req.Description,
		PaymentMethod:    &req.PaymentMethod,
		PaymentReference: &req.PaymentReference,
		Status:           "completed",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	c.JSON(http.StatusCreated, response)
}

// TopUpBalance handles user balance top-up
func (h *TransactionHandler) TopUpBalance(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		Amount    float64 `json:"amount" binding:"required,min=1000"`
		Method    string  `json:"method" binding:"required"`
		Reference string  `json:"reference"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create top-up transaction
	transactionReq := models.TransactionCreateRequest{
		Type:             "top-up",
		Amount:           req.Amount,
		Description:      "Balance top-up via " + req.Method,
		PaymentMethod:    req.Method,
		PaymentReference: req.Reference,
	}

	// Get current user balance
	var currentBalance float64
	balanceQuery := `SELECT balance FROM users WHERE id = $1`
	err := h.db.QueryRow(balanceQuery, userID).Scan(&currentBalance)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user balance"})
		return
	}

	// Calculate new balance
	newBalance := currentBalance + req.Amount

	// Start transaction
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Insert transaction
	insertQuery := `
		INSERT INTO transactions (
			user_id, group_id, type, amount, balance_before, balance_after,
			description, payment_method, payment_reference, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`

	var transactionID uuid.UUID
	now := time.Now()
	err = tx.QueryRow(
		insertQuery, userID, nil, transactionReq.Type, transactionReq.Amount,
		currentBalance, newBalance, transactionReq.Description, transactionReq.PaymentMethod,
		transactionReq.PaymentReference, "completed", now, now,
	).Scan(&transactionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction"})
		return
	}

	// Update user balance
	updateBalanceQuery := `UPDATE users SET balance = $1, updated_at = $2 WHERE id = $3`
	_, err = tx.Exec(updateBalanceQuery, newBalance, now, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user balance"})
		return
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	response := models.TransactionResponse{
		ID:               transactionID,
		UserID:           userID.(uuid.UUID),
		GroupID:          nil,
		Type:             transactionReq.Type,
		Amount:           transactionReq.Amount,
		BalanceBefore:    currentBalance,
		BalanceAfter:     newBalance,
		Description:      transactionReq.Description,
		PaymentMethod:    &transactionReq.PaymentMethod,
		PaymentReference: &transactionReq.PaymentReference,
		Status:           "completed",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	c.JSON(http.StatusCreated, response)
}

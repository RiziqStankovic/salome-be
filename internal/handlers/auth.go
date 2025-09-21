package handlers

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"salome-be/internal/models"
	"salome-be/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	db *sql.DB
}

func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{db: db}
}

// generateOTP generates a random 6-digit OTP code
func (h *AuthHandler) generateOTP() (string, error) {
	// Generate random 6-digit number
	max := big.NewInt(999999)
	min := big.NewInt(100000)

	n, err := rand.Int(rand.Reader, new(big.Int).Sub(max, min))
	if err != nil {
		return "", err
	}

	otp := n.Add(n, min)
	return fmt.Sprintf("%06d", otp.Int64()), nil
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req models.UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user already exists
	var existingUser models.User
	err := h.db.QueryRow("SELECT id FROM users WHERE email = $1", req.Email).Scan(&existingUser.ID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Create user with pending verification status
	userID := uuid.New()
	_, err = h.db.Exec(`
		INSERT INTO users (id, email, password_hash, full_name, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, userID, req.Email, string(hashedPassword), req.FullName, "pending_verification", time.Now(), time.Now())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Generate OTP for email verification
	otpCode, err := h.generateOTP()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OTP"})
		return
	}

	// Save OTP to database
	otpID := uuid.New().String()
	expiresAt := time.Now().Add(5 * time.Minute)
	_, err = h.db.Exec(`
		INSERT INTO otps (id, user_id, email, otp_code, purpose, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, otpID, userID, req.Email, otpCode, "email_verification", expiresAt, time.Now())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save OTP"})
		return
	}

	// Generate JWT token
	token, err := utils.GenerateJWT(userID, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	userResponse := models.UserResponse{
		ID:        userID,
		Email:     req.Email,
		FullName:  req.FullName,
		Status:    "pending_verification",
		CreatedAt: time.Now(),
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "User created successfully. Please check your email for verification code.",
		"user":       userResponse,
		"token":      token,
		"otp_code":   otpCode, // Remove this in production
		"expires_in": "5 minutes",
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user from database
	var user models.User
	err := h.db.QueryRow(`
		SELECT id, email, password_hash, full_name, avatar_url, status, balance, total_spent, created_at, updated_at
		FROM users WHERE email = $1
	`, req.Email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.AvatarURL, &user.Status, &user.Balance, &user.TotalSpent, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate JWT token
	token, err := utils.GenerateJWT(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	userResponse := models.UserResponse{
		ID:         user.ID,
		Email:      user.Email,
		FullName:   user.FullName,
		AvatarURL:  user.AvatarURL,
		Status:     user.Status,
		Balance:    user.Balance,
		TotalSpent: user.TotalSpent,
		CreatedAt:  user.CreatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"user":    userResponse,
		"token":   token,
	})
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var user models.User
	err := h.db.QueryRow(`
		SELECT id, email, full_name, avatar_url, status, balance, total_spent, created_at, updated_at
		FROM users WHERE id = $1
	`, userID).Scan(&user.ID, &user.Email, &user.FullName, &user.AvatarURL, &user.Status, &user.Balance, &user.TotalSpent, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	userResponse := models.UserResponse{
		ID:         user.ID,
		Email:      user.Email,
		FullName:   user.FullName,
		AvatarURL:  user.AvatarURL,
		Status:     user.Status,
		Balance:    user.Balance,
		TotalSpent: user.TotalSpent,
		CreatedAt:  user.CreatedAt,
	}

	c.JSON(http.StatusOK, gin.H{"user": userResponse})
}

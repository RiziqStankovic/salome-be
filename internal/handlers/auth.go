package handlers

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"salome-be/internal/config"
	"salome-be/internal/models"
	"salome-be/internal/service"
	"salome-be/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	db           *sql.DB
	emailService *service.MultiProviderEmailService
}

func NewAuthHandler(db *sql.DB) *AuthHandler {
	// Initialize multi-provider email service
	cfg := config.GetConfig()
	var providers []service.EmailProvider

	// Add MailerSend if enabled
	if cfg.Email.MailerSend.Enabled {
		mailerSendService := service.NewEmailService(
			cfg.Email.MailerSend.APIKey,
			cfg.Email.MailerSend.FromEmail,
			cfg.Email.MailerSend.FromName,
		)
		providers = append(providers, mailerSendService)
	}

	// Add Resend if enabled
	if cfg.Email.Resend.Enabled {
		fmt.Printf("Initializing Resend service with API key: %s\n", cfg.Email.Resend.APIKey[:10]+"...")
		resendService := service.NewResendService(
			cfg.Email.Resend.APIKey,
			cfg.Email.Resend.FromEmail,
		)
		providers = append(providers, resendService)
		fmt.Printf("Resend service added to providers. Total providers: %d\n", len(providers))
	} else {
		fmt.Printf("Resend is disabled in config\n")
	}

	emailService := service.NewMultiProviderEmailService(providers)

	return &AuthHandler{
		db:           db,
		emailService: emailService,
	}
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

	// Send OTP via email using configured providers
	cfg := config.GetConfig()
	fmt.Printf("AuthHandler: Checking email providers - MailerSend: %v, Resend: %v\n",
		cfg.Email.MailerSend.Enabled, cfg.Email.Resend.Enabled)

	if cfg.Email.MailerSend.Enabled || cfg.Email.Resend.Enabled {
		otpData := service.OTPEmailData{
			Email:     req.Email,
			Name:      req.FullName,
			OTPCode:   otpCode,
			ExpiresIn: 5, // 5 minutes
		}

		fmt.Printf("AuthHandler: Attempting to send OTP email to %s\n", req.Email)
		err = h.emailService.SendOTPEmail(c.Request.Context(), otpData)
		if err != nil {
			// Log error but don't fail registration
			fmt.Printf("AuthHandler: Failed to send OTP email to %s: %v\n", req.Email, err)
			// Show OTP in response for testing when email fails
			userResponse := models.UserResponse{
				ID:        userID,
				Email:     req.Email,
				FullName:  req.FullName,
				Status:    "pending_verification",
				CreatedAt: time.Now(),
			}

			token, err := utils.GenerateJWT(userID, req.Email)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
				return
			}

			fmt.Printf("AuthHandler: Email failed, showing OTP in response for %s\n", req.Email)
			c.JSON(http.StatusCreated, gin.H{
				"message":    "User created successfully. Please check your email for verification code.",
				"user":       userResponse,
				"token":      token,
				"otp_code":   otpCode, // Show OTP for testing when email fails
				"expires_in": "5 minutes",
				"note":       "Email delivery failed - using OTP for testing",
			})
			return
		} else {
			fmt.Printf("AuthHandler: OTP email sent successfully to %s\n", req.Email)
		}
	} else {
		fmt.Printf("AuthHandler: No email providers enabled, showing OTP in response\n")
		// No email providers enabled, show OTP in response
		userResponse := models.UserResponse{
			ID:        userID,
			Email:     req.Email,
			FullName:  req.FullName,
			Status:    "pending_verification",
			CreatedAt: time.Now(),
		}

		token, err := utils.GenerateJWT(userID, req.Email)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":    "User created successfully. Please check your email for verification code.",
			"user":       userResponse,
			"token":      token,
			"otp_code":   otpCode, // Show OTP when no email providers
			"expires_in": "5 minutes",
			"note":       "No email providers configured - using OTP for testing",
		})
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
		SELECT id, email, password_hash, full_name, avatar_url, status, balance, total_spent, is_admin, created_at, updated_at
		FROM users WHERE email = $1
	`, req.Email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.AvatarURL, &user.Status, &user.Balance, &user.TotalSpent, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt)

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
		IsAdmin:    user.IsAdmin,
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
		SELECT id, email, full_name, avatar_url, status, balance, total_spent, is_admin, created_at, updated_at
		FROM users WHERE id = $1
	`, userID).Scan(&user.ID, &user.Email, &user.FullName, &user.AvatarURL, &user.Status, &user.Balance, &user.TotalSpent, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt)

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
		IsAdmin:    user.IsAdmin,
		CreatedAt:  user.CreatedAt,
	}

	c.JSON(http.StatusOK, gin.H{"user": userResponse})
}

// ChangePasswordWithOTP changes user password using OTP verification
func (h *AuthHandler) ChangePasswordWithOTP(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		NewPassword string `json:"new_password" binding:"required,min=6"`
		OTPCode     string `json:"otp_code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check password change limit (2 times per month)
	var changeCount int
	err := h.db.QueryRow(`
		SELECT COUNT(*) FROM password_changes 
		WHERE user_id = $1 AND created_at >= NOW() - INTERVAL '30 days'
	`, userID).Scan(&changeCount)

	if err != nil {
		// If table doesn't exist, create it and continue
		if err == sql.ErrNoRows {
			// Table might not exist, we'll create it later
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check password change limit"})
			return
		}
	}

	if changeCount >= 2 {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Anda hanya bisa mengubah password maksimal 2 kali dalam sebulan"})
		return
	}

	// Get user email for OTP verification
	var userEmail string
	err = h.db.QueryRow("SELECT email FROM users WHERE id = $1", userID).Scan(&userEmail)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Verify OTP
	var otp models.OTP
	err = h.db.QueryRow(`
		SELECT id, user_id, email, otp_code, purpose, expires_at, is_used, attempts, created_at
		FROM otps 
		WHERE email = $1 AND otp_code = $2 AND purpose = $3
		ORDER BY created_at DESC
		LIMIT 1
	`, userEmail, req.OTPCode, "password_reset").Scan(
		&otp.ID, &otp.UserID, &otp.Email, &otp.OTPCode,
		&otp.Purpose, &otp.ExpiresAt, &otp.IsUsed, &otp.Attempts, &otp.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid OTP code"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Check if OTP is already used
	if otp.IsUsed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OTP code has already been used"})
		return
	}

	// Check if OTP is expired
	if time.Now().After(otp.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OTP code has expired"})
		return
	}

	// Check attempt limit
	if otp.Attempts >= 5 {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many failed attempts"})
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Start transaction
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Update password
	_, err = tx.Exec(`
		UPDATE users 
		SET password_hash = $1, updated_at = $2 
		WHERE id = $3
	`, string(hashedPassword), time.Now(), userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	// Log password change
	_, err = tx.Exec(`
		INSERT INTO password_changes (id, user_id, created_at) 
		VALUES ($1, $2, $3)
	`, uuid.New().String(), userID, time.Now())

	if err != nil {
		// If table doesn't exist, create it
		_, createErr := tx.Exec(`
			CREATE TABLE IF NOT EXISTS password_changes (
				id VARCHAR(50) PRIMARY KEY,
				user_id UUID NOT NULL REFERENCES users(id),
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if createErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create password_changes table"})
			return
		}

		// Try to insert again
		_, err = tx.Exec(`
			INSERT INTO password_changes (id, user_id, created_at) 
			VALUES ($1, $2, $3)
		`, uuid.New().String(), userID, time.Now())

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to log password change"})
			return
		}
	}

	// Mark OTP as used
	_, err = tx.Exec(`
		UPDATE otps SET is_used = true WHERE id = $1
	`, otp.ID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark OTP as used"})
		return
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete password change"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password berhasil diubah"})
}

// ResetPasswordWithOTP resets user password using OTP verification (no auth required)
func (h *AuthHandler) ResetPasswordWithOTP(c *gin.Context) {
	var req struct {
		Email       string `json:"email" binding:"required,email"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
		OTPCode     string `json:"otp_code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID by email
	var userID uuid.UUID
	err := h.db.QueryRow("SELECT id FROM users WHERE email = $1", req.Email).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Verify OTP
	var otp models.OTP
	err = h.db.QueryRow(`
		SELECT id, user_id, email, otp_code, purpose, expires_at, is_used, attempts, created_at
		FROM otps 
		WHERE email = $1 AND otp_code = $2 AND purpose = $3
		ORDER BY created_at DESC
		LIMIT 1
	`, req.Email, req.OTPCode, "password_reset").Scan(
		&otp.ID, &otp.UserID, &otp.Email, &otp.OTPCode,
		&otp.Purpose, &otp.ExpiresAt, &otp.IsUsed, &otp.Attempts, &otp.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Printf("Invalid OTP code for email: %s, OTP: %s\n", req.Email, req.OTPCode)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid OTP code"})
			return
		}
		fmt.Printf("Database error checking OTP: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Check if OTP is already used
	if otp.IsUsed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OTP code has already been used"})
		return
	}

	// Check if OTP is expired
	if time.Now().After(otp.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OTP code has expired"})
		return
	}

	// Check attempt limit
	if otp.Attempts >= 6 {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Terlalu banyak percobaan. Tunggu 2 jam sebelum mencoba lagi."})
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Start transaction
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Update password
	_, err = tx.Exec(`
		UPDATE users 
		SET password_hash = $1, updated_at = $2 
		WHERE id = $3
	`, string(hashedPassword), time.Now(), userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	// Mark OTP as used
	_, err = tx.Exec(`
		UPDATE otps SET is_used = true WHERE id = $1
	`, otp.ID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark OTP as used"})
		return
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete password reset"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password berhasil diubah"})
}

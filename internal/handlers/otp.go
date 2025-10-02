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

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OTPHandler struct {
	db           *sql.DB
	emailService *service.MultiProviderEmailService
}

func NewOTPHandler(db *sql.DB) *OTPHandler {
	// Initialize multi-provider email service
	cfg := config.GetConfig()
	var providers []service.EmailProvider

	fmt.Printf("OTPHandler: Initializing email providers - MailerSend: %v, Resend: %v\n",
		cfg.Email.MailerSend.Enabled, cfg.Email.Resend.Enabled)

	// Add MailerSend if enabled
	if cfg.Email.MailerSend.Enabled {
		mailerSendService := service.NewEmailService(
			cfg.Email.MailerSend.APIKey,
			cfg.Email.MailerSend.FromEmail,
			cfg.Email.MailerSend.FromName,
		)
		providers = append(providers, mailerSendService)
		fmt.Printf("OTPHandler: MailerSend provider added\n")
	}

	// Add Resend if enabled
	if cfg.Email.Resend.Enabled {
		resendService := service.NewResendService(
			cfg.Email.Resend.APIKey,
			cfg.Email.Resend.FromEmail,
		)
		providers = append(providers, resendService)
		fmt.Printf("OTPHandler: Resend provider added\n")
	}

	emailService := service.NewMultiProviderEmailService(providers)
	fmt.Printf("OTPHandler: Email service initialized with %d providers\n", len(providers))

	return &OTPHandler{
		db:           db,
		emailService: emailService,
	}
}

// generateOTP generates a random 6-digit OTP code
func (h *OTPHandler) generateOTP() (string, error) {
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

// GenerateOTP generates and stores a new OTP code
func (h *OTPHandler) GenerateOTP(c *gin.Context) {
	var req models.OTPGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user exists
	var userID uuid.UUID
	err := h.db.QueryRow("SELECT id FROM users WHERE email = $1", req.Email).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error 1"})
		return
	}

	// Check rate limiting - count OTPs generated in the last 2 hours
	var otpCount int
	var latestRateLimitResetAt *time.Time

	// Count OTPs generated in the last 2 hours
	err = h.db.QueryRow(`
		SELECT COUNT(*), MAX(rate_limit_reset_at)
		FROM otps
		WHERE email = $1
		AND created_at > NOW() - INTERVAL '2 hours'
		`, req.Email).Scan(&otpCount, &latestRateLimitResetAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking rate limit"})
		return
	}

	// Check if there's an active rate limit reset time
	if latestRateLimitResetAt != nil && time.Now().Before(*latestRateLimitResetAt) {
		remainingTime := time.Until(*latestRateLimitResetAt)
		hours := int(remainingTime.Hours())
		minutes := int(remainingTime.Minutes()) % 60

		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":       "Rate limit exceeded",
			"message":     fmt.Sprintf("Terlalu banyak permintaan OTP. Silakan coba lagi dalam %d jam %d menit", hours, minutes),
			"retry_after": remainingTime.Seconds(),
			"reset_at":    latestRateLimitResetAt,
		})
		return
	}

	// Check if user has exceeded 3 attempts in the last 2 hours
	if otpCount >= 3 {
		// Set rate limit for 2 hours from now
		resetAt := time.Now().Add(2 * time.Hour)

		// Update all OTPs for this email with rate limit
		_, err = h.db.Exec(`
			UPDATE otps
			SET rate_limit_reset_at = $1
			WHERE email = $2
		`, resetAt, req.Email)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set rate limit"})
			return
		}

		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":       "Rate limit exceeded",
			"message":     "Terlalu banyak permintaan OTP. Silakan coba lagi dalam 2 jam",
			"retry_after": 2 * 3600, // 6 hours in seconds
			"reset_at":    resetAt,
		})
		return
	}

	// Generate OTP code
	otpCode, err := h.generateOTP()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OTP"})
		return
	}

	// Clean up old unused OTPs for this user and purpose
	_, err = h.db.Exec(`
		DELETE FROM otps 
		WHERE user_id = $1 AND purpose = $2 AND (is_used = true OR expires_at < NOW())
	`, userID, req.Purpose)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clean up old OTPs"})
		return
	}

	// Check if there's already an active OTP
	var existingOTP string
	err = h.db.QueryRow(`
		SELECT id FROM otps 
		WHERE user_id = $1 AND purpose = $2 AND is_used = false AND expires_at > NOW()
	`, userID, req.Purpose).Scan(&existingOTP)

	if err == nil {
		// Active OTP exists, return it
		c.JSON(http.StatusOK, gin.H{
			"message":    "OTP already exists and is still valid",
			"expires_in": "5 minutes",
		})
		return
	}

	// Create new OTP
	otpID := uuid.New().String()
	expiresAt := time.Now().Add(5 * time.Minute) // OTP expires in 5 minutes

	_, err = h.db.Exec(`
		INSERT INTO otps (id, user_id, email, otp_code, purpose, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, otpID, userID, req.Email, otpCode, req.Purpose, expiresAt, time.Now())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save OTP"})
		return
	}

	// Get user name for email
	var userName string
	err = h.db.QueryRow("SELECT full_name FROM users WHERE id = $1", userID).Scan(&userName)
	if err != nil {
		fmt.Printf("Failed to get user name: %v\n", err)
		userName = "User" // fallback
	}

	// Send OTP via email
	otpData := service.OTPEmailData{
		Email:     req.Email,
		Name:      userName,
		OTPCode:   otpCode,
		ExpiresIn: 5, // 5 minutes
	}

	fmt.Printf("OTPHandler: Attempting to send OTP email to %s\n", req.Email)
	fmt.Printf("OTPHandler: Email service available: %v\n", h.emailService != nil)

	err = h.emailService.SendOTPEmail(c.Request.Context(), otpData)
	if err != nil {
		// Log error but don't fail OTP generation
		fmt.Printf("OTPHandler: Failed to send OTP email to %s: %v\n", req.Email, err)
		// For development, we'll still return the OTP in response
		// In production, you might want to return an error or retry
	} else {
		fmt.Printf("OTPHandler: Successfully sent OTP email to %s\n", req.Email)
	}

	response := models.OTPResponse{
		ID:        otpID,
		Email:     req.Email,
		Purpose:   req.Purpose,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "OTP generated successfully",
		"otp":        response,
		"otp_code":   otpCode, // Keep for development/testing
		"expires_in": "5 minutes",
	})
}

// VerifyOTP verifies an OTP code
func (h *OTPHandler) VerifyOTP(c *gin.Context) {
	var req models.OTPVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find the OTP
	var otp models.OTP
	err := h.db.QueryRow(`
		SELECT id, user_id, email, otp_code, purpose, expires_at, is_used, attempts, created_at
		FROM otps 
		WHERE email = $1 AND otp_code = $2 AND purpose = $3
		ORDER BY created_at DESC
		LIMIT 1
	`, req.Email, req.OTPCode, req.Purpose).Scan(
		&otp.ID, &otp.UserID, &otp.Email, &otp.OTPCode, &otp.Purpose,
		&otp.ExpiresAt, &otp.IsUsed, &otp.Attempts, &otp.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusBadRequest, gin.H{
				"valid":   false,
				"message": "Invalid OTP code",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error 2"})
		return
	}

	// Check if OTP is already used
	if otp.IsUsed {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid":   false,
			"message": "OTP code has already been used",
		})
		return
	}

	// Check if OTP is expired
	if time.Now().After(otp.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid":   false,
			"message": "OTP code has expired",
		})
		return
	}

	// Check attempt limit (max 6 attempts) - but reset after 2 hours
	if otp.Attempts >= 6 {
		// Check if 2 hours have passed since OTP creation
		if time.Since(otp.CreatedAt) < 2*time.Hour {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"valid":   false,
				"message": "Terlalu banyak percobaan. Tunggu 2 jam sebelum mencoba lagi.",
			})
			return
		} else {
			// Reset attempts after 2 hours
			_, err = h.db.Exec(`
				UPDATE otps SET attempts = 0 WHERE id = $1
			`, otp.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset attempts"})
				return
			}
			// Update local otp object
			otp.Attempts = 0
		}
	}

	// Increment attempts
	_, err = h.db.Exec(`
		UPDATE otps SET attempts = attempts + 1 WHERE id = $1
	`, otp.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update attempts"})
		return
	}

	// Mark OTP as used only for email verification, not for password reset
	if req.Purpose == "email_verification" {
		_, err = h.db.Exec(`
			UPDATE otps SET is_used = true WHERE id = $1
		`, otp.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark OTP as used"})
			return
		}
	}

	// Update user status to active if it's email verification
	if req.Purpose == "email_verification" {
		_, err = h.db.Exec(`
			UPDATE users SET status = 'active', updated_at = $1 WHERE id = $2
		`, time.Now(), otp.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user status"})
			return
		}

		// Send welcome email after successful verification
		cfg := config.GetConfig()
		if cfg.Email.MailerSend.Enabled {
			// Get user details for welcome email
			var userFullName string
			err = h.db.QueryRow("SELECT full_name FROM users WHERE id = $1", otp.UserID).Scan(&userFullName)
			if err == nil {
				err = h.emailService.SendWelcomeEmail(c.Request.Context(), otp.Email, userFullName)
				if err != nil {
					// Log error but don't fail verification
					fmt.Printf("Failed to send welcome email to %s: %v\n", otp.Email, err)
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":   true,
		"message": "OTP verified successfully",
		"user_id": otp.UserID,
	})
}

// ResendOTP resends OTP for the same purpose
func (h *OTPHandler) ResendOTP(c *gin.Context) {
	var req models.OTPGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user exists
	var userID uuid.UUID
	err := h.db.QueryRow("SELECT id FROM users WHERE email = $1", req.Email).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error 3"})
		return
	}

	// Check rate limiting for verify attempts - count verify attempts in the last hour
	var verifyAttempts int
	var latestVerifyRateLimitResetAt *time.Time

	// Count verify attempts in the last hour
	err = h.db.QueryRow(`
		SELECT COUNT(*), MAX(rate_limit_reset_at)
		FROM otps
		WHERE email = $1
			AND purpose = $2
			AND created_at > NOW() - INTERVAL '1 hour'
			AND attempts > 0
		`, req.Email, req.Purpose).Scan(&verifyAttempts, &latestVerifyRateLimitResetAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking verify rate limit"})
		return
	}

	// Check if there's an active verify rate limit reset time
	if latestVerifyRateLimitResetAt != nil && time.Now().Before(*latestVerifyRateLimitResetAt) {
		remainingTime := time.Until(*latestVerifyRateLimitResetAt)
		minutes := int(remainingTime.Minutes())

		c.JSON(http.StatusTooManyRequests, gin.H{
			"valid":       false,
			"error":       "Verify rate limit exceeded",
			"message":     fmt.Sprintf("Terlalu banyak percobaan verifikasi. Silakan coba lagi dalam %d menit", minutes),
			"retry_after": remainingTime.Seconds(),
			"reset_at":    latestVerifyRateLimitResetAt,
		})
		return
	}

	// Check if user has exceeded 10 verify attempts in the last hour
	if verifyAttempts >= 10 {
		// Set rate limit for 1 hour from now
		resetAt := time.Now().Add(1 * time.Hour)

		// Update all OTPs for this email with rate limit
		_, err = h.db.Exec(`
				UPDATE otps
				SET rate_limit_reset_at = $1
				WHERE email = $2 AND purpose = $3
			`, resetAt, req.Email, req.Purpose)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set verify rate limit"})
			return
		}

		c.JSON(http.StatusTooManyRequests, gin.H{
			"valid":       false,
			"error":       "Verify rate limit exceeded",
			"message":     "Terlalu banyak percobaan verifikasi. Silakan coba lagi dalam 1 jam",
			"retry_after": 3600, // 1 hour in seconds
			"reset_at":    resetAt,
		})
		return
	}

	// Check rate limiting - count OTPs generated in the last 6 hours
	var otpCount int
	var latestRateLimitResetAt *time.Time

	// Count OTPs generated in the last 6 hours
	err = h.db.QueryRow(`
		SELECT COUNT(*), MAX(rate_limit_reset_at)
		FROM otps
		WHERE email = $1
		AND created_at > NOW() - INTERVAL '6 hours'
	`, req.Email).Scan(&otpCount, &latestRateLimitResetAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking rate limit"})
		return
	}

	// Check if there's an active rate limit reset time
	if latestRateLimitResetAt != nil && time.Now().Before(*latestRateLimitResetAt) {
		remainingTime := time.Until(*latestRateLimitResetAt)
		hours := int(remainingTime.Hours())
		minutes := int(remainingTime.Minutes()) % 60

		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":       "Rate limit exceeded",
			"message":     fmt.Sprintf("Terlalu banyak permintaan OTP. Silakan coba lagi dalam %d jam %d menit", hours, minutes),
			"retry_after": remainingTime.Seconds(),
			"reset_at":    latestRateLimitResetAt,
		})
		return
	}

	// Check if user has exceeded 6 attempts in the last 6 hours
	if otpCount >= 3 {
		// Set rate limit for 6 hours from now
		resetAt := time.Now().Add(2 * time.Hour)

		// Update all OTPs for this email with rate limit
		_, err = h.db.Exec(`
				UPDATE otps
				SET rate_limit_reset_at = $1
				WHERE email = $2
			`, resetAt, req.Email)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set rate limit"})
			return
		}

		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":       "Rate limit exceeded",
			"message":     "Terlalu banyak permintaan OTP. Silakan coba lagi dalam 2 jam",
			"retry_after": 2 * 3600, // 6 hours in seconds
			"reset_at":    resetAt,
		})
		return
	}

	// Delete existing unused OTPs for this user and purpose
	_, err = h.db.Exec(`
		DELETE FROM otps 
		WHERE user_id = $1 AND purpose = $2 AND is_used = false
	`, userID, req.Purpose)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clean up old OTPs"})
		return
	}

	// Generate new OTP
	otpCode, err := h.generateOTP()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OTP"})
		return
	}

	// Create new OTP
	otpID := uuid.New().String()
	expiresAt := time.Now().Add(5 * time.Minute)

	_, err = h.db.Exec(`
		INSERT INTO otps (id, user_id, email, otp_code, purpose, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, otpID, userID, req.Email, otpCode, req.Purpose, expiresAt, time.Now())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save OTP"})
		return
	}

	// Send OTP via email using configured providers
	cfg := config.GetConfig()
	fmt.Printf("OTPHandler: Checking email providers for resend - MailerSend: %v, Resend: %v\n",
		cfg.Email.MailerSend.Enabled, cfg.Email.Resend.Enabled)

	if cfg.Email.MailerSend.Enabled || cfg.Email.Resend.Enabled {
		// Get user details for email
		var userFullName string
		err = h.db.QueryRow("SELECT full_name FROM users WHERE id = $1", userID).Scan(&userFullName)
		if err != nil {
			userFullName = "User" // fallback
		}

		otpData := service.OTPEmailData{
			Email:     req.Email,
			Name:      userFullName,
			OTPCode:   otpCode,
			ExpiresIn: 5, // 5 minutes
		}

		fmt.Printf("OTPHandler: Attempting to resend OTP email to %s\n", req.Email)
		err = h.emailService.SendOTPEmail(c.Request.Context(), otpData)
		if err != nil {
			// Log error but don't fail resend
			fmt.Printf("OTPHandler: Failed to resend OTP email to %s: %v\n", req.Email, err)
			// Show OTP in response for testing when email fails
			response := models.OTPResponse{
				ID:        otpID,
				Email:     req.Email,
				Purpose:   req.Purpose,
				ExpiresAt: expiresAt,
				CreatedAt: time.Now(),
			}

			fmt.Printf("OTPHandler: Email failed, showing OTP in response for %s\n", req.Email)
			c.JSON(http.StatusCreated, gin.H{
				"message":    "OTP resent successfully. Please check your email for verification code.",
				"otp":        response,
				"otp_code":   otpCode, // Show OTP for testing when email fails
				"expires_in": "5 minutes",
				"note":       "Email delivery failed - using OTP for testing",
			})
			return
		} else {
			fmt.Printf("OTPHandler: OTP email resent successfully to %s\n", req.Email)
		}
	} else {
		fmt.Printf("OTPHandler: No email providers enabled or not email verification purpose\n")
		// No email providers enabled, show OTP in response
		response := models.OTPResponse{
			ID:        otpID,
			Email:     req.Email,
			Purpose:   req.Purpose,
			ExpiresAt: expiresAt,
			CreatedAt: time.Now(),
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":    "OTP resent successfully. Please check your email for verification code.",
			"otp":        response,
			"otp_code":   otpCode, // Show OTP when no email providers
			"expires_in": "5 minutes",
			"note":       "No email providers configured - using OTP for testing",
		})
		return
	}

	response := models.OTPResponse{
		ID:        otpID,
		Email:     req.Email,
		Purpose:   req.Purpose,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "OTP resent successfully. Please check your email for verification code.",
		"otp":        response,
		"expires_in": "5 minutes",
	})
}

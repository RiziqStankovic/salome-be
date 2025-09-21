package middleware

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// OTPRateLimit checks if user has exceeded OTP generation rate limit
func OTPRateLimit(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get email from request body
		var requestBody struct {
			Email string `json:"email"`
		}

		if err := c.ShouldBindJSON(&requestBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			c.Abort()
			return
		}

		email := requestBody.Email
		if email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email is required"})
			c.Abort()
			return
		}

		// Check current rate limit status
		var rateLimitCount int
		var rateLimitResetAt *time.Time

		err := db.QueryRow(`
			SELECT COALESCE(rate_limit_count, 0), rate_limit_reset_at 
			FROM otps 
			WHERE email = $1 
			ORDER BY created_at DESC 
			LIMIT 1
		`, email).Scan(&rateLimitCount, &rateLimitResetAt)

		if err != nil && err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error 4"})
			c.Abort()
			return
		}

		// Check if rate limit is active
		if rateLimitResetAt != nil && time.Now().Before(*rateLimitResetAt) {
			// Rate limit is still active
			remainingTime := time.Until(*rateLimitResetAt)
			hours := int(remainingTime.Hours())
			minutes := int(remainingTime.Minutes()) % 60

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"message":     fmt.Sprintf("Terlalu banyak permintaan OTP. Silakan coba lagi dalam %d jam %d menit", hours, minutes),
				"retry_after": remainingTime.Seconds(),
				"reset_at":    rateLimitResetAt,
			})
			c.Abort()
			return
		}

		// Check if user has exceeded 6 attempts in the last 6 hours
		if rateLimitCount >= 6 {
			// Set rate limit for 6 hours from now
			resetAt := time.Now().Add(6 * time.Hour)

			// Update all OTPs for this email with rate limit
			_, err = db.Exec(`
				UPDATE otps 
				SET rate_limit_count = 6, rate_limit_reset_at = $1 
				WHERE email = $2
			`, resetAt, email)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set rate limit"})
				c.Abort()
				return
			}

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"message":     "Terlalu banyak permintaan OTP. Silakan coba lagi dalam 6 jam",
				"retry_after": 6 * 3600, // 6 hours in seconds
				"reset_at":    resetAt,
			})
			c.Abort()
			return
		}

		// Store email in context for use in handler
		c.Set("email", email)
		c.Next()
	}
}

// OTPVerifyRateLimit checks if user has exceeded OTP verification rate limit
func OTPVerifyRateLimit(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get email from request body
		var requestBody struct {
			Email string `json:"email"`
		}

		if err := c.ShouldBindJSON(&requestBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			c.Abort()
			return
		}

		email := requestBody.Email
		if email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email is required"})
			c.Abort()
			return
		}

		// Check current rate limit status for verification
		var rateLimitCount int
		var rateLimitResetAt *time.Time

		err := db.QueryRow(`
			SELECT COALESCE(rate_limit_count, 0), rate_limit_reset_at 
			FROM otps 
			WHERE email = $1 
			ORDER BY created_at DESC 
			LIMIT 1
		`, email).Scan(&rateLimitCount, &rateLimitResetAt)

		if err != nil && err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error 5"})
			c.Abort()
			return
		}

		// Check if rate limit is active
		if rateLimitResetAt != nil && time.Now().Before(*rateLimitResetAt) {
			// Rate limit is still active
			remainingTime := time.Until(*rateLimitResetAt)
			hours := int(remainingTime.Hours())
			minutes := int(remainingTime.Minutes()) % 60

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"message":     fmt.Sprintf("Terlalu banyak percobaan verifikasi. Silakan coba lagi dalam %d jam %d menit", hours, minutes),
				"retry_after": remainingTime.Seconds(),
				"reset_at":    rateLimitResetAt,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

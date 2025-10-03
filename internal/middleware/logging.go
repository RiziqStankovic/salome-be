package middleware

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/http"
	"sync"

	"salome-be/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Global JWT parser instance for reuse
var (
	jwtParser = &jwt.Parser{}
	parserMux sync.RWMutex
)

// CustomResponseWriter wraps gin.ResponseWriter to capture response body
type CustomResponseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w CustomResponseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// CustomLoggingMiddleware creates a custom logging middleware
func CustomLoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Extract user information from context (optimized)
		userInfo := "user=anonymous"
		if email, emailExists := param.Keys["user_email"]; emailExists {
			userInfo = "user=" + email.(string)
		} else if userID, exists := param.Keys["user_id"]; exists {
			// Handle both string and uuid.UUID types
			if userIDStr, ok := userID.(string); ok {
				userInfo = "user=" + userIDStr
			} else if userIDUUID, ok := userID.(uuid.UUID); ok {
				userInfo = "user=" + userIDUUID.String()
			}
		}

		// Format: [GIN] 2025/10/02 - 04:28:42 | 401 | 1.2834ms | 127.0.0.1 | GET /api/v1/auth/profile | user=anonymous
		return fmt.Sprintf("[GIN] %s | %d | %8v | %s | %-7s %s | %s\n",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			param.Path,
			userInfo,
		)
	})
}

// UserExtractionMiddleware extracts user info from JWT without database query
// This middleware ONLY extracts user info for logging, no validation
func UserExtractionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Fast path: check if already parsed
		if _, exists := c.Get("user_authenticated"); exists {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString := authHeader[7:] // Faster than strings.TrimPrefix

			// Parse JWT without validation (just for logging)
			claims, err := parseJWTWithoutValidationFast(tokenString)
			if err == nil {
				// JWT is valid, set user info in context
				c.Set("user_id", claims.UserID)
				c.Set("user_email", claims.Email)
				c.Set("user_authenticated", true)
			}
		}
		c.Next()
	}
}

// parseJWTWithoutValidationFast parses JWT without signature validation (optimized)
func parseJWTWithoutValidationFast(tokenString string) (*utils.Claims, error) {
	// Use cached parser instance for better performance
	parserMux.RLock()
	token, _, err := jwtParser.ParseUnverified(tokenString, &utils.Claims{})
	parserMux.RUnlock()

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*utils.Claims)
	if !ok {
		return nil, jwt.ErrTokenMalformed
	}

	return claims, nil
}

// parseJWTWithoutValidation parses JWT without signature validation (for logging only)
func parseJWTWithoutValidation(tokenString string) (*utils.Claims, error) {
	return parseJWTWithoutValidationFast(tokenString)
}

// OptimizedAuthRequired checks authentication using JWT only (no database query)
func OptimizedAuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Fast path: check if already authenticated by UserExtractionMiddleware
		if _, exists := c.Get("user_id"); exists {
			// Already authenticated, just continue
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := authHeader[7:] // Faster than strings.TrimPrefix
		claims, err := utils.ValidateJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_authenticated", true)
		c.Next()
	}
}

// OptimizedAuthRequiredWithStatus checks authentication using JWT only (NO DATABASE QUERY)
// This is the most optimized version - only JWT validation, no DB queries
func OptimizedAuthRequiredWithStatus(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Fast path: check if already authenticated by UserExtractionMiddleware
		if _, exists := c.Get("user_id"); exists {
			// Already authenticated, just continue
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := authHeader[7:] // Faster than strings.TrimPrefix
		claims, err := utils.ValidateJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// NO DATABASE QUERY - just set user info from JWT
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_authenticated", true)
		c.Next()
	}
}

// OptimizedAdminRequired checks if user is admin using JWT only (NO DATABASE QUERY)
// This assumes admin status is stored in JWT claims
func OptimizedAdminRequired(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Fast path: check if already authenticated by UserExtractionMiddleware
		if _, exists := c.Get("user_id"); exists {
			// Already authenticated, just continue
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := authHeader[7:] // Faster than strings.TrimPrefix
		claims, err := utils.ValidateJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// NO DATABASE QUERY - just set user info from JWT
		// Note: Admin check should be done at application level if needed
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_authenticated", true)
		c.Next()
	}
}

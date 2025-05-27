package service

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// APIKeyMiddleware creates a middleware that validates API keys
// excludedPaths: paths that don't require API key validation
func APIKeyMiddleware(excludedPaths ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if current path should be excluded from API key validation
		currentPath := c.Request.URL.Path
		for _, excludedPath := range excludedPaths {
			// Support both exact matches and prefix matches (for wildcard paths)
			if currentPath == excludedPath || strings.HasPrefix(currentPath, excludedPath) {
				c.Next()
				return
			}
		}

		// Get expected API key from environment
		expectedAPIKey := os.Getenv("RUNDJ_API_KEY")

		if expectedAPIKey == "" {
			logger.Error("API key not configured in environment")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "API key validation is not properly configured",
				"status":  "error",
				"message": "Server configuration error",
			})
			c.Abort()
			return
		}

		// Check for API key in header first
		apiKey := c.GetHeader("X-API-Key")

		// If not in header, check query parameter
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}

		// Validate API key
		if apiKey == "" {
			logger.Warn("API key missing",
				zap.String("path", currentPath),
				zap.String("method", c.Request.Method),
				zap.String("ip", c.ClientIP()),
			)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "API key required",
				"status":  "error",
				"message": "Please provide a valid API key in X-API-Key header or api_key query parameter",
			})
			c.Abort()
			return
		}

		if apiKey != expectedAPIKey {
			logger.Warn("Invalid API key provided",
				zap.String("path", currentPath),
				zap.String("method", c.Request.Method),
				zap.String("ip", c.ClientIP()),
				zap.String("providedKey", apiKey[:min(len(apiKey), 8)]+"..."), // Log only first 8 chars for security
			)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid API key",
				"status":  "error",
				"message": "The provided API key is not valid",
			})
			c.Abort()
			return
		}

		// API key is valid, log successful authentication and continue
		logger.Debug("API key validated successfully",
			zap.String("path", currentPath),
			zap.String("method", c.Request.Method),
			zap.String("ip", c.ClientIP()),
		)
		c.Next()
	}
}

// Helper function for minimum of two integers (for Go versions < 1.21)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RequireAPIKey is a convenience function for protecting specific route groups
func RequireAPIKey() gin.HandlerFunc {
	return APIKeyMiddleware()
}

package middleware

import (
	"email-service/internal/config"
	"email-service/pkg/jwt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	AuthorizationHeader = "Authorization"
	BearerSchema        = "Bearer"
	UserIDKey           = "userID"
)

// AuthMiddleware creates a gin middleware for JWT authentication.
func AuthMiddleware(jwtConfig config.JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthorizationHeader)
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is missing"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != BearerSchema {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			return
		}

		tokenString := parts[1]
		claims, err := jwt.ValidateToken(tokenString, jwtConfig.Secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		// Set user ID in context for downstream handlers
		c.Set(UserIDKey, claims.UserID)

		c.Next()
	}
}

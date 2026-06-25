package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthRequired is a Gin middleware that validates the JWT access token.
func AuthRequired(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "Missing Authorization header"},
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid Authorization header format"},
			})
			return
		}

		claims, err := ValidateAccessToken(parts[1], secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid or expired token"},
			})
			return
		}

		// Store claims in context for downstream handlers
		c.Set("claims", claims)
		c.Next()
	}
}

// AdminRequired is a Gin middleware that requires the operator to have the admin role.
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetClaims(c)
		if claims == nil || claims.Role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{"code": "FORBIDDEN", "message": "Admin role required"},
			})
			return
		}
		c.Next()
	}
}

// GetClaims extracts the JWT claims from the Gin context.
func GetClaims(c *gin.Context) *Claims {
	val, exists := c.Get("claims")
	if !exists {
		return nil
	}
	claims, ok := val.(*Claims)
	if !ok {
		return nil
	}
	return claims
}

package middleware

import (
	"net/http"

	"auth/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RequireRole middleware pour vérifier qu'un utilisateur a un rôle spécifique
func RequireRole(db *gorm.DB, requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		var user models.User
		if err := db.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		if !user.HasRole(requiredRole) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":         "Insufficient permissions",
				"required_role": requiredRole,
			})
			c.Abort()
			return
		}

		c.Set("user_roles", user.Roles)
		c.Next()
	}
}

// RequireAnyRole middleware pour vérifier qu'un utilisateur a au moins un des rôles spécifiés
func RequireAnyRole(db *gorm.DB, roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		var user models.User
		if err := db.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		hasRole := false
		for _, role := range roles {
			if user.HasRole(role) {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{
				"error":          "Insufficient permissions",
				"required_roles": roles,
			})
			c.Abort()
			return
		}

		c.Set("user_roles", user.Roles)
		c.Next()
	}
}

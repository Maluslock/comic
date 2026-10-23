package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/repository"
)

func AdminAuthRequired(admins *repository.AdminRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing admin token"})
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		a, err := admins.GetAdminByToken(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid admin token"})
			return
		}
		c.Set("admin_id", a.ID)
		c.Set("admin_role", a.Role)
		c.Next()
	}
}

// RequireSuper must run after AdminAuthRequired. Admin-account management
// (create / disable / reset password) is a super-admin-only capability: a plain
// admin must not be able to mint accounts or take over another admin's session.
func RequireSuper() gin.HandlerFunc {
	return func(c *gin.Context) {
		if AdminRole(c) != "super" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "super admin required"})
			return
		}
		c.Next()
	}
}

func AdminRole(c *gin.Context) string {
	v, _ := c.Get("admin_role")
	role, _ := v.(string)
	return role
}

func AdminID(c *gin.Context) int64 {
	v, _ := c.Get("admin_id")
	id, _ := v.(int64)
	return id
}

package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/repository"
)

func AuthRequired(users *repository.UserRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		u, err := users.GetUserByToken(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("user_id", u.ID)
		c.Set("user_name", u.Name)
		c.Next()
	}
}

func AuthOptional(users *repository.UserRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			token := strings.TrimPrefix(auth, "Bearer ")
			if u, err := users.GetUserByToken(c.Request.Context(), token); err == nil {
				c.Set("user_id", u.ID)
				c.Set("user_name", u.Name)
			}
		}
		c.Next()
	}
}

func UserID(c *gin.Context) int64 {
	v, _ := c.Get("user_id")
	id, _ := v.(int64)
	return id
}

func UserIDStr(c *gin.Context) string {
	return strconv.FormatInt(UserID(c), 10)
}

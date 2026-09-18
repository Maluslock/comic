package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/repository"
)

func AdminAuditLogger(logs *repository.AdminAuditRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		path := c.Request.URL.Path

		c.Next()

		if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = logs.Insert(ctx, AdminID(c), method, path, c.Writer.Status())
	}
}

package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/middleware"
)

// assertSelf 校验路径/请求体中的 userId 是否等于 token 所属用户。
// 不匹配时写出 403 并返回 false，调用方应立即 return。
func assertSelf(c *gin.Context, userID int64) bool {
	if userID != middleware.UserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot access another user's data"})
		return false
	}
	return true
}

func parseUserID(raw string) int64 {
	id, _ := strconv.ParseInt(raw, 10, 64)
	return id
}

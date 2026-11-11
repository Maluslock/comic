package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/middleware"
	"github.com/Maluslock/comic/server/internal/service"
)

// StylePrefHandler serves GET/PUT /api/v1/me/style-prefs (C-end first-launch picker).
type StylePrefHandler struct {
	svc *service.StylePrefService
}

func NewStylePrefHandler(svc *service.StylePrefService) *StylePrefHandler {
	return &StylePrefHandler{svc: svc}
}

// Get serves GET /api/v1/me/style-prefs → {"tags":[{"id":..,"name":..}],"chosen":bool}.
func (h *StylePrefHandler) Get(c *gin.Context) {
	data, err := h.svc.Get(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, data)
}

// Replace serves PUT /api/v1/me/style-prefs with body {"tagIds":[...]}.
// userID 只从 token 上下文取（middleware.UserID(c)），body/query 里的任何
// userId 字段一律忽略 —— 与全仓 IDOR 防护口径一致。
func (h *StylePrefHandler) Replace(c *gin.Context) {
	var req struct {
		TagIDs []int64 `json:"tagIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	data, err := h.svc.Replace(c.Request.Context(), middleware.UserID(c), req.TagIDs)
	if err != nil {
		if errors.Is(err, service.ErrInvalidStyleSelection) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid style selection"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, data)
}

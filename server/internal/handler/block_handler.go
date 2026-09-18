package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/middleware"
	"github.com/Maluslock/comic/server/internal/service"
)

type BlockHandler struct {
	svc *service.BlockService
}

func NewBlockHandler(svc *service.BlockService) *BlockHandler {
	return &BlockHandler{svc: svc}
}

func (h *BlockHandler) Add(c *gin.Context) {
	var req struct {
		BlockedUserID int64 `json:"blockedUserId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	err := h.svc.Block(c.Request.Context(), middleware.UserID(c), req.BlockedUserID)
	switch {
	case err == nil:
		c.JSON(http.StatusCreated, gin.H{"ok": true})
	case errors.Is(err, service.ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
	case errors.Is(err, service.ErrInvalidReference):
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot block yourself"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

func (h *BlockHandler) Remove(c *gin.Context) {
	blockedID, err := strconv.ParseInt(c.Param("blockedUserId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	if err := h.svc.Unblock(c.Request.Context(), middleware.UserID(c), blockedID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *BlockHandler) List(c *gin.Context) {
	items, err := h.svc.List(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": items})
}

package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/middleware"
	"github.com/Maluslock/comic/server/internal/service"
)

type NotificationHandler struct {
	svc *service.NotificationService
}

func NewNotificationHandler(svc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

func (h *NotificationHandler) Create(c *gin.Context) {
	var req struct {
		Type     string `json:"type" binding:"required"`
		Title    string `json:"title" binding:"required"`
		Content  string `json:"content" binding:"required"`
		LinkType string `json:"linkType"`
		LinkID   *int64 `json:"linkId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	var id int64
	var err error
	if req.LinkType != "" && req.LinkID != nil {
		id, err = h.svc.CreateLinked(c.Request.Context(), middleware.UserID(c), req.Type, req.Title, req.Content, req.LinkType, *req.LinkID)
	} else {
		id, err = h.svc.Create(c.Request.Context(), middleware.UserID(c), req.Type, req.Title, req.Content)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *NotificationHandler) List(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	if userID != middleware.UserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot read another user's notifications"})
		return
	}
	items, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *NotificationHandler) MarkRead(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notification id"})
		return
	}
	if err := h.svc.MarkRead(c.Request.Context(), middleware.UserID(c), id); err != nil {
		if errors.Is(err, service.ErrNotificationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/middleware"
	"github.com/Maluslock/comic/server/internal/service"
)

type ChatHandler struct {
	svc *service.ChatService
}

func NewChatHandler(svc *service.ChatService) *ChatHandler {
	return &ChatHandler{svc: svc}
}

func (h *ChatHandler) CreateSession(c *gin.Context) {
	var req struct {
		OtherUserID int64 `json:"otherUserId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	id, err := h.svc.GetOrCreateSession(c.Request.Context(), middleware.UserID(c), req.OtherUserID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		case errors.Is(err, service.ErrForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "cannot start a chat with this user"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *ChatHandler) ListSessions(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	if !assertSelf(c, userID) {
		return
	}
	items, err := h.svc.ListSessions(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *ChatHandler) ListMessages(c *gin.Context) {
	sessionID, err := strconv.ParseInt(c.Param("sessionId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}
	items, err := h.svc.ListMessagesForUser(c.Request.Context(), sessionID, middleware.UserID(c))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, service.ErrSessionNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *ChatHandler) SendMessage(c *gin.Context) {
	var req struct {
		SessionID int64  `json:"sessionId" binding:"required"`
		Content   string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	id, err := h.svc.SendMessageForUser(c.Request.Context(), req.SessionID, middleware.UserID(c), req.Content)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, service.ErrSessionNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *ChatHandler) MarkRead(c *gin.Context) {
	sessionID, err := strconv.ParseInt(c.Param("sessionId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}
	if err := h.svc.MarkSessionRead(c.Request.Context(), sessionID, middleware.UserID(c)); err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, service.ErrSessionNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *ChatHandler) Unread(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	if userID != middleware.UserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot read another user's unread count"})
		return
	}
	n, err := h.svc.TotalUnread(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": n})
}

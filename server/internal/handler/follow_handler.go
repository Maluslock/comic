package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/middleware"
	"github.com/Maluslock/comic/server/internal/service"
)

type FollowHandler struct {
	svc *service.FollowService
}

func NewFollowHandler(svc *service.FollowService) *FollowHandler {
	return &FollowHandler{svc: svc}
}

func (h *FollowHandler) Follow(c *gin.Context) {
	var req service.FollowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	req.UserID = middleware.UserIDStr(c)

	followed, err := h.svc.Follow(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"followed": followed})
}

func (h *FollowHandler) Unfollow(c *gin.Context) {
	userID := c.Param("userId")
	eventIDStr := c.Param("eventId")
	eventID, err := strconv.ParseInt(eventIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
		return
	}
	if !assertSelf(c, parseUserID(userID)) {
		return
	}

	followed, err := h.svc.Unfollow(c.Request.Context(), service.FollowRequest{
		UserID:  userID,
		EventID: eventID,
	})
	if err != nil {
		if isNotFound(err) {
			c.JSON(http.StatusOK, gin.H{"followed": false})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"followed": !followed})
}

func (h *FollowHandler) ListFollows(c *gin.Context) {
	userID := c.Param("userId")
	if !assertSelf(c, parseUserID(userID)) {
		return
	}

	list, err := h.svc.ListFollows(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": list})
}

func (h *FollowHandler) Subscribe(c *gin.Context) {
	var req service.SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	req.UserID = middleware.UserIDStr(c)

	subscribed, err := h.svc.Subscribe(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"subscribed": subscribed})
}

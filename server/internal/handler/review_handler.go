package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/middleware"
	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/Maluslock/comic/server/internal/service"
)

type ReviewHandler struct {
	svc      *service.ReviewService
	userRepo *repository.UserRepo
}

func NewReviewHandler(svc *service.ReviewService, userRepo *repository.UserRepo) *ReviewHandler {
	return &ReviewHandler{svc: svc, userRepo: userRepo}
}

func (h *ReviewHandler) Create(c *gin.Context) {
	var req service.CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	uid := middleware.UserID(c)
	req.UserID = int32(uid)
	req.UserName = ""
	req.UserAvatar = ""
	if u, err := h.userRepo.GetByID(c.Request.Context(), uid); err == nil {
		req.UserName = u.Name
		req.UserAvatar = u.Avatar
	}

	data, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrReviewNotAllowed):
			c.JSON(http.StatusForbidden, gin.H{"error": "only completed shoots can be reviewed"})
		case errors.Is(err, service.ErrReviewSelf):
			c.JSON(http.StatusForbidden, gin.H{"error": "cannot review yourself"})
		case errors.Is(err, service.ErrReviewDuplicate):
			c.JSON(http.StatusConflict, gin.H{"error": "already reviewed this photographer"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}
	c.JSON(http.StatusCreated, data)
}

func (h *ReviewHandler) ListByPhotographer(c *gin.Context) {
	photographerIDStr := c.Param("photographerId")
	photographerID, err := strconv.Atoi(photographerIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid photographer id"})
		return
	}

	data, err := h.svc.ListByPhotographer(c.Request.Context(), int32(photographerID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *ReviewHandler) MyReviews(c *gin.Context) {
	userID := middleware.UserID(c)
	data, err := h.svc.ListByUser(c.Request.Context(), int32(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, data)
}

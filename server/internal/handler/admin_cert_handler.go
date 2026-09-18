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

type AdminCertHandler struct {
	svc *service.CertApplicationService
}

func NewAdminCertHandler(svc *service.CertApplicationService) *AdminCertHandler {
	return &AdminCertHandler{svc: svc}
}

func (h *AdminCertHandler) List(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	list, total, err := h.svc.List(c.Request.Context(), status, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cert applications unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": list, "total": total})
}

func (h *AdminCertHandler) Detail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid application id"})
		return
	}

	app, err := h.svc.Detail(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrManageNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "cert application not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cert application unavailable"})
		return
	}
	c.JSON(http.StatusOK, app)
}

func (h *AdminCertHandler) Review(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid application id"})
		return
	}
	var req struct {
		Action string `json:"action" binding:"required"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action required"})
		return
	}

	adminID := middleware.AdminID(c)
	if err := h.svc.Review(c.Request.Context(), id, req.Action, req.Reason, adminID); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAction), errors.Is(err, service.ErrReasonRequired):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid review request"})
		case errors.Is(err, service.ErrAlreadyReviewed), errors.Is(err, repository.ErrReviewConflict):
			c.JSON(http.StatusConflict, gin.H{"error": "cert application already reviewed"})
		case errors.Is(err, repository.ErrManageNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "cert application not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "review failed"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

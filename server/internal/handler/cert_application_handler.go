package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/middleware"
	"github.com/Maluslock/comic/server/internal/service"
)

type CertApplicationHandler struct {
	svc *service.CertApplicationService
}

func NewCertApplicationHandler(svc *service.CertApplicationService) *CertApplicationHandler {
	return &CertApplicationHandler{svc: svc}
}

// Submit handles POST /api/v1/photographers/cert-apply (C-end, auth required).
func (h *CertApplicationHandler) Submit(c *gin.Context) {
	userID := middleware.UserID(c)
	var req struct {
		EvidenceImages []string `json:"evidenceImages" binding:"required"`
		EvidenceDesc   string   `json:"evidenceDesc" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "evidenceImages and evidenceDesc required"})
		return
	}

	id, err := h.svc.Submit(c.Request.Context(), userID, req.EvidenceImages, req.EvidenceDesc)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotPhotographer):
			c.JSON(http.StatusForbidden, gin.H{"error": "only photographers can apply for certification"})
		case errors.Is(err, service.ErrAlreadyApplied):
			c.JSON(http.StatusConflict, gin.H{"error": "certification already applied"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "submit failed"})
		}
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// MyApplication handles GET /api/v1/photographers/cert-application (C-end, auth required).
// Returns a slim DTO — evidence images/desc and internal IDs are admin-only.
func (h *CertApplicationHandler) MyApplication(c *gin.Context) {
	userID := middleware.UserID(c)
	app, err := h.svc.GetMyApplication(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrNotPhotographer) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not a photographer"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "application unavailable"})
		return
	}
	if app == nil {
		c.JSON(http.StatusOK, gin.H{"application": nil})
		return
	}
	// 精简 DTO（camelCase）
	c.JSON(http.StatusOK, gin.H{"application": gin.H{
		"id":           app.ID,
		"status":       app.Status,
		"reviewReason": app.ReviewReason,
		"createdAt":    app.CreatedAt,
	}})
}

// MyApplications handles GET /api/v1/photographers/cert-applications (C-end, auth required).
func (h *CertApplicationHandler) MyApplications(c *gin.Context) {
	userID := middleware.UserID(c)
	apps, err := h.svc.MyApplications(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrNotPhotographer) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not a photographer"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "applications unavailable"})
		return
	}
	list := make([]gin.H, 0, len(apps))
	for _, a := range apps {
		list = append(list, gin.H{
			"id":           a.ID,
			"status":       a.Status,
			"reviewReason": a.ReviewReason,
			"createdAt":    a.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"list": list})
}

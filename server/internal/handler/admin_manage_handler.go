package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/Maluslock/comic/server/internal/service"
)

type AdminManageHandler struct {
	svc *service.AdminManageService
}

func NewAdminManageHandler(svc *service.AdminManageService) *AdminManageHandler {
	return &AdminManageHandler{svc: svc}
}

func (h *AdminManageHandler) ListPhotographers(c *gin.Context) {
	var certified *bool
	if v, ok := c.GetQuery("certified"); ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid certified value"})
			return
		}
		certified = &b
	}

	list, err := h.svc.ListPhotographers(c.Request.Context(), certified)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "photographers unavailable"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *AdminManageHandler) Detail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid photographer id"})
		return
	}
	d, err := h.svc.GetPhotographerDetail(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrManageNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "photographer not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "detail unavailable"})
		return
	}
	c.JSON(http.StatusOK, d)
}

func (h *AdminManageHandler) SetCertified(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid photographer id"})
		return
	}
	var req struct {
		Certified bool `json:"certified"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.svc.SetCertified(c.Request.Context(), id, req.Certified); err != nil {
		if errors.Is(err, repository.ErrManageNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "photographer not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AdminManageHandler) ListOrders(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	list, total, err := h.svc.ListOrders(c.Request.Context(), status, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "orders unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": list, "total": total})
}

func (h *AdminManageHandler) SetOrderStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status required"})
		return
	}

	item, err := h.svc.SetOrderStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		if errors.Is(err, service.ErrInvalidTransition) {
			c.JSON(http.StatusConflict, gin.H{"error": "invalid transition"})
			return
		}
		if errors.Is(err, service.ErrBookingNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *AdminManageHandler) ListEvents(c *gin.Context) {
	list, err := h.svc.ListEvents(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "events unavailable"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *AdminManageHandler) SetEventStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
		return
	}
	var req struct {
		DelFlag bool `json:"delFlag"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.svc.SetEventStatus(c.Request.Context(), id, req.DelFlag); err != nil {
		if errors.Is(err, repository.ErrManageNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

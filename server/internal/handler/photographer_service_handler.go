package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/middleware"
	"github.com/Maluslock/comic/server/internal/service"
)

type PhotographerServiceHandler struct {
	svc *service.PhotographerService
}

func NewPhotographerServiceHandler(svc *service.PhotographerService) *PhotographerServiceHandler {
	return &PhotographerServiceHandler{svc: svc}
}

func (h *PhotographerServiceHandler) mapServiceErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "not a photographer"})
	case errors.Is(err, service.ErrServiceForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot modify another photographer's service"})
	case errors.Is(err, service.ErrServiceNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
	case errors.Is(err, service.ErrInvalidPrice):
		c.JSON(http.StatusBadRequest, gin.H{"error": "price must be >= 0"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

func (h *PhotographerServiceHandler) MyServices(c *gin.Context) {
	items, err := h.svc.MyServices(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		h.mapServiceErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": items})
}

func (h *PhotographerServiceHandler) Create(c *gin.Context) {
	var req service.ServiceUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	id, err := h.svc.CreateService(c.Request.Context(), middleware.UserID(c), req)
	if err != nil {
		h.mapServiceErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *PhotographerServiceHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	var req service.ServiceUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if err := h.svc.UpdateService(c.Request.Context(), middleware.UserID(c), id, req); err != nil {
		h.mapServiceErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *PhotographerServiceHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service id"})
		return
	}
	if err := h.svc.DeleteService(c.Request.Context(), middleware.UserID(c), id); err != nil {
		h.mapServiceErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *PhotographerServiceHandler) Templates(c *gin.Context) {
	items, err := h.svc.ListTemplates(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, items)
}

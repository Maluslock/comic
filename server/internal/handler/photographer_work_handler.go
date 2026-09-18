package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/middleware"
	"github.com/Maluslock/comic/server/internal/service"
)

type PhotographerWorkHandler struct {
	svc *service.PhotographerService
}

func NewPhotographerWorkHandler(svc *service.PhotographerService) *PhotographerWorkHandler {
	return &PhotographerWorkHandler{svc: svc}
}

func (h *PhotographerWorkHandler) CreateWork(c *gin.Context) {
	userID := middleware.UserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req struct {
		Title       string   `json:"title" binding:"required"`
		Images      []string `json:"images" binding:"required,min=1"`
		Description string   `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title and at least one image are required"})
		return
	}
	id, err := h.svc.CreateWork(c.Request.Context(), userID, req.Title, req.Images, req.Description)
	if err != nil {
		if errors.Is(err, service.ErrNotPhotographer) {
			c.JSON(http.StatusForbidden, gin.H{"error": "only photographers can publish works"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create work failed"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *PhotographerWorkHandler) UpdateWork(c *gin.Context) {
	userID := middleware.UserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid work id"})
		return
	}
	var req struct {
		Title       string   `json:"title" binding:"required"`
		Images      []string `json:"images" binding:"required,min=1"`
		Description string   `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title and at least one image are required"})
		return
	}
	if err := h.svc.UpdateWork(c.Request.Context(), userID, id, req.Title, req.Images, req.Description); err != nil {
		switch {
		case errors.Is(err, service.ErrNotPhotographer):
			c.JSON(http.StatusForbidden, gin.H{"error": "not a photographer"})
		case errors.Is(err, service.ErrWorkForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "cannot edit another photographer's work"})
		case errors.Is(err, service.ErrWorkNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "work not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "update work failed"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *PhotographerWorkHandler) MyWorks(c *gin.Context) {
	userID := middleware.UserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	works, err := h.svc.MyWorks(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrNotPhotographer) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not a photographer"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list works failed"})
		return
	}
	c.JSON(http.StatusOK, works)
}

func (h *PhotographerWorkHandler) DeleteWork(c *gin.Context) {
	userID := middleware.UserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid work id"})
		return
	}
	if err := h.svc.DeleteWork(c.Request.Context(), userID, id); err != nil {
		switch {
		case errors.Is(err, service.ErrNotPhotographer):
			c.JSON(http.StatusForbidden, gin.H{"error": "not a photographer"})
		case errors.Is(err, service.ErrWorkForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete another photographer's work"})
		case errors.Is(err, service.ErrWorkNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "work not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "delete work failed"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/Maluslock/comic/server/internal/service"
)

type AdminBannerHandler struct {
	svc *service.AdminBannerService
}

func NewAdminBannerHandler(svc *service.AdminBannerService) *AdminBannerHandler {
	return &AdminBannerHandler{svc: svc}
}

type bannerUpsertRequest struct {
	ImageURL  string `json:"imageUrl" binding:"required"`
	Title     string `json:"title" binding:"required"`
	LinkType  string `json:"linkType" binding:"required"`
	LinkID    int32  `json:"linkId"`
	SortOrder int32  `json:"sortOrder"`
}

type bannerUpdateRequest struct {
	bannerUpsertRequest
	IsActive bool `json:"isActive"`
}

func toBannerUpsert(req bannerUpsertRequest) service.BannerUpsert {
	return service.BannerUpsert{
		ImageURL:  req.ImageURL,
		Title:     req.Title,
		LinkType:  req.LinkType,
		LinkID:    req.LinkID,
		SortOrder: req.SortOrder,
	}
}

func (h *AdminBannerHandler) List(c *gin.Context) {
	banners, err := h.svc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "banners unavailable"})
		return
	}
	c.JSON(http.StatusOK, banners)
}

func (h *AdminBannerHandler) Create(c *gin.Context) {
	var req bannerUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "imageUrl, title and linkType are required"})
		return
	}

	id, err := h.svc.Create(c.Request.Context(), toBannerUpsert(req))
	if err != nil {
		if errors.Is(err, service.ErrInvalidLinkType) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid link type"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *AdminBannerHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid banner id"})
		return
	}
	var req bannerUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "imageUrl, title and linkType are required"})
		return
	}

	if err := h.svc.Update(c.Request.Context(), id, toBannerUpsert(req.bannerUpsertRequest), req.IsActive); err != nil {
		if errors.Is(err, service.ErrInvalidLinkType) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid link type"})
			return
		}
		if errors.Is(err, repository.ErrBannerNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "banner not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AdminBannerHandler) SetStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid banner id"})
		return
	}
	var req struct {
		IsActive *bool `json:"isActive" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "isActive required"})
		return
	}

	if err := h.svc.SetStatus(c.Request.Context(), id, *req.IsActive); err != nil {
		if errors.Is(err, repository.ErrBannerNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "banner not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

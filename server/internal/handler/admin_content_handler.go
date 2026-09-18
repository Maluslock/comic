package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/cache"
	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/Maluslock/comic/server/internal/service"
)

type AdminContentHandler struct {
	svc   *service.AdminContentService
	cache *cache.RedisCache
}

func NewAdminContentHandler(svc *service.AdminContentService, cache *cache.RedisCache) *AdminContentHandler {
	return &AdminContentHandler{svc: svc, cache: cache}
}

func parseOptionalID(c *gin.Context, key string) (*int64, bool) {
	raw := c.Query(key)
	if raw == "" {
		return nil, true
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, false
	}
	return &id, true
}

func (h *AdminContentHandler) ListWorks(c *gin.Context) {
	photographerID, ok := parseOptionalID(c, "photographerId")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid photographerId"})
		return
	}
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	list, total, err := h.svc.ListWorks(c.Request.Context(), photographerID, status, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "works unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": list, "total": total})
}

func (h *AdminContentHandler) SetWorkStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid work id"})
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status required"})
		return
	}

	if err := h.svc.SetWorkStatus(c.Request.Context(), id, req.Status); err != nil {
		if errors.Is(err, service.ErrInvalidContentStatus) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
			return
		}
		if errors.Is(err, repository.ErrContentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "work not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	// Works feed the cached home response (featured works) — drop it so the
	// down/up state is reflected immediately instead of after the 5-min TTL.
	h.cache.Delete(c.Request.Context(), "home")
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AdminContentHandler) ListReviews(c *gin.Context) {
	keyword := c.Query("keyword")
	photographerID, ok := parseOptionalID(c, "photographerId")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid photographerId"})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	list, total, err := h.svc.ListReviews(c.Request.Context(), keyword, photographerID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "reviews unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": list, "total": total})
}

func (h *AdminContentHandler) DeleteReview(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid review id"})
		return
	}

	if err := h.svc.DeleteReview(c.Request.Context(), id); err != nil {
		if errors.Is(err, repository.ErrContentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "review not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AdminContentHandler) ListTags(c *gin.Context) {
	tags, err := h.svc.ListTags(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tags unavailable"})
		return
	}
	c.JSON(http.StatusOK, tags)
}

func (h *AdminContentHandler) CreateTag(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}

	id, err := h.svc.CreateTag(c.Request.Context(), req.Name)
	if err != nil {
		if errors.Is(err, service.ErrTagExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "tag exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *AdminContentHandler) UpdateTag(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tag id"})
		return
	}
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}

	if err := h.svc.UpdateTag(c.Request.Context(), id, req.Name); err != nil {
		if errors.Is(err, service.ErrTagExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "tag exists"})
			return
		}
		if errors.Is(err, repository.ErrContentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "tag not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AdminContentHandler) DeleteTag(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tag id"})
		return
	}

	if err := h.svc.DeleteTag(c.Request.Context(), id); err != nil {
		if errors.Is(err, repository.ErrTagInUse) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tag in use"})
			return
		}
		if errors.Is(err, repository.ErrContentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "tag not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AdminContentHandler) MergeTag(c *gin.Context) {
	var req struct {
		FromID int64 `json:"fromId" binding:"required"`
		ToID   int64 `json:"toId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.svc.MergeTag(c.Request.Context(), req.FromID, req.ToID); err != nil {
		switch {
		case errors.Is(err, service.ErrSameTag):
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot merge a tag into itself"})
		case errors.Is(err, repository.ErrContentNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "tag not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "merge failed"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

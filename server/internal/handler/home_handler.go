package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/cache"
	"github.com/Maluslock/comic/server/internal/service"
)

// HomeHandler handles HTTP requests for the home page.
type HomeHandler struct {
	svc   *service.HomeService
	cache *cache.RedisCache
}

// NewHomeHandler creates a new HomeHandler.
func NewHomeHandler(svc *service.HomeService, c *cache.RedisCache) *HomeHandler {
	return &HomeHandler{svc: svc, cache: c}
}

// GetHome handles GET /api/v1/home — returns aggregated home page data as JSON.
func (h *HomeHandler) GetHome(c *gin.Context) {
	cacheKey := "home"

	var resp service.HomeResponse
	if h.cache.Get(c.Request.Context(), cacheKey, &resp) {
		c.JSON(http.StatusOK, resp)
		return
	}

	data, err := h.svc.GetHomeData(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	h.cache.Set(c.Request.Context(), cacheKey, data, 5*time.Minute)
	c.JSON(http.StatusOK, data)
}

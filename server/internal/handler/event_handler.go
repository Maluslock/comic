package handler

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/cache"
	"github.com/Maluslock/comic/server/internal/service"
)

type EventHandler struct {
	svc   *service.EventService
	cache *cache.RedisCache
}

func NewEventHandler(svc *service.EventService, c *cache.RedisCache) *EventHandler {
	return &EventHandler{svc: svc, cache: c}
}

func (h *EventHandler) List(c *gin.Context) {
	location := c.Query("location")
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	cacheKey := "events:" + location + ":" + status + ":" + strconv.Itoa(page) + ":" + strconv.Itoa(size)

	var resp service.EventListResponse
	if h.cache.Get(c.Request.Context(), cacheKey, &resp) {
		c.JSON(http.StatusOK, resp)
		return
	}

	data, err := h.svc.List(c.Request.Context(), location, status, page, size)
	if err != nil {
		log.Printf("ERROR event.List: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	h.cache.Set(c.Request.Context(), cacheKey, data, 10*time.Minute)
	c.JSON(http.StatusOK, data)
}

func (h *EventHandler) Detail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
		return
	}

	data, err := h.svc.GetDetail(c.Request.Context(), id)
	if err != nil {
		log.Printf("ERROR event.Detail id=%d: %v", id, err)
		if isNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, data)
}

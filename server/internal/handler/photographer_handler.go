package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/cache"
	"github.com/Maluslock/comic/server/internal/middleware"
	"github.com/Maluslock/comic/server/internal/service"
)

type PhotographerHandler struct {
	svc        *service.PhotographerService
	cache      *cache.RedisCache
	bookingSvc *service.BookingService
}

func NewPhotographerHandler(svc *service.PhotographerService, c *cache.RedisCache, bookingSvc *service.BookingService) *PhotographerHandler {
	return &PhotographerHandler{svc: svc, cache: c, bookingSvc: bookingSvc}
}

func (h *PhotographerHandler) List(c *gin.Context) {
	keyword := c.Query("keyword")
	location := c.Query("location")
	tag := c.Query("tags")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	cacheKey := "photographers:" + keyword + ":" + location + ":" + tag + ":" + strconv.Itoa(page) + ":" + strconv.Itoa(size)

	var excludeUserIDs []int64
	if uid := middleware.UserID(c); uid > 0 {
		if ids, err := h.svc.HiddenUserIDs(c.Request.Context(), uid); err == nil {
			excludeUserIDs = ids
		}
	}

	if len(excludeUserIDs) > 0 {
		data, err := h.svc.List(c.Request.Context(), keyword, location, tag, page, size, excludeUserIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		c.JSON(http.StatusOK, data)
		return
	}

	var resp service.PhotographerListResponse
	if h.cache.Get(c.Request.Context(), cacheKey, &resp) {
		c.JSON(http.StatusOK, resp)
		return
	}

	data, err := h.svc.List(c.Request.Context(), keyword, location, tag, page, size, excludeUserIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	h.cache.Set(c.Request.Context(), cacheKey, data, 5*time.Minute)
	c.JSON(http.StatusOK, data)
}

func (h *PhotographerHandler) Detail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid photographer id"})
		return
	}

	data, err := h.svc.GetDetail(c.Request.Context(), int32(id))
	if err != nil {
		if isNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "photographer not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *PhotographerHandler) TimeSlots(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid photographer id"})
		return
	}
	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date is required"})
		return
	}
	times, err := h.bookingSvc.GetOccupiedTimes(c.Request.Context(), int32(id), date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"date": date, "occupied": times})
}

func (h *PhotographerHandler) Activate(c *gin.Context) {
	var req struct {
		Name  string `json:"name" binding:"required"`
		Mode  string `json:"mode" binding:"required"`
		Intro string `json:"intro"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	userID := middleware.UserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	pid, err := h.svc.Activate(c.Request.Context(), userID, req.Name, req.Mode, req.Intro)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"photographerId": pid})
}

func (h *PhotographerHandler) GetByUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	item, err := h.svc.GetByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not activated"})
		return
	}
	c.JSON(http.StatusOK, item)
}

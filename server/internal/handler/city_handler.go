package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/cache"
	"github.com/Maluslock/comic/server/internal/service"
)

// CityCacheKey 是城市清单的缓存键（整块响应，不分页）。
const CityCacheKey = "cities"

// CityHandler 暴露 GET /api/v1/cities。
//
// 城市清单是重聚合（comic_events 上 GROUP BY location），但数据一天只变两次
// （cron 早上 3 点、下午 15 点各跑一轮 events-sync），所以用 10 分钟 TTL 的整块缓存：
// 和 event_handler.go 的 `events:{location}:{status}:{page}:{size}` 同一套写法，
// 只是这里键是常量，不用拼参数。
type CityHandler struct {
	svc   *service.CityService
	cache *cache.RedisCache
}

func NewCityHandler(svc *service.CityService, c *cache.RedisCache) *CityHandler {
	return &CityHandler{svc: svc, cache: c}
}

func (h *CityHandler) List(c *gin.Context) {
	var resp service.CityListResponse
	if h.cache != nil && h.cache.Get(c.Request.Context(), CityCacheKey, &resp) {
		c.JSON(http.StatusOK, resp)
		return
	}

	data, err := h.svc.List(c.Request.Context())
	if err != nil {
		log.Printf("ERROR city.List: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	if h.cache != nil {
		h.cache.Set(c.Request.Context(), CityCacheKey, data, cityCacheTTL)
	}
	c.JSON(http.StatusOK, data)
}

package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/middleware"
	"github.com/Maluslock/comic/server/internal/service"
)

// photographerListCachePrefix 与 PhotographerHandler.List 写入的 key 前缀一致
// （"photographers:<keyword>:<location>:<tag>:<sort>:<page>:<size>"）。套餐价格是列表
// 卡片的一部分，所以任何套餐增删改都必须让列表缓存失效 —— 否则摄影师改完价，
// C 端最长 5 分钟仍显示旧价。
const photographerListCachePrefix = "photographers:"

// 首页与活动列表同样把「¥最低价 / ¥x 起 / ¥x/张 起」渲染进卡片（home_service.go 与
// event_service.go 都回填了 MinPrice/MinPricePerPhoto/ServiceCount）。只清列表缓存
// 会留下一个更隐蔽的不一致：列表已是新价、首页还是旧价，用户两次刷新看到两个数字。
// home 的缓存键是裸 "home"（handler/home_handler.go），故前缀不带冒号。
const (
	homeCachePrefix  = "home"
	eventCachePrefix = "events:"
)

// pricingCachePrefixes 是「所有会展示套餐价格」的缓存前缀集合。
var pricingCachePrefixes = []string{photographerListCachePrefix, homeCachePrefix, eventCachePrefix}

// invalidatePricingCaches 清掉全部渲染套餐价格的缓存。best-effort：缓存清理失败
// 绝不能让写请求失败（条目最多存活一个 TTL）。
// cache 为 nil（Redis 不可用时 NewRedisCache 返回 nil）必须安全空转。
func invalidatePricingCaches(ctx context.Context, cache listCacheInvalidator) {
	if cache == nil {
		return
	}
	for _, prefix := range pricingCachePrefixes {
		cache.DeletePrefix(ctx, prefix)
	}
}

// listCacheInvalidator 只依赖「按前缀清理」这一个能力，便于在测试里替换成 fake；
// *cache.RedisCache 为 nil 时方法本身是安全的空操作。
type listCacheInvalidator interface {
	DeletePrefix(ctx context.Context, prefix string) int
}

type PhotographerServiceHandler struct {
	svc   *service.PhotographerService
	cache listCacheInvalidator
}

func NewPhotographerServiceHandler(svc *service.PhotographerService, c listCacheInvalidator) *PhotographerServiceHandler {
	return &PhotographerServiceHandler{svc: svc, cache: c}
}

// invalidateListCache 清掉全部列表 / 首页 / 活动缓存分页与筛选组合。
func (h *PhotographerServiceHandler) invalidateListCache(ctx context.Context) {
	invalidatePricingCaches(ctx, h.cache)
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
	case errors.Is(err, service.ErrPerPhotoPriceRequired):
		c.JSON(http.StatusBadRequest, gin.H{"error": "按张计价的单价必须大于 0（免费请用整套价 0 元）"})
	case errors.Is(err, service.ErrInvalidPriceUnit):
		c.JSON(http.StatusBadRequest, gin.H{"error": "priceUnit must be one of fixed/per_photo/per_hour/per_day"})
	case errors.Is(err, service.ErrInvalidPhotoCount):
		c.JSON(http.StatusBadRequest, gin.H{"error": "photoCount must be > 0"})
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
	h.invalidateListCache(c.Request.Context())
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
	h.invalidateListCache(c.Request.Context())
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
	h.invalidateListCache(c.Request.Context())
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

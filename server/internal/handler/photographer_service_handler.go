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
// （"photographers:<keyword>:<location>:<tag>:<page>:<size>"）。套餐价格是列表卡片
// 的一部分，所以任何套餐增删改都必须让列表缓存失效 —— 否则摄影师改完价，
// C 端最长 5 分钟仍显示旧价。
const photographerListCachePrefix = "photographers:"

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

// invalidateListCache 清掉全部列表缓存分页/筛选组合。best-effort：失败不影响写入结果。
func (h *PhotographerServiceHandler) invalidateListCache(ctx context.Context) {
	if h.cache == nil {
		return
	}
	h.cache.DeletePrefix(ctx, photographerListCachePrefix)
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

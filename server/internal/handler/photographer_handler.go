package handler

import (
	"context"
	"errors"
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
	// invalidator 与 cache 指向同一个 Redis 缓存，只是收窄成「按前缀清理」这一个
	// 能力：缓存置灰（NewRedisCache 在 Redis 不可用时返回 nil）时保持为 nil，
	// 清理调用随之空转，而不是在 handler 里散落 nil 判断。
	invalidator listCacheInvalidator
}

func NewPhotographerHandler(svc *service.PhotographerService, c *cache.RedisCache, bookingSvc *service.BookingService) *PhotographerHandler {
	h := &PhotographerHandler{svc: svc, cache: c, bookingSvc: bookingSvc}
	if c != nil {
		h.invalidator = c
	}
	return h
}

// invalidatePricingCaches 清掉所有渲染套餐价格的缓存（列表 / 首页 / 活动）。
// 摄影师改名、开通、改套餐都会改变卡片上展示的东西，不清就会出现「列表已是新名字、
// 首页还是旧名字」这种更难排查的半新半旧状态。best-effort：清理失败不影响写结果。
func (h *PhotographerHandler) invalidatePricingCaches(ctx context.Context) {
	invalidatePricingCaches(ctx, h.invalidator)
}

func (h *PhotographerHandler) List(c *gin.Context) {
	keyword := c.Query("keyword")
	location := c.Query("location")
	tag := c.Query("tags")
	sort := c.Query("sort")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	// sort 必须进缓存键：否则切「评分最高」会命中「接单最多」的缓存，越权返回错误内容。
	cacheKey := "photographers:" + keyword + ":" + location + ":" + tag + ":" + sort + ":" + strconv.Itoa(page) + ":" + strconv.Itoa(size)

	var excludeUserIDs []int64
	viewerUserID := middleware.UserID(c)
	if viewerUserID > 0 {
		if ids, err := h.svc.HiddenUserIDs(c.Request.Context(), viewerUserID); err == nil {
			excludeUserIDs = ids
		}
	}

	if len(excludeUserIDs) > 0 {
		data, err := h.svc.List(c.Request.Context(), keyword, location, tag, page, size, excludeUserIDs, sort)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		data.List = service.MarkSelf(data.List, viewerUserID)
		c.JSON(http.StatusOK, data)
		return
	}

	var resp service.PhotographerListResponse
	if h.cache.Get(c.Request.Context(), cacheKey, &resp) {
		// 缓存里存的是 IsSelf 全 false 的原始页（缓存键不含 viewer，否则会按登录
		// 用户切成 N 份、还会让匿名与登录分裂成两套 key），命中后按当前 token 现算。
		resp.List = service.MarkSelf(resp.List, viewerUserID)
		c.JSON(http.StatusOK, resp)
		return
	}

	data, err := h.svc.List(c.Request.Context(), keyword, location, tag, page, size, excludeUserIDs, sort)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	h.cache.Set(c.Request.Context(), cacheKey, data, 5*time.Minute)
	data.List = service.MarkSelf(data.List, viewerUserID)
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
		// 此前这里一律 500，把「mode 填错」这种纯客户端错误也报成服务端故障，
		// 前端既拿不到可用提示，日志里也失去区分度。按错误语义分流：
		//   - ErrInvalidMode：请求体里的 mode 不在 free/pay/both 之内 → 400。
		//   - ErrProfileNotFound：档案在写入过程中消失（并发删号）→ 404。
		//   - 其余（DB / 网络）才是 500。
		switch {
		case errors.Is(err, service.ErrInvalidMode):
			c.JSON(http.StatusBadRequest, gin.H{"error": "mode must be one of free/pay/both"})
			return
		case errors.Is(err, service.ErrProfileNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "photographer profile not found"})
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
	}
	// 开通 / 二次 activate 可能改写展示名与 mode，而两者都渲染在列表、首页、活动
	// 卡片上：不清缓存就会出现「刚开通完，列表里还是旧名字」。
	h.invalidatePricingCaches(c.Request.Context())
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

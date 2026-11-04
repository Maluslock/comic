package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/middleware"
	"github.com/Maluslock/comic/server/internal/service"
)

func (h *PhotographerHandler) UpdateProfile(c *gin.Context) {
	userID := middleware.UserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req service.ProfileUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.svc.UpdateProfile(c.Request.Context(), userID, req); err != nil {
		switch {
		case errors.Is(err, service.ErrNotPhotographer):
			c.JSON(http.StatusForbidden, gin.H{"error": "not a photographer"})
		case errors.Is(err, service.ErrInvalidMode):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mode"})
		case errors.Is(err, service.ErrProfileNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "photographer not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "update profile failed"})
		}
		return
	}
	// 展示名 / 城市 / 简介 / mode 都渲染在列表与首页卡片上：改完必须立刻失效缓存，
	// 否则摄影师自己刷新列表仍看到旧信息，会以为没保存成功而重复提交。
	h.invalidatePricingCaches(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *PhotographerHandler) MyProfile(c *gin.Context) {
	userID := middleware.UserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	item, err := h.svc.MyProfile(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrNotPhotographer) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not a photographer"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, item)
}

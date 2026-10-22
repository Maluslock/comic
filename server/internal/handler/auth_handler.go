package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// WeChatLogin 处理小程序 wx.login 的 code 登录。
// 未配置 AppID 时服务端使用 mock 客户端（只接受 mock_ 前缀 code），方便本地联调。
func (h *AuthHandler) WeChatLogin(c *gin.Context) {
	var req service.WeChatLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	data, err := h.svc.WeChatLogin(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAccountDisabled):
			c.JSON(http.StatusForbidden, gin.H{"error": "account disabled"})
		case errors.Is(err, service.ErrWeChatDisabled):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "wechat login is not configured"})
		case errors.Is(err, service.ErrWeChatCodeBad):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid wechat code"})
		case errors.Is(err, service.ErrWeChatConfig):
			c.JSON(http.StatusInternalServerError, gin.H{"error": "wechat credentials are not configured correctly"})
		default:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "wechat login failed"})
		}
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	data, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrAccountDisabled) {
			c.JSON(http.StatusForbidden, gin.H{"error": "account disabled"})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	c.JSON(http.StatusOK, data)
}

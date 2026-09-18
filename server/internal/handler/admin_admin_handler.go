package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/Maluslock/comic/server/internal/service"
)

type AdminAdminHandler struct {
	svc *service.AdminService
}

func NewAdminAdminHandler(svc *service.AdminService) *AdminAdminHandler {
	return &AdminAdminHandler{svc: svc}
}

func (h *AdminAdminHandler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "admins unavailable"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *AdminAdminHandler) Create(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and password required"})
		return
	}

	id, err := h.svc.Create(c.Request.Context(), req.Username, req.Password, req.Role)
	if err != nil {
		if errors.Is(err, service.ErrUsernameExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "username exists"})
			return
		}
		if errors.Is(err, service.ErrInvalidRole) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *AdminAdminHandler) SetStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid admin id"})
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status required"})
		return
	}

	if err := h.svc.SetStatus(c.Request.Context(), id, req.Status); err != nil {
		if errors.Is(err, service.ErrInvalidStatus) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
			return
		}
		if errors.Is(err, service.ErrPrimaryAdmin) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot disable primary admin"})
			return
		}
		if errors.Is(err, repository.ErrAdminNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "admin not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AdminAdminHandler) ResetPassword(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid admin id"})
		return
	}
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password required"})
		return
	}

	if err := h.svc.ResetPassword(c.Request.Context(), id, req.Password); err != nil {
		if errors.Is(err, repository.ErrAdminNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "admin not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "password reset failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

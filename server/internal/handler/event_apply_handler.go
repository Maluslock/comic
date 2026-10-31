package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/middleware"
	"github.com/Maluslock/comic/server/internal/service"
)

// EventApplyHandler 是「摄影师申请入驻某漫展」的 C 端入口。
// 身份一律取 token（middleware.UserID），路径与 body 里的用户 id 不参与授权判断。
type EventApplyHandler struct {
	svc *service.EventApplyService
}

func NewEventApplyHandler(svc *service.EventApplyService) *EventApplyHandler {
	return &EventApplyHandler{svc: svc}
}

type applyEventPhotographerRequest struct {
	Message string `json:"message"`
}

// parseEventIDParam 统一解析 :id，失败写 400 并返回 false。
func parseEventIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
		return 0, false
	}
	return id, true
}

// writeEventApplyError 把 service 错误映射成 HTTP 状态码。
// 23505 → 409、不存在 → 404、非摄影师 → 403，绝不统一 500。
func writeEventApplyError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEventApplyNotPhotographer):
		c.JSON(http.StatusForbidden, gin.H{"error": "only photographers can apply"})
	case errors.Is(err, service.ErrEventNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
	case errors.Is(err, service.ErrEventEnded):
		c.JSON(http.StatusConflict, gin.H{"error": "event already ended"})
	case errors.Is(err, service.ErrEventApplyDuplicate):
		c.JSON(http.StatusConflict, gin.H{"error": "already applied for this event"})
	case errors.Is(err, service.ErrEventApplyNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
	case errors.Is(err, service.ErrEventApplyInvalidStatus):
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be approved or rejected"})
	case errors.Is(err, service.ErrEventApplyReasonRequired):
		c.JSON(http.StatusBadRequest, gin.H{"error": "reason is required when rejecting"})
	case errors.Is(err, service.ErrEventApplyAlreadyReviewed):
		c.JSON(http.StatusConflict, gin.H{"error": "application already reviewed"})
	case isNotFound(err):
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
	default:
		log.Printf("ERROR event apply: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

// Apply 提交入驻申请：POST /api/v1/events/:id/photographer-apply
func (h *EventApplyHandler) Apply(c *gin.Context) {
	eventID, ok := parseEventIDParam(c)
	if !ok {
		return
	}
	var req applyEventPhotographerRequest
	// body 可选：什么都不传也要能申请
	if c.Request != nil && c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
	}

	app, err := h.svc.Apply(c.Request.Context(), middleware.UserID(c), eventID, req.Message)
	if err != nil {
		writeEventApplyError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":        app.ID,
		"eventId":   app.EventID,
		"status":    app.Status,
		"appliedAt": app.AppliedAt,
	})
}

// MyApplication 查询自己在某漫展的申请：GET /api/v1/events/:id/photographer-apply/mine
func (h *EventApplyHandler) MyApplication(c *gin.Context) {
	eventID, ok := parseEventIDParam(c)
	if !ok {
		return
	}
	app, err := h.svc.MyApplication(c.Request.Context(), middleware.UserID(c), eventID)
	if err != nil {
		if isNotFound(err) {
			// 从未申请不是错误：返回 application: null，前端据此显示「申请入驻」按钮
			c.JSON(http.StatusOK, gin.H{"application": nil})
			return
		}
		writeEventApplyError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"application": gin.H{
		"id":           app.ID,
		"eventId":      app.EventID,
		"status":       app.Status,
		"message":      app.Message,
		"reviewReason": app.ReviewReason,
		"appliedAt":    app.AppliedAt,
		"reviewedAt":   app.ReviewedAt,
	}})
}

// Withdraw 撤回申请：DELETE /api/v1/events/:id/photographer-apply
func (h *EventApplyHandler) Withdraw(c *gin.Context) {
	eventID, ok := parseEventIDParam(c)
	if !ok {
		return
	}
	if err := h.svc.Withdraw(c.Request.Context(), middleware.UserID(c), eventID); err != nil {
		writeEventApplyError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// MyApplications 我申请过的漫展：GET /api/v1/photographer/event-applications
func (h *EventApplyHandler) MyApplications(c *gin.Context) {
	list, err := h.svc.MyApplications(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		writeEventApplyError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": list})
}

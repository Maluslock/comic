package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/service"
)

// AdminEventSourceHandler 提供 admin「数据源状态」页的只读接口。
// 页面需要的一切（每源卡片 + 全局运行历史 + 各源在库总数）一次 GET 拿全，
// 避免前端串行发三四个请求再自己拼。
type AdminEventSourceHandler struct {
	svc *service.EventSourceService
}

func NewAdminEventSourceHandler(svc *service.EventSourceService) *AdminEventSourceHandler {
	return &AdminEventSourceHandler{svc: svc}
}

// EventSources 处理 GET /api/admin/v1/event-sources。
func (h *AdminEventSourceHandler) EventSources(c *gin.Context) {
	overview, err := h.svc.Overview(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "event sources unavailable"})
		return
	}
	c.JSON(http.StatusOK, overview)
}

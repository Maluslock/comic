package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Maluslock/comic/server/internal/service"
)

type AdminExportHandler struct {
	manage *service.AdminManageService
	users  *service.AdminUserService
}

func NewAdminExportHandler(manage *service.AdminManageService, users *service.AdminUserService) *AdminExportHandler {
	return &AdminExportHandler{manage: manage, users: users}
}

func writeCSV(c *gin.Context, filename string, header []string, rows [][]string) {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	w := csv.NewWriter(c.Writer)
	defer w.Flush()

	if err := w.Write(header); err != nil {
		return
	}
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return
		}
	}
}

func (h *AdminExportHandler) Orders(c *gin.Context) {
	list, err := h.manage.ExportOrders(c.Request.Context(), c.Query("status"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "export failed"})
		return
	}

	rows := make([][]string, 0, len(list))
	for _, o := range list {
		rows = append(rows, []string{
			strconv.FormatInt(o.ID, 10),
			o.Status,
			o.Date,
			o.Time,
			strconv.FormatInt(int64(o.Price), 10),
			o.CreatedAt,
			o.CoserName,
			o.PhotographerName,
		})
	}
	writeCSV(c, "orders.csv", []string{"ID", "状态", "日期", "时间", "金额", "创建时间", "Coser", "摄影师"}, rows)
}

func (h *AdminExportHandler) Users(c *gin.Context) {
	list, err := h.users.ExportUsers(c.Request.Context(), c.Query("keyword"), c.Query("role"), c.Query("status"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "export failed"})
		return
	}

	rows := make([][]string, 0, len(list))
	for _, u := range list {
		rows = append(rows, []string{
			strconv.FormatInt(u.ID, 10),
			u.Phone,
			u.Name,
			u.Role,
			u.Status,
			u.CreatedAt,
		})
	}
	writeCSV(c, "users.csv", []string{"ID", "手机号", "昵称", "角色", "状态", "注册时间"}, rows)
}

func (h *AdminExportHandler) Photographers(c *gin.Context) {
	var certified *bool
	if v, ok := c.GetQuery("certified"); ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid certified value"})
			return
		}
		certified = &b
	}

	list, err := h.manage.ExportPhotographers(c.Request.Context(), certified)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "export failed"})
		return
	}

	rows := make([][]string, 0, len(list))
	for _, p := range list {
		rows = append(rows, []string{
			strconv.FormatInt(p.ID, 10),
			p.Name,
			p.Location,
			strconv.FormatFloat(p.Rating, 'f', 1, 64),
			p.Mode,
			strconv.FormatInt(p.OrderCount, 10),
			strconv.FormatBool(p.Certified),
			p.UserPhone,
		})
	}
	writeCSV(c, "photographers.csv", []string{"ID", "昵称", "城市", "评分", "接单模式", "接单数", "已认证", "手机号"}, rows)
}

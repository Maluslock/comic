package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const maxUploadBytes = 5 << 20

var allowedImageExt = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
}

type UploadHandler struct {
	dir string
}

func NewUploadHandler(dir string) *UploadHandler {
	if dir == "" {
		dir = "./static/uploads"
	}

	return &UploadHandler{dir: dir}
}

func (h *UploadHandler) Upload(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(maxUploadBytes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid multipart form"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	if file.Size > maxUploadBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file too large (max 5MB)"})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedImageExt[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported image type"})
		return
	}

	if err := os.MkdirAll(h.dir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create upload dir"})
		return
	}

	suffix := make([]byte, 8)
	rand.Read(suffix)
	name := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), hex.EncodeToString(suffix), ext)

	if err := c.SaveUploadedFile(file, filepath.Join(h.dir, name)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot save file"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"url": "/static/uploads/" + name})
}

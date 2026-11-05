package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const maxUploadBytes = 5 << 20

// sniffLen 是识别图片格式需要读的字节数。所有主流图片格式的魔数都在前 16 字节内，
// 取 512 与 http.DetectContentType 的内部缓冲一致，顺带覆盖了它的判定范围。
const sniffLen = 512

// allowedImageExt 只用于「注册表校验」：确认我们把 sniff 出来的类型映射到了白名单扩展名。
// 注意它**不再**用来判断上传文件本身合法与否 —— 见 sniffImageExt 的注释。
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

// sniffImageExt 按**文件内容**判定图片类型并返回落盘用的扩展名。
//
// 为什么不能用 filepath.Ext(file.Filename)：
// 小程序 image 组件和 wx.chooseAvatar / chooseImage 给出的临时路径是
// `wxfile://tmp_xxx` 这类**没有扩展名**的形式（H5 端 blob: URL 同理），
// 微信开发者工具与真机的差别还会让同一份图在不同端表现不同。
// 结果就是：接口返回 400 unsupported image type，前端只看到「上传失败」，
// 而抓包看 HTTP 状态与请求体都正常 —— 正是「接口调用正常但上传失败」的成因。
//
// 这里改用魔数嗅探（顺序敏感：webp 必须排在 RIFF 之前判，否则 RIFF 会先命中）。
func sniffImageExt(head []byte) (string, bool) {
	if len(head) >= 12 && bytes.Equal(head[0:4], []byte("RIFF")) && bytes.Equal(head[8:12], []byte("WEBP")) {
		return ".webp", true
	}
	switch {
	case len(head) >= 3 && head[0] == 0xFF && head[1] == 0xD8 && head[2] == 0xFF:
		// JPEG：FF D8 FF（SOI + 第一个 marker）。.jpg 是通行写法。
		return ".jpg", true
	case len(head) >= 8 && bytes.Equal(head[0:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}):
		return ".png", true
	case len(head) >= 6 && (bytes.Equal(head[0:6], []byte("GIF87a")) || bytes.Equal(head[0:6], []byte("GIF89a"))):
		return ".gif", true
	}
	return "", false
}

// saveImage 把已解出的图片字节按魔数定扩展名落盘，返回可对外访问的相对路径。
// multipart 与 base64 两条上传通道共用，保证两种入口的校验规则完全一致。
func (h *UploadHandler) saveImage(data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("file is empty")
	}
	if len(data) > maxUploadBytes {
		return "", fmt.Errorf("file too large (max 5MB)")
	}

	head := data
	if len(head) > sniffLen {
		head = head[:sniffLen]
	}
	ext, ok := sniffImageExt(head)
	if !ok || !allowedImageExt[ext] {
		return "", fmt.Errorf("unsupported image type (支持 jpg/png/gif/webp)")
	}

	if err := os.MkdirAll(h.dir, 0o755); err != nil {
		return "", fmt.Errorf("cannot create upload dir")
	}

	suffix := make([]byte, 8)
	rand.Read(suffix)
	name := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), hex.EncodeToString(suffix), ext)
	dstPath := filepath.Join(h.dir, name)

	dst, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("cannot save file")
	}
	if _, err := dst.Write(data); err != nil {
		dst.Close()
		os.Remove(dstPath)
		return "", fmt.Errorf("cannot save file")
	}
	if err := dst.Close(); err != nil {
		os.Remove(dstPath)
		return "", fmt.Errorf("cannot save file")
	}
	return "/static/uploads/" + name, nil
}

// UploadBase64 是**绕过 uploadFile 白名单**的上传通道。
//
// 为什么需要它：小程序的 uni.uploadFile 走独立的「uploadFile 合法域名」白名单，
// 与 uni.request 的 request 白名单是两套配置。现场症状是登录（request）一直正常、
// 头像（uploadFile）永远失败，且**服务端零日志**——请求在客户端就被拦了。
// 这条路走普通 request（已验证可用），把图片 base64 塞进 JSON body 发上来。
//
// body: {"data":"data:image/jpeg;base64,...."} 或 {"data":"<裸 base64>"}
func (h *UploadHandler) UploadBase64(c *gin.Context) {
	var req struct {
		Data string `json:"data" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "data is required"})
		return
	}

	payload := req.Data
	// 允许标准 DataURL 前缀，剥掉后再解码
	if idx := strings.Index(payload, "base64,"); idx >= 0 {
		payload = payload[idx+len("base64,"):]
	}
	// base64 里允许换行（部分端会按行折行），统一去掉
	payload = strings.ReplaceAll(payload, "\n", "")
	payload = strings.ReplaceAll(payload, "\r", "")

	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid base64 payload"})
		return
	}

	url, err := h.saveImage(data)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "cannot save file" || err.Error() == "cannot create upload dir" {
			status = http.StatusInternalServerError
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"url": url})
}

func (h *UploadHandler) Upload(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(maxUploadBytes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid multipart form"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	if fileHeader.Size > maxUploadBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file too large (max 5MB)"})
		return
	}
	if fileHeader.Size == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is empty"})
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot read file"})
		return
	}
	defer src.Close()

	// 整体读进内存后交给 saveImage：与 base64 通道共用同一套校验与落盘逻辑，
	// 上限 5MB，不存在把内存吃爆的风险。
	data, err := io.ReadAll(io.LimitReader(src, maxUploadBytes+1))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot read file"})
		return
	}

	url, err := h.saveImage(data)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "cannot save file" || err.Error() == "cannot create upload dir" {
			status = http.StatusInternalServerError
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"url": url})
}

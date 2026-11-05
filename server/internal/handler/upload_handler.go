package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
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

// readHead 读取文件前 sniffLen 字节，并把读走的部分拼回去，保证后续 SaveUploadedFile 写出完整内容。
func readHead(f multipart.File) ([]byte, io.Reader, error) {
	head := make([]byte, sniffLen)
	n, err := io.ReadFull(f, head)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, nil, err
	}
	head = head[:n]
	return head, io.MultiReader(bytes.NewReader(head), f), nil
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

	head, full, err := readHead(src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot read file"})
		return
	}

	ext, ok := sniffImageExt(head)
	if !ok || !allowedImageExt[ext] {
		// 文案保留原来的机器可读前缀（前端按 error 字段直接展示），
		// 但补一句人话，避免用户只知道"失败"却不知道要换张图。
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported image type (支持 jpg/png/gif/webp)"})
		return
	}

	if err := os.MkdirAll(h.dir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create upload dir"})
		return
	}

	suffix := make([]byte, 8)
	rand.Read(suffix)
	name := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), hex.EncodeToString(suffix), ext)
	dstPath := filepath.Join(h.dir, name)

	dst, err := os.Create(dstPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot save file"})
		return
	}
	if _, err := io.Copy(dst, full); err != nil {
		dst.Close()
		os.Remove(dstPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot save file"})
		return
	}
	if err := dst.Close(); err != nil {
		os.Remove(dstPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot save file"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"url": "/static/uploads/" + name})
}

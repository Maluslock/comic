package middleware

import (
	"compress/gzip"
	"io"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// gzipExtensions is the allowlist of textual assets worth compressing. Images
// (jpg/png/webp) and fonts are already compressed; gzipping them would only burn
// CPU and slightly grow the payload.
var gzipExtensions = map[string]bool{
	".js": true, ".mjs": true, ".css": true, ".html": true, ".json": true,
	".svg": true, ".txt": true, ".map": true, ".ico": false,
}

func isCompressibleContentType(ct string) bool {
	ct = strings.ToLower(strings.TrimSpace(ct))
	if ct == "" {
		return false
	}
	if idx := strings.Index(ct, ";"); idx >= 0 {
		ct = strings.TrimSpace(ct[:idx])
	}
	if ct == "application/json" || ct == "application/javascript" {
		return true
	}
	if strings.HasPrefix(ct, "text/") {
		return true
	}
	return false
}

// Gzip compresses textual responses when the client advertises support.
func Gzip() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		ext := strings.ToLower(filepath.Ext(c.Request.URL.Path))
		byExt := gzipExtensions[ext]

		w := &gzipResponseWriter{
			ResponseWriter: c.Writer,
			byExt:          byExt,
		}
		c.Writer = w

		defer func() {
			w.finish()
		}()

		c.Next()
	}
}

type gzipResponseWriter struct {
	gin.ResponseWriter
	byExt       bool
	gz          *gzip.Writer
	initialized bool
	shouldGzip  bool
	headerCode  int
}

func (w *gzipResponseWriter) initGzipIfNeeded() {
	if w.initialized {
		return
	}
	w.initialized = true

	ct := w.Header().Get("Content-Type")
	if w.byExt || isCompressibleContentType(ct) {
		w.shouldGzip = true
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
		w.Header().Del("Content-Length")
		w.gz = gzip.NewWriter(w.ResponseWriter)
	}

	if w.headerCode > 0 {
		w.ResponseWriter.WriteHeader(w.headerCode)
	}
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	// 与 gin 的 responseWriter 语义保持一致：在真正写出字节之前，**最后**一次
	// WriteHeader 生效，而不是最先一次。
	//
	// 这对静态文件是必需的：gin 的 createStaticHandler 会先调
	// c.Writer.WriteHeader(http.StatusNotFound) 做试探（Dir(root,false) 返回的
	// onlyFilesFS 让它必然走到这一行），随后 http.FileServer.ServeHTTP 内部又调
	// WriteHeader(200)。gin 原生 writer 因为还没有 Write 过，允许后者覆盖前者，
	// 所以无 gzip 时响应是 200；若这里改成「首个 code 锁死」就会把试探性的 404
	// 固化下来，出现「状态行 404 + 完整文件内容」的坏响应。
	if code > 0 {
		w.headerCode = code
	}
	if w.initialized {
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *gzipResponseWriter) WriteHeaderNow() {
	// Delay until first Write or finish to allow Content-Type sniffing/headers
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	w.initGzipIfNeeded()
	if w.shouldGzip && w.gz != nil {
		w.Header().Del("Content-Length")
		return w.gz.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

func (w *gzipResponseWriter) WriteString(s string) (int, error) {
	w.initGzipIfNeeded()
	if w.shouldGzip && w.gz != nil {
		w.Header().Del("Content-Length")
		return io.WriteString(w.gz, s)
	}
	return w.ResponseWriter.WriteString(s)
}

func (w *gzipResponseWriter) finish() {
	w.initGzipIfNeeded()
	if w.gz != nil {
		_ = w.gz.Close()
	}
}

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

// Gzip compresses textual responses when the client advertises support. It exists
// because the admin bundle is ~5MB and the deployment sits behind a slow tunnel:
// without compression the login page takes tens of seconds to become interactive.
func Gzip() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}
		if !gzipExtensions[strings.ToLower(filepath.Ext(c.Request.URL.Path))] {
			c.Next()
			return
		}

		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")
		gz := gzip.NewWriter(c.Writer)
		defer gz.Close()

		c.Writer = &gzipResponseWriter{ResponseWriter: c.Writer, gz: gz}
		c.Next()
	}
}

type gzipResponseWriter struct {
	gin.ResponseWriter
	gz *gzip.Writer
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	// Static serving sets Content-Length from the uncompressed file; keeping it
	// would make the browser wait for bytes that never arrive.
	w.Header().Del("Content-Length")
	return w.gz.Write(b)
}

func (w *gzipResponseWriter) WriteString(s string) (int, error) {
	w.Header().Del("Content-Length")
	return io.WriteString(w.gz, s)
}

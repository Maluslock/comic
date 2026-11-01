package middleware

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// StaticCache returns a gin.HandlerFunc for /static route that:
// 1. Sets Cache-Control:
//   - /static/remote/ and /static/img/ -> public, max-age=604800 (7 days)
//   - /static/uploads/ -> no-cache, must-revalidate
//
// 2. Computes ETag based on file size + modtime
// 3. Handles If-None-Match: returns 304 Not Modified with empty body if matched.
func StaticCache(staticDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := c.Request.URL.Path
		if !strings.HasPrefix(p, "/static/") {
			c.Next()
			return
		}

		// Set Cache-Control according to subpath
		switch {
		case strings.HasPrefix(p, "/static/remote/") || strings.HasPrefix(p, "/static/img/"):
			c.Header("Cache-Control", "public, max-age=604800")
		case strings.HasPrefix(p, "/static/uploads/"):
			c.Header("Cache-Control", "no-cache, must-revalidate")
		}

		relPath := strings.TrimPrefix(p, "/static")
		relPath = filepath.Clean(relPath)
		filePath := filepath.Join(staticDir, relPath)

		st, err := os.Stat(filePath)
		if err != nil || st.IsDir() {
			c.Next()
			return
		}

		// ETag based on file size and modtime unix nano
		etag := fmt.Sprintf(`"%x-%x"`, st.Size(), st.ModTime().UnixNano())
		c.Header("ETag", etag)

		// Check If-None-Match
		if ifNoneMatch := c.GetHeader("If-None-Match"); ifNoneMatch != "" {
			// If client sends If-None-Match matching our etag (or weak comparison / contains)
			for _, match := range strings.Split(ifNoneMatch, ",") {
				match = strings.TrimSpace(match)
				if match == etag || match == "W/"+etag || (strings.HasPrefix(etag, "W/") && match == strings.TrimPrefix(etag, "W/")) || match == "*" {
					c.Status(http.StatusNotModified)
					c.Writer.WriteHeaderNow()
					c.Abort()
					return
				}
			}
		}

		c.Next()
	}
}

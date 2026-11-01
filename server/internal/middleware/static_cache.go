package middleware

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// blockedStaticPrefixes 是 /static 下不允许对外暴露的私有目录（前缀匹配）。
//
// 目前只有 _orig_covers：optimize-covers.py 把 365 张原图（265MB）归档在这里做回滚备份，
// 它们既不被 C 端引用，也没有任何对外用途，但 router.Static 会无差别地把 ./static 整棵
// 目录树挂出去（实测 GET /static/remote/_orig_covers/1006166.jpg → 200 254617B）。
// 备份数据被公开下载既浪费带宽又扩大了暴露面，所以在这里统一 404。
var blockedStaticPrefixes = []string{
	"/static/remote/_orig_covers/",
	"/static/remote/_orig_covers",
}

// isBlockedStaticPath 判断请求路径是否落在私有静态目录内。
func isBlockedStaticPath(p string) bool {
	for _, prefix := range blockedStaticPrefixes {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

// StaticCache returns a gin.HandlerFunc for /static route that:
// 1. Sets Cache-Control:
//   - /static/remote/ and /static/img/ -> public, max-age=604800 (7 days)
//   - /static/uploads/ -> no-cache, must-revalidate
//
// 2. Computes ETag based on file size + modtime
// 3. Handles If-None-Match: returns 304 Not Modified with empty body if matched.
// 4. Refuses private archives under /static（见 blockedStaticPrefixes）。
func StaticCache(staticDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := c.Request.URL.Path
		if !strings.HasPrefix(p, "/static/") {
			c.Next()
			return
		}

		if isBlockedStaticPath(p) {
			c.Status(http.StatusNotFound)
			c.Writer.WriteHeaderNow()
			c.Abort()
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

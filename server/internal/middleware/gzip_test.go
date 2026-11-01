package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestGzipMiddleware(t *testing.T) {
	r := gin.New()
	r.Use(Gzip())

	r.GET("/api/v1/home", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "hello world this is a test response for gzip compression with reasonable length to verify compression works as expected",
		})
	})

	r.GET("/static/test.js", func(c *gin.Context) {
		c.String(http.StatusOK, "console.log('hello world js asset for testing gzip compression');")
	})

	r.GET("/image.png", func(c *gin.Context) {
		c.Data(http.StatusOK, "image/png", []byte("fake png binary data"))
	})

	t.Run("with Accept-Encoding gzip on JSON endpoint", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/home", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if ce := w.Header().Get("Content-Encoding"); ce != "gzip" {
			t.Errorf("expected Content-Encoding: gzip, got %q", ce)
		}
		if vary := w.Header().Get("Vary"); vary != "Accept-Encoding" {
			t.Errorf("expected Vary: Accept-Encoding, got %q", vary)
		}
		if cl := w.Header().Get("Content-Length"); cl != "" {
			t.Errorf("expected Content-Length to be removed, got %q", cl)
		}

		// Verify body can be decompressed
		gr, err := gzip.NewReader(w.Body)
		if err != nil {
			t.Fatalf("failed to decompress gzip body: %v", err)
		}
		defer gr.Close()
		decompressed, err := io.ReadAll(gr)
		if err != nil {
			t.Fatalf("failed to read decompressed body: %v", err)
		}
		if !bytes.Contains(decompressed, []byte("hello world this is a test response")) {
			t.Errorf("unexpected decompressed content: %s", string(decompressed))
		}
	})

	t.Run("without Accept-Encoding on JSON endpoint", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/home", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if ce := w.Header().Get("Content-Encoding"); ce != "" {
			t.Errorf("expected no Content-Encoding, got %q", ce)
		}
		if !bytes.Contains(w.Body.Bytes(), []byte("hello world this is a test response")) {
			t.Errorf("unexpected body content: %s", w.Body.String())
		}
	})

	t.Run("with Accept-Encoding gzip on .js extension", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/static/test.js", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if ce := w.Header().Get("Content-Encoding"); ce != "gzip" {
			t.Errorf("expected Content-Encoding: gzip, got %q", ce)
		}
	})

	t.Run("with Accept-Encoding gzip on .png (should not gzip)", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/image.png", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if ce := w.Header().Get("Content-Encoding"); ce != "" {
			t.Errorf("expected no Content-Encoding for png, got %q", ce)
		}
	})
}

// TestGzipPreservesStaticFileStatus 是回归测试：gin 的 createStaticHandler 会用
// WriteHeader(404) 做存在性试探，随后 http.FileServer 再 WriteHeader(200)。
// gzipResponseWriter 必须像 gin 原生 writer 一样让**最后一次** WriteHeader 生效，
// 否则静态文件会出现「状态行 404 + 完整文件内容」，浏览器与 wx.request 都会当成失败。
func TestGzipPreservesStaticFileStatus(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.webp")
	if err := os.WriteFile(file, []byte("RIFF----WEBPVP8 fake webp payload"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	newRouter := func() *gin.Engine {
		r := gin.New()
		r.Use(Gzip())
		r.Static("/static", dir)
		return r
	}
	r := newRouter()

	cases := []struct {
		name string
		ae   string
	}{
		{"no Accept-Encoding", ""},
		{"Accept-Encoding: gzip", "gzip"},
		{"Accept-Encoding: gzip, deflate, br", "gzip, deflate, br"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/static/a.webp", nil)
			if tc.ae != "" {
				req.Header.Set("Accept-Encoding", tc.ae)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200 for existing static file, got %d", w.Code)
			}
			if w.Body.Len() == 0 {
				t.Fatalf("expected non-empty body")
			}
			// 二进制资源不应被 gzip 包装
			if ce := w.Header().Get("Content-Encoding"); ce != "" {
				t.Errorf("expected no Content-Encoding for .webp, got %q", ce)
			}
		})
	}

	t.Run("missing static file still 404", func(t *testing.T) {
		for _, ae := range []string{"", "gzip"} {
			req, _ := http.NewRequest(http.MethodGet, "/static/nope.webp", nil)
			if ae != "" {
				req.Header.Set("Accept-Encoding", ae)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusNotFound {
				t.Fatalf("ae=%q expected 404 for missing file, got %d", ae, w.Code)
			}
		}
	})
}

package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestStaticCacheMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	remoteDir := filepath.Join(tmpDir, "remote", "covers")
	imgDir := filepath.Join(tmpDir, "img")
	uploadsDir := filepath.Join(tmpDir, "uploads")
	_ = os.MkdirAll(remoteDir, 0o755)
	_ = os.MkdirAll(imgDir, 0o755)
	_ = os.MkdirAll(uploadsDir, 0o755)

	testCover := filepath.Join(remoteDir, "123.jpg")
	_ = os.WriteFile(testCover, []byte("fake image content 123"), 0o644)
	testUpload := filepath.Join(uploadsDir, "avatar.png")
	_ = os.WriteFile(testUpload, []byte("fake upload content"), 0o644)

	r := gin.New()
	r.Use(StaticCache(tmpDir))
	r.Static("/static", tmpDir)

	t.Run("remote covers cache header and etag", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/static/remote/covers/123.jpg", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if cc := w.Header().Get("Cache-Control"); cc != "public, max-age=604800" {
			t.Errorf("expected Cache-Control: public, max-age=604800, got %q", cc)
		}
		etag := w.Header().Get("ETag")
		if etag == "" {
			t.Fatalf("expected ETag header to be present")
		}

		// Second request with If-None-Match
		req2, _ := http.NewRequest(http.MethodGet, "/static/remote/covers/123.jpg", nil)
		req2.Header.Set("If-None-Match", etag)
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req2)

		if w2.Code != http.StatusNotModified {
			t.Fatalf("expected 304 Not Modified, got %d", w2.Code)
		}
		if w2.Body.Len() != 0 {
			t.Errorf("expected empty body on 304, got %d bytes", w2.Body.Len())
		}
		if w2.Header().Get("ETag") != etag {
			t.Errorf("expected ETag %q on 304, got %q", etag, w2.Header().Get("ETag"))
		}
	})

	t.Run("uploads cache header no-cache must-revalidate", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/static/uploads/avatar.png", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if cc := w.Header().Get("Cache-Control"); cc != "no-cache, must-revalidate" {
			t.Errorf("expected Cache-Control: no-cache, must-revalidate, got %q", cc)
		}
		etag := w.Header().Get("ETag")
		if etag == "" {
			t.Fatalf("expected ETag header to be present")
		}

		// 304 test for uploads too
		req2, _ := http.NewRequest(http.MethodGet, "/static/uploads/avatar.png", nil)
		req2.Header.Set("If-None-Match", etag)
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req2)

		if w2.Code != http.StatusNotModified {
			t.Fatalf("expected 304 Not Modified, got %d", w2.Code)
		}
		if w2.Body.Len() != 0 {
			t.Errorf("expected empty body on 304, got %d bytes", w2.Body.Len())
		}
	})
}

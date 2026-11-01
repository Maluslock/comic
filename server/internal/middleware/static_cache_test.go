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

// TestStaticCache_BlocksPrivateArchive 覆盖 _orig_covers 备份目录的封堵。
//
// 归档原图既不被业务引用也不该被公开下载，但 router.Static 会把 ./static 整棵树挂出去，
// 所以必须在中间件层统一 404，并且不能影响同目录下正常封面的可访问性。
func TestStaticCache_BlocksPrivateArchive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	archiveDir := filepath.Join(tmpDir, "remote", "_orig_covers")
	_ = os.MkdirAll(archiveDir, 0o755)
	_ = os.WriteFile(filepath.Join(archiveDir, "1006166.jpg"), []byte("original backup bytes"), 0o644)

	// 对照组：正常封面必须仍可访问。
	coversDir := filepath.Join(tmpDir, "remote", "covers")
	_ = os.MkdirAll(coversDir, 0o755)
	_ = os.WriteFile(filepath.Join(coversDir, "1006166.webp"), []byte("optimized cover"), 0o644)

	r := gin.New()
	r.Use(StaticCache(tmpDir))
	r.Static("/static", tmpDir)

	cases := []struct {
		name     string
		path     string
		wantCode int
	}{
		{"归档文件被拒绝", "/static/remote/_orig_covers/1006166.jpg", http.StatusNotFound},
		{"归档目录本身被拒绝", "/static/remote/_orig_covers", http.StatusNotFound},
		{"正常封面不受影响", "/static/remote/covers/1006166.webp", http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantCode {
				t.Fatalf("%s: expected %d, got %d", tc.path, tc.wantCode, w.Code)
			}
			if tc.wantCode == http.StatusNotFound && w.Body.Len() != 0 {
				t.Errorf("非 200 不应带内容，实际 %d 字节", w.Body.Len())
			}
		})
	}

	// 带 Accept-Encoding 也必须 404，避免又出现「状态码随 AE 变化」的回归。
	req, _ := http.NewRequest(http.MethodGet, "/static/remote/_orig_covers/1006166.jpg", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("带 Accept-Encoding 时归档仍应 404，实际 %d", w.Code)
	}
}

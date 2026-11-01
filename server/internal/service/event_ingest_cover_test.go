package service

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCoverExtsAndFindExisting(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Create a dummy base.webp and base.jpg
	webpFile := filepath.Join(tmpDir, "test1.webp")
	jpgFile := filepath.Join(tmpDir, "test1.jpg")
	_ = os.WriteFile(webpFile, []byte("fake webp"), 0o644)
	_ = os.WriteFile(jpgFile, []byte("fake jpg"), 0o644)

	// findExistingCover should hit webp first
	hit := findExistingCover(tmpDir, "test1")
	expected := coverURLPrefix + "test1.webp"
	if hit != expected {
		t.Fatalf("expected findExistingCover to return %q, got %q", expected, hit)
	}

	// 2. Only jpg exists
	_ = os.Remove(webpFile)
	hitJpg := findExistingCover(tmpDir, "test1")
	expectedJpg := coverURLPrefix + "test1.jpg"
	if hitJpg != expectedJpg {
		t.Fatalf("expected findExistingCover to return %q, got %q", expectedJpg, hitJpg)
	}
}

func TestDownloadCover_CompressWebP(t *testing.T) {
	// Create an 800x600 test JPEG image (> 750 max side)
	src := image.NewRGBA(image.Rect(0, 0, 800, 600))
	for y := 0; y < 600; y++ {
		for x := 0; x < 800; x++ {
			src.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 120, A: 255})
		}
	}
	var jpegBuf bytes.Buffer
	if err := jpeg.Encode(&jpegBuf, src, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("failed to encode jpeg: %v", err)
	}
	jpegBytes := jpegBuf.Bytes()

	// HTTP test server serving this jpeg
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(jpegBytes)
	}))
	defer ts.Close()

	svc := &EventIngestService{}
	tmpDir := t.TempDir()

	name, err := svc.downloadCover(context.Background(), tmpDir, "event_100", ts.URL)
	if err != nil {
		t.Fatalf("downloadCover failed: %v", err)
	}
	if name != "event_100.webp" {
		t.Fatalf("expected downloaded cover to be event_100.webp, got %q", name)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, name))
	if err != nil {
		t.Fatalf("failed to read saved cover: %v", err)
	}
	if len(content) == 0 {
		t.Fatalf("saved cover is empty")
	}

	// Verify it decoded as webp
	img, format, err := image.Decode(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("failed to decode saved webp: %v", err)
	}
	if format != "webp" {
		t.Errorf("expected format webp, got %s", format)
	}
	b := img.Bounds()
	if b.Dx() > 750 || b.Dy() > 750 {
		t.Errorf("expected max side <= 750, got %dx%d", b.Dx(), b.Dy())
	}
	if b.Dx() != 750 {
		t.Errorf("expected width 750, got %d", b.Dx())
	}
}

func TestDownloadCover_FallbackOnCorrupt(t *testing.T) {
	// A valid size (> 2000 bytes) but corrupt image bytes
	corruptBytes := bytes.Repeat([]byte("corrupt data payload "), 150) // ~3150 bytes

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(corruptBytes)
	}))
	defer ts.Close()

	svc := &EventIngestService{}
	tmpDir := t.TempDir()

	name, err := svc.downloadCover(context.Background(), tmpDir, "event_corrupt", ts.URL)
	if err != nil {
		t.Fatalf("downloadCover should not fail when fallback to original: %v", err)
	}
	if name != "event_corrupt.jpg" {
		t.Fatalf("expected fallback to event_corrupt.jpg, got %q", name)
	}

	saved, _ := os.ReadFile(filepath.Join(tmpDir, name))
	if !bytes.Equal(saved, corruptBytes) {
		t.Fatalf("saved content does not match original bytes on fallback")
	}
}

func TestDownloadCover_OverMaxBytes(t *testing.T) {
	// > 3MB
	oversizedBytes := bytes.Repeat([]byte("A"), coverMaxBytes+10)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(oversizedBytes)
	}))
	defer ts.Close()

	svc := &EventIngestService{}
	tmpDir := t.TempDir()

	_, err := svc.downloadCover(context.Background(), tmpDir, "event_huge", ts.URL)
	if err == nil {
		t.Fatalf("expected error for oversized cover, got nil")
	}
}

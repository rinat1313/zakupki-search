package eissearch

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadCAPoolIncludesMintsifry(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	// internal/eissearch → repo root/certs
	caDir := filepath.Join(filepath.Dir(file), "..", "..", "certs")
	pool := loadCAPool(caDir)
	if pool == nil {
		t.Fatal("expected non-nil pool with certs/")
	}
}

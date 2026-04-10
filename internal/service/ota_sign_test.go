package service

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSignPackageSourceWithLocalFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "firmware.bin")
	content := []byte("firmware-local")
	if err := os.WriteFile(filePath, content, 0o600); err != nil {
		t.Fatalf("write temp file failed: %v", err)
	}

	got, err := signPackageSource(filePath, "SHA256")
	if err != nil {
		t.Fatalf("signPackageSource returned error: %v", err)
	}

	sum := sha256.Sum256(content)
	want := hex.EncodeToString(sum[:])
	if got != want {
		t.Fatalf("unexpected sha256 signature, got %s want %s", got, want)
	}
}

func TestSignPackageSourceWithRemoteURL(t *testing.T) {
	t.Parallel()

	content := []byte("firmware-remote")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(content)
	}))
	defer server.Close()

	got, err := signPackageSource(server.URL, "MD5")
	if err != nil {
		t.Fatalf("signPackageSource returned error: %v", err)
	}

	sum := md5.Sum(content)
	want := hex.EncodeToString(sum[:])
	if got != want {
		t.Fatalf("unexpected md5 signature, got %s want %s", got, want)
	}
}

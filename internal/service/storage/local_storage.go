package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// LocalStorage is a controlled, offline-safe attachment storage adapter.
// It validates content types and sizes against an allowlist before persisting
// to disk. Files are stored in a deterministic directory layout.
type LocalStorage struct {
	rootDir        string
	maxSize        int64
	allowedTypes   map[string]bool
	allowedExts    map[string]bool
	mu             sync.Mutex
	forbidSymlinks bool
}

// Config configures LocalStorage.
type Config struct {
	RootDir        string
	MaxSizeBytes   int64
	AllowedTypes   []string
	AllowedExts    []string
	ForbidSymlinks bool
}

// DefaultAllowedTypes is the default MIME-type allowlist.
var DefaultAllowedTypes = []string{
	"image/png", "image/jpeg", "image/webp", "image/gif",
	"application/pdf", "text/plain",
}

// DefaultAllowedExts is the default file-extension allowlist.
var DefaultAllowedExts = []string{".png", ".jpg", ".jpeg", ".webp", ".gif", ".pdf", ".txt"}

// New returns a LocalStorage. The root directory is created if it does not exist.
func New(cfg Config) (*LocalStorage, error) {
	if cfg.RootDir == "" {
		return nil, errors.New("root dir is required")
	}
	if cfg.MaxSizeBytes == 0 {
		cfg.MaxSizeBytes = 8 * 1024 * 1024 // 8 MiB
	}
	if len(cfg.AllowedTypes) == 0 {
		cfg.AllowedTypes = DefaultAllowedTypes
	}
	if len(cfg.AllowedExts) == 0 {
		cfg.AllowedExts = DefaultAllowedExts
	}
	if err := os.MkdirAll(cfg.RootDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir storage: %w", err)
	}
	ls := &LocalStorage{
		rootDir:        cfg.RootDir,
		maxSize:        cfg.MaxSizeBytes,
		allowedTypes:   map[string]bool{},
		allowedExts:    map[string]bool{},
		forbidSymlinks: cfg.ForbidSymlinks,
	}
	for _, t := range cfg.AllowedTypes {
		ls.allowedTypes[t] = true
	}
	for _, e := range cfg.AllowedExts {
		ls.allowedExts[e] = true
	}
	return ls, nil
}

// Save persists a file with a stable, hash-derived filename. It validates type
// and size before writing. Returns the relative path that can be used to
// retrieve the file later.
func (s *LocalStorage) Save(ctx context.Context, name string, contentType string, data []byte) (string, error) {
	if !s.allowedTypes[contentType] {
		return "", fmt.Errorf("content type not allowed: %s", contentType)
	}
	if int64(len(data)) > s.maxSize {
		return "", fmt.Errorf("file size %d exceeds max %d", len(data), s.maxSize)
	}
	ext := strings.ToLower(filepath.Ext(name))
	if !s.allowedExts[ext] {
		return "", fmt.Errorf("file extension not allowed: %s", ext)
	}
	sum := sha256.Sum256(data)
	hex := hex.EncodeToString(sum[:])
	// Layout: <rootDir>/<first2>/<hex><ext>
	first2 := hex[:2]
	subdir := filepath.Join(s.rootDir, first2)
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir subdir: %w", err)
	}
	rel := filepath.Join(first2, hex+ext)
	abs := filepath.Join(s.rootDir, rel)
	// Atomic-ish: write to a temp file then rename.
	tmp := abs + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}
	if err := os.Rename(tmp, abs); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("rename file: %w", err)
	}
	return rel, nil
}

// Read retrieves a file by relative path. It enforces path containment to
// prevent path traversal.
func (s *LocalStorage) Read(ctx context.Context, rel string) (string, []byte, error) {
	cleanRel := filepath.Clean(rel)
	if strings.Contains(cleanRel, "..") {
		return "", nil, errors.New("invalid path")
	}
	abs := filepath.Join(s.rootDir, cleanRel)
	if s.forbidSymlinks {
		if fi, err := os.Lstat(abs); err == nil && fi.Mode()&os.ModeSymlink != 0 {
			return "", nil, errors.New("symlinks are forbidden")
		}
	}
	f, err := os.Open(abs)
	if err != nil {
		return "", nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return "", nil, fmt.Errorf("read file: %w", err)
	}
	ct := guessContentType(abs)
	return ct, data, nil
}

// Delete removes a file.
func (s *LocalStorage) Delete(ctx context.Context, rel string) error {
	cleanRel := filepath.Clean(rel)
	if strings.Contains(cleanRel, "..") {
		return errors.New("invalid path")
	}
	abs := filepath.Join(s.rootDir, cleanRel)
	return os.Remove(abs)
}

// RootDir returns the storage root directory.
func (s *LocalStorage) RootDir() string { return s.rootDir }

// guessContentType is a simple extension-based MIME guesser.
func guessContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

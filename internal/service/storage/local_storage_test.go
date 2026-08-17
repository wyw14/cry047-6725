package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalStorage_SaveAndRead(t *testing.T) {
	tmp := t.TempDir()
	ls, err := New(Config{RootDir: tmp})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	ctx := context.Background()
	data := []byte("hello world")
	rel, err := ls.Save(ctx, "test.txt", "text/plain", data)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	ct, got, err := ls.Read(ctx, rel)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("expected %q, got %q", data, got)
	}
	if ct != "text/plain" {
		t.Errorf("expected text/plain, got %s", ct)
	}
}

func TestLocalStorage_RejectsUnknownType(t *testing.T) {
	tmp := t.TempDir()
	ls, err := New(Config{RootDir: tmp})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	_, err = ls.Save(context.Background(), "evil.exe", "application/x-msdownload", []byte("x"))
	if err == nil {
		t.Fatalf("expected error for unknown type, got nil")
	}
}

func TestLocalStorage_RejectsOversize(t *testing.T) {
	tmp := t.TempDir()
	ls, err := New(Config{RootDir: tmp, MaxSizeBytes: 4})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	_, err = ls.Save(context.Background(), "big.txt", "text/plain", []byte("12345"))
	if err == nil {
		t.Fatalf("expected error for oversize, got nil")
	}
}

func TestLocalStorage_RejectsPathTraversal(t *testing.T) {
	tmp := t.TempDir()
	ls, _ := New(Config{RootDir: tmp})
	_, _, err := ls.Read(context.Background(), "../../../etc/passwd")
	if err == nil {
		t.Fatalf("expected path traversal error, got nil")
	}
}

func TestLocalStorage_AtomicRename(t *testing.T) {
	tmp := t.TempDir()
	ls, err := New(Config{RootDir: tmp})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	rel, err := ls.Save(context.Background(), "a.txt", "text/plain", []byte("x"))
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	// No .tmp file should be left behind.
	_ = filepath.Walk(tmp, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if strings.HasSuffix(path, ".tmp") {
			t.Errorf("left-behind tmp file: %s", path)
		}
		return nil
	})
	// Same content produces the same path (content-addressed).
	rel2, _ := ls.Save(context.Background(), "different-name.txt", "text/plain", []byte("x"))
	_ = rel2
	_ = rel
}

func TestLocalStorage_Delete(t *testing.T) {
	tmp := t.TempDir()
	ls, _ := New(Config{RootDir: tmp})
	rel, _ := ls.Save(context.Background(), "x.txt", "text/plain", []byte("x"))
	if err := ls.Delete(context.Background(), rel); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, _, err := ls.Read(context.Background(), rel)
	if err == nil {
		t.Fatalf("expected error after delete, got nil")
	}
}

func TestLocalStorage_SameContentSamePath(t *testing.T) {
	tmp := t.TempDir()
	ls, _ := New(Config{RootDir: tmp})
	rel1, _ := ls.Save(context.Background(), "a.txt", "text/plain", []byte("x"))
	rel2, _ := ls.Save(context.Background(), "b.txt", "text/plain", []byte("x"))
	if rel1 != rel2 {
		t.Errorf("expected same path, got %s vs %s", rel1, rel2)
	}
}

func TestLocalStorage_ForbidsSymlinks(t *testing.T) {
	tmp := t.TempDir()
	ls, _ := New(Config{RootDir: tmp, ForbidSymlinks: true})
	target := filepath.Join(tmp, "target.txt")
	_ = os.WriteFile(target, []byte("y"), 0o644)
	link := filepath.Join(tmp, "link.txt")
	_ = os.Symlink(target, link)
	_, _, err := ls.Read(context.Background(), "link.txt")
	if err == nil {
		t.Fatalf("expected error reading symlink, got nil")
	}
}

var _ = os.Remove

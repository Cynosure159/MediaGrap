package files

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateTarget(t *testing.T) {
	dir := t.TempDir()

	// 1. Non-existent file
	willReplace, conflict, err := ValidateTarget(filepath.Join(dir, "nonexistent.nfo"))
	if err != nil || willReplace || conflict {
		t.Fatalf("expected non-existent: willReplace=false, conflict=false, got %v, %v, err=%v", willReplace, conflict, err)
	}

	// 2. Regular file
	regPath := filepath.Join(dir, "regular.nfo")
	_ = os.WriteFile(regPath, []byte("content"), 0o600)
	willReplace, conflict, err = ValidateTarget(regPath)
	if err != nil || !willReplace || conflict {
		t.Fatalf("expected regular file: willReplace=true, conflict=false, got %v, %v, err=%v", willReplace, conflict, err)
	}

	// 3. Symlink target (conflict)
	symPath := filepath.Join(dir, "symlink.nfo")
	if err := os.Symlink(regPath, symPath); err == nil {
		willReplace, conflict, err = ValidateTarget(symPath)
		if err != nil || !willReplace || !conflict {
			t.Fatalf("expected symlink conflict: willReplace=true, conflict=true, got %v, %v, err=%v", willReplace, conflict, err)
		}
	}
}

func TestAtomicWriteFile(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "test.nfo")
	content := []byte("<movie><title>Matrix</title></movie>")

	if err := AtomicWriteFile(targetPath, content); err != nil {
		t.Fatalf("AtomicWriteFile failed: %v", err)
	}

	read, err := os.ReadFile(targetPath)
	if err != nil || !bytes.Equal(read, content) {
		t.Fatalf("unexpected content read: got %s, want %s (err: %v)", string(read), string(content), err)
	}

	// Overwrite atomically
	newContent := []byte("<movie><title>Matrix Reloaded</title></movie>")
	if err := AtomicWriteFile(targetPath, newContent); err != nil {
		t.Fatalf("AtomicWriteFile overwrite failed: %v", err)
	}

	read, err = os.ReadFile(targetPath)
	if err != nil || !bytes.Equal(read, newContent) {
		t.Fatalf("unexpected content after overwrite: got %s, want %s", string(read), string(newContent))
	}
}

func TestWriteTempFromReaderWithLimit(t *testing.T) {
	dir := t.TempDir()

	// 1. Within limit
	data := []byte("hello safe write")
	tempPath, err := WriteTempFromReader(dir, ".test-*.tmp", bytes.NewReader(data), 1024)
	if err != nil {
		t.Fatalf("WriteTempFromReader failed: %v", err)
	}
	defer os.Remove(tempPath)

	read, _ := os.ReadFile(tempPath)
	if !bytes.Equal(read, data) {
		t.Fatalf("expected %s, got %s", string(data), string(read))
	}

	// 2. Exceeds limit
	bigData := []byte(strings.Repeat("a", 200))
	_, err = WriteTempFromReader(dir, ".test-*.tmp", bytes.NewReader(bigData), 100)
	if err == nil || !strings.Contains(err.Error(), "exceeds maximum safety limit") {
		t.Fatalf("expected safety limit error, got %v", err)
	}
}

func TestCheckAllowed(t *testing.T) {
	allowed := func(path string) bool {
		return strings.HasPrefix(path, "/media")
	}

	if err := CheckAllowed("/media/movies/test.mkv", allowed); err != nil {
		t.Fatalf("expected allowed, got %v", err)
	}

	if err := CheckAllowed("/etc/passwd", allowed); err != ErrOutsideMediaRoot {
		t.Fatalf("expected ErrOutsideMediaRoot, got %v", err)
	}
}

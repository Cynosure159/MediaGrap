package files

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var (
	ErrOutsideMediaRoot = errors.New("target path is outside configured media roots")
	ErrTargetConflict   = errors.New("target is not a regular file and cannot be replaced")
)

// ValidateTarget checks whether targetPath exists, is regular, or has conflict (e.g. symlink/directory).
func ValidateTarget(targetPath string) (willReplace bool, conflict bool, err error) {
	info, statErr := os.Lstat(targetPath)
	if errors.Is(statErr, os.ErrNotExist) {
		return false, false, nil
	}
	if statErr != nil {
		return false, false, fmt.Errorf("stat target path: %w", statErr)
	}
	willReplace = true
	conflict = info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular()
	return willReplace, conflict, nil
}

// CheckAllowed verifies whether the path is permitted under configured roots.
func CheckAllowed(path string, allowed func(string) bool) error {
	if allowed == nil || !allowed(path) {
		return ErrOutsideMediaRoot
	}
	return nil
}

// AtomicWriteFile writes content to a temporary file in target directory, syncs, closes, and atomically renames.
func AtomicWriteFile(targetPath string, content []byte) error {
	dir := filepath.Dir(targetPath)
	temp, err := os.CreateTemp(dir, ".mediagrap-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)

	if _, err := temp.Write(content); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync temporary file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}
	if err := os.Rename(tempName, targetPath); err != nil {
		return fmt.Errorf("atomic rename to target: %w", err)
	}
	return nil
}

// WriteTempFromReader copies from a reader to a temporary file in dir with a max byte limit.
func WriteTempFromReader(dir string, pattern string, reader io.Reader, limitBytes int64) (string, error) {
	temp, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	tempName := temp.Name()

	written, copyErr := io.Copy(temp, io.LimitReader(reader, limitBytes+1))
	if copyErr == nil && limitBytes > 0 && written > limitBytes {
		copyErr = fmt.Errorf("content exceeds maximum safety limit of %d bytes", limitBytes)
	}
	if copyErr == nil {
		copyErr = temp.Sync()
	}
	if closeErr := temp.Close(); copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		_ = os.Remove(tempName)
		return "", copyErr
	}
	return tempName, nil
}

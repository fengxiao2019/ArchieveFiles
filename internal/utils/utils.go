package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CalculateSize calculates the total size of a file or directory
func CalculateSize(path string) int64 {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	if err != nil {
		// Log error but don't fail - return 0 size on error
		return 0
	}
	return size
}

// FormatBytes formats bytes in human-readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatDuration formats duration in human-readable format
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm%.0fs", d.Minutes(), d.Seconds()-60*d.Minutes())
	}
	return fmt.Sprintf("%.0fh%.0fm", d.Hours(), d.Minutes()-60*d.Hours())
}

// TruncateString truncates string to specified length
func TruncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

// CopyFile copies a file from source to destination
func CopyFile(sourcePath, targetPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer sourceFile.Close()

	targetFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %v", err)
	}
	defer targetFile.Close()

	_, err = io.Copy(targetFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
	}

	// Preserve file permissions
	if sourceInfo, err := os.Stat(sourcePath); err == nil {
		if chmodErr := os.Chmod(targetPath, sourceInfo.Mode()); chmodErr != nil {
			// Log error but don't fail the copy operation
			log.Printf("Warning: Failed to preserve file permissions for %s: %v", targetPath, chmodErr)
		}
	}

	return nil
}

// ShouldIncludeFile checks if a file should be included based on patterns
func ShouldIncludeFile(path, includePattern, excludePattern string) bool {
	filename := filepath.Base(path)

	// Check exclude pattern first
	if excludePattern != "" {
		patterns := strings.Split(excludePattern, ",")
		for _, pattern := range patterns {
			pattern = strings.TrimSpace(pattern)
			if matched, _ := filepath.Match(pattern, filename); matched {
				return false
			}
		}
	}

	// Check include pattern
	if includePattern != "" {
		patterns := strings.Split(includePattern, ",")
		for _, pattern := range patterns {
			pattern = strings.TrimSpace(pattern)
			if matched, _ := filepath.Match(pattern, filename); matched {
				return true
			}
		}
		return false // If include pattern is specified but doesn't match
	}

	return true // Include by default if no patterns specified
}

// BytesEqual compares two byte slices for equality
func BytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

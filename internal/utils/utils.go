package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
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
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) - minutes*60
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) - hours*60
	return fmt.Sprintf("%dh%dm", hours, minutes)
}

// TruncateString truncates string to specified length
func TruncateString(s string, length int) string {
	// Convert to runes for proper Unicode handling
	runes := []rune(s)
	if len(runes) <= length {
		return s
	}
	if length < 3 {
		if length <= 0 {
			return ""
		}
		return string(runes[:length])
	}
	return string(runes[:length-3]) + "..."
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

// ReplaceDateVars 替换 $(date +%Y%m%d_%H%M%S) 为当前时间戳
func ReplaceDateVars(s string) string {
	pattern := regexp.MustCompile(`\$\(\s*date \+%Y%m%d_%H%M%S\s*\)`)
	return pattern.ReplaceAllStringFunc(s, func(_ string) string {
		return time.Now().Format("20060102_150405")
	})
}

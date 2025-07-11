package utils

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"Zero bytes", 0, "0 B"},
		{"Single byte", 1, "1 B"},
		{"Bytes", 512, "512 B"},
		{"Bytes at boundary", 1023, "1023 B"},
		{"Kilobytes", 1024, "1.0 KB"},
		{"Kilobytes with decimals", 1536, "1.5 KB"},
		{"Megabytes", 1024 * 1024, "1.0 MB"},
		{"Megabytes with decimals", 1024*1024 + 512*1024, "1.5 MB"},
		{"Gigabytes", 1024 * 1024 * 1024, "1.0 GB"},
		{"Terabytes", 1024 * 1024 * 1024 * 1024, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatBytes(tt.bytes)
			if got != tt.expected {
				t.Errorf("FormatBytes(%d) = %v, want %v", tt.bytes, got, tt.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"Zero duration", 0, "0s"},
		{"Seconds", 30 * time.Second, "30s"},
		{"Under a minute", 59 * time.Second, "59s"},
		{"Exactly one minute", 60 * time.Second, "1m0s"},
		{"Minutes and seconds", 90 * time.Second, "1m30s"},
		{"Multiple minutes", 5*time.Minute + 30*time.Second, "5m30s"},
		{"Exactly one hour", 60 * time.Minute, "1h0m"},
		{"Hours and minutes", 90 * time.Minute, "1h30m"},
		{"Multiple hours", 5*time.Hour + 30*time.Minute, "5h30m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDuration(tt.duration)
			if got != tt.expected {
				t.Errorf("FormatDuration(%v) = %v, want %v", tt.duration, got, tt.expected)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		length   int
		expected string
	}{
		{"Empty string", "", 10, ""},
		{"Short string", "hello", 10, "hello"},
		{"Exact length", "hello", 5, "hello"},
		{"Truncate required", "hello world", 8, "hello..."},
		{"Truncate to very short", "hello world", 5, "he..."},
		{"Truncate to minimum", "hello world", 3, "..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateString(tt.input, tt.length)
			if got != tt.expected {
				t.Errorf("TruncateString(%q, %d) = %q, want %q", tt.input, tt.length, got, tt.expected)
			}
		})
	}
}

func TestShouldIncludeFile(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		includePattern string
		excludePattern string
		expected       bool
	}{
		{"No patterns", "/path/to/file.txt", "", "", true},
		{"Include match", "/path/to/file.db", "*.db", "", true},
		{"Include no match", "/path/to/file.txt", "*.db", "", false},
		{"Exclude match", "/path/to/file.tmp", "", "*.tmp", false},
		{"Exclude no match", "/path/to/file.db", "", "*.tmp", true},
		{"Include and exclude both match", "/path/to/file.db", "*.db", "*.db", false},
		{"Include match exclude no match", "/path/to/file.db", "*.db", "*.tmp", true},
		{"Multiple include patterns", "/path/to/file.sqlite", "*.db,*.sqlite", "", true},
		{"Multiple exclude patterns", "/path/to/file.tmp", "", "*.tmp,*.cache", false},
		{"Complex patterns", "/path/to/app.db", "*.db,*.sqlite", "temp*,*cache*", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldIncludeFile(tt.path, tt.includePattern, tt.excludePattern)
			if got != tt.expected {
				t.Errorf("ShouldIncludeFile(%q, %q, %q) = %v, want %v",
					tt.path, tt.includePattern, tt.excludePattern, got, tt.expected)
			}
		})
	}
}

func TestBytesEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        []byte
		b        []byte
		expected bool
	}{
		{"Empty slices", []byte{}, []byte{}, true},
		{"Nil slices", nil, nil, true},
		{"Empty vs nil", []byte{}, nil, true},
		{"Equal slices", []byte{1, 2, 3}, []byte{1, 2, 3}, true},
		{"Different lengths", []byte{1, 2, 3}, []byte{1, 2}, false},
		{"Different content", []byte{1, 2, 3}, []byte{1, 2, 4}, false},
		{"Single byte equal", []byte{42}, []byte{42}, true},
		{"Single byte different", []byte{42}, []byte{24}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BytesEqual(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("BytesEqual(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "utils_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test successful copy
	t.Run("Successful copy", func(t *testing.T) {
		sourceFile := filepath.Join(tempDir, "source.txt")
		targetFile := filepath.Join(tempDir, "target.txt")

		// Create source file
		content := []byte("Hello, World!\nThis is a test file.")
		err := os.WriteFile(sourceFile, content, 0644)
		if err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		// Copy file
		err = CopyFile(sourceFile, targetFile)
		if err != nil {
			t.Errorf("CopyFile failed: %v", err)
		}

		// Verify content
		copiedContent, err := os.ReadFile(targetFile)
		if err != nil {
			t.Errorf("Failed to read copied file: %v", err)
		}

		if !BytesEqual(content, copiedContent) {
			t.Errorf("Copied content doesn't match original")
		}

		// Verify permissions
		sourceInfo, _ := os.Stat(sourceFile)
		targetInfo, _ := os.Stat(targetFile)
		if sourceInfo.Mode() != targetInfo.Mode() {
			t.Errorf("File permissions not preserved")
		}
	})

	// Test copy non-existent source
	t.Run("Non-existent source", func(t *testing.T) {
		sourceFile := filepath.Join(tempDir, "non_existent.txt")
		targetFile := filepath.Join(tempDir, "target2.txt")

		err := CopyFile(sourceFile, targetFile)
		if err == nil {
			t.Error("Expected error for non-existent source file")
		}
	})

	// Test copy to invalid destination
	t.Run("Invalid destination", func(t *testing.T) {
		sourceFile := filepath.Join(tempDir, "source2.txt")
		targetFile := "/invalid/path/target.txt"

		// Create source file
		content := []byte("test content")
		err := os.WriteFile(sourceFile, content, 0644)
		if err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		err = CopyFile(sourceFile, targetFile)
		if err == nil {
			t.Error("Expected error for invalid destination path")
		}
	})
}

func TestCalculateSize(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "utils_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test with single file
	t.Run("Single file", func(t *testing.T) {
		file := filepath.Join(tempDir, "test.txt")
		content := []byte("Hello, World!")
		err := os.WriteFile(file, content, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		size := CalculateSize(file)
		if size != int64(len(content)) {
			t.Errorf("CalculateSize() = %d, want %d", size, len(content))
		}
	})

	// Test with directory
	t.Run("Directory with files", func(t *testing.T) {
		subDir := filepath.Join(tempDir, "subdir")
		err := os.MkdirAll(subDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create subdirectory: %v", err)
		}

		file1 := filepath.Join(tempDir, "file1.txt")
		file2 := filepath.Join(subDir, "file2.txt")
		content1 := []byte("content1")
		content2 := []byte("content2")

		err = os.WriteFile(file1, content1, 0644)
		if err != nil {
			t.Fatalf("Failed to create file1: %v", err)
		}

		err = os.WriteFile(file2, content2, 0644)
		if err != nil {
			t.Fatalf("Failed to create file2: %v", err)
		}

		size := CalculateSize(tempDir)
		expectedSize := int64(len(content1) + len(content2))

		// Check that the size is at least the expected size (could be more due to filesystem metadata)
		if size < expectedSize {
			t.Errorf("CalculateSize() = %d, should be at least %d", size, expectedSize)
		}

		// Also check that it's not unreasonably large (should be less than 10x expected)
		if size > expectedSize*10 {
			t.Errorf("CalculateSize() = %d, unexpectedly large compared to expected %d", size, expectedSize)
		}
	})

	// Test with non-existent path
	t.Run("Non-existent path", func(t *testing.T) {
		size := CalculateSize("/non/existent/path")
		if size != 0 {
			t.Errorf("CalculateSize() = %d, want 0 for non-existent path", size)
		}
	})
}

func TestFormatBytes_EdgeCases(t *testing.T) {
	// Test very large numbers
	t.Run("Very large bytes", func(t *testing.T) {
		// Test petabytes
		pb := int64(1024 * 1024 * 1024 * 1024 * 1024)
		result := FormatBytes(pb)
		if result != "1.0 PB" {
			t.Errorf("FormatBytes(1PB) = %s, want 1.0 PB", result)
		}
	})

	// Test negative numbers (edge case)
	t.Run("Negative bytes", func(t *testing.T) {
		result := FormatBytes(-1024)
		// Should handle negative gracefully
		if result == "" {
			t.Error("FormatBytes should handle negative numbers")
		}
	})
}

func TestTruncateString_EdgeCases(t *testing.T) {
	// Test edge cases for truncation
	t.Run("Length less than 3", func(t *testing.T) {
		result := TruncateString("hello", 2)
		if len(result) > 2 {
			t.Errorf("TruncateString should not exceed specified length")
		}
	})

	t.Run("Unicode characters", func(t *testing.T) {
		result := TruncateString("你好世界测试", 5)
		if result != "你好..." {
			t.Errorf("TruncateString should handle unicode properly, got %s", result)
		}
	})
}

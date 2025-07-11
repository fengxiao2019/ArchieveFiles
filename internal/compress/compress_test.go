package compress

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCompressDirectory(t *testing.T) {
	// Create temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "compress_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test directory structure
	sourceDir := filepath.Join(tempDir, "source")
	err = os.MkdirAll(sourceDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Test successful compression
	t.Run("Successful compression", func(t *testing.T) {
		// Create test files
		testFiles := map[string]string{
			"file1.txt":         "Hello, World!",
			"file2.db":          "Database content",
			"subdir/file3.log":  "Log file content",
			"subdir/file4.conf": "Configuration data",
		}

		for relPath, content := range testFiles {
			fullPath := filepath.Join(sourceDir, relPath)
			err := os.MkdirAll(filepath.Dir(fullPath), 0755)
			if err != nil {
				t.Fatalf("Failed to create directory for %s: %v", relPath, err)
			}
			err = os.WriteFile(fullPath, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file %s: %v", relPath, err)
			}
		}

		// Compress directory
		archivePath := filepath.Join(tempDir, "test.tar.gz")
		err := CompressDirectory(sourceDir, archivePath)
		if err != nil {
			t.Errorf("CompressDirectory failed: %v", err)
		}

		// Verify archive was created
		if _, err := os.Stat(archivePath); os.IsNotExist(err) {
			t.Error("Archive file was not created")
		}

		// Verify archive contents
		err = verifyArchiveContents(archivePath, testFiles)
		if err != nil {
			t.Errorf("Archive verification failed: %v", err)
		}
	})

	// Test with empty directory
	t.Run("Empty directory", func(t *testing.T) {
		emptyDir := filepath.Join(tempDir, "empty")
		err := os.MkdirAll(emptyDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create empty directory: %v", err)
		}

		archivePath := filepath.Join(tempDir, "empty.tar.gz")
		err = CompressDirectory(emptyDir, archivePath)
		if err != nil {
			t.Errorf("CompressDirectory failed for empty directory: %v", err)
		}

		// Verify archive was created
		if _, err := os.Stat(archivePath); os.IsNotExist(err) {
			t.Error("Archive file was not created for empty directory")
		}
	})

	// Test with non-existent source directory
	t.Run("Non-existent source", func(t *testing.T) {
		nonExistentDir := filepath.Join(tempDir, "non_existent")
		archivePath := filepath.Join(tempDir, "non_existent.tar.gz")

		err := CompressDirectory(nonExistentDir, archivePath)
		if err == nil {
			t.Error("Expected error for non-existent source directory")
		}
	})

	// Test with invalid target path
	t.Run("Invalid target path", func(t *testing.T) {
		invalidPath := "/invalid/path/archive.tar.gz"

		err := CompressDirectory(sourceDir, invalidPath)
		if err == nil {
			t.Error("Expected error for invalid target path")
		}
	})

	// Test with single file
	t.Run("Single file", func(t *testing.T) {
		singleFileDir := filepath.Join(tempDir, "single")
		err := os.MkdirAll(singleFileDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create single file directory: %v", err)
		}

		singleFile := filepath.Join(singleFileDir, "single.txt")
		content := "Single file content"
		err = os.WriteFile(singleFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create single file: %v", err)
		}

		archivePath := filepath.Join(tempDir, "single.tar.gz")
		err = CompressDirectory(singleFileDir, archivePath)
		if err != nil {
			t.Errorf("CompressDirectory failed for single file: %v", err)
		}

		// Verify archive contents
		testFiles := map[string]string{
			"single.txt": content,
		}
		err = verifyArchiveContents(archivePath, testFiles)
		if err != nil {
			t.Errorf("Single file archive verification failed: %v", err)
		}
	})

	// Test with large file
	t.Run("Large file", func(t *testing.T) {
		largeFileDir := filepath.Join(tempDir, "large")
		err := os.MkdirAll(largeFileDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create large file directory: %v", err)
		}

		largeFile := filepath.Join(largeFileDir, "large.txt")
		// Create a 1MB file
		largeContent := make([]byte, 1024*1024)
		for i := range largeContent {
			largeContent[i] = byte(i % 256)
		}
		err = os.WriteFile(largeFile, largeContent, 0644)
		if err != nil {
			t.Fatalf("Failed to create large file: %v", err)
		}

		archivePath := filepath.Join(tempDir, "large.tar.gz")
		err = CompressDirectory(largeFileDir, archivePath)
		if err != nil {
			t.Errorf("CompressDirectory failed for large file: %v", err)
		}

		// Verify archive was created and compressed
		archiveInfo, err := os.Stat(archivePath)
		if err != nil {
			t.Errorf("Failed to stat archive: %v", err)
		}

		// Archive should be smaller than original due to compression
		if archiveInfo.Size() >= int64(len(largeContent)) {
			t.Error("Archive should be smaller than original file due to compression")
		}
	})

	// Test with nested directories
	t.Run("Nested directories", func(t *testing.T) {
		nestedDir := filepath.Join(tempDir, "nested")
		err := os.MkdirAll(nestedDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create nested directory: %v", err)
		}

		// Create nested structure
		nestedFiles := map[string]string{
			"root.txt":                      "Root level file",
			"level1/file1.txt":              "Level 1 file",
			"level1/level2/file2.txt":       "Level 2 file",
			"level1/level2/file3.db":        "Database file",
			"level1/level2/level3/deep.log": "Deep nested file",
		}

		for relPath, content := range nestedFiles {
			fullPath := filepath.Join(nestedDir, relPath)
			err := os.MkdirAll(filepath.Dir(fullPath), 0755)
			if err != nil {
				t.Fatalf("Failed to create directory for %s: %v", relPath, err)
			}
			err = os.WriteFile(fullPath, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to create nested file %s: %v", relPath, err)
			}
		}

		archivePath := filepath.Join(tempDir, "nested.tar.gz")
		err = CompressDirectory(nestedDir, archivePath)
		if err != nil {
			t.Errorf("CompressDirectory failed for nested directories: %v", err)
		}

		// Verify archive contents
		err = verifyArchiveContents(archivePath, nestedFiles)
		if err != nil {
			t.Errorf("Nested directory archive verification failed: %v", err)
		}
	})

	// Test with special characters in filenames
	t.Run("Special characters", func(t *testing.T) {
		specialDir := filepath.Join(tempDir, "special")
		err := os.MkdirAll(specialDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create special directory: %v", err)
		}

		// Create files with special characters (avoiding problematic ones for cross-platform)
		specialFiles := map[string]string{
			"file with spaces.txt":      "File with spaces",
			"file-with-dashes.txt":      "File with dashes",
			"file_with_underscores.txt": "File with underscores",
			"file.with.dots.txt":        "File with dots",
		}

		for relPath, content := range specialFiles {
			fullPath := filepath.Join(specialDir, relPath)
			err = os.WriteFile(fullPath, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to create special file %s: %v", relPath, err)
			}
		}

		archivePath := filepath.Join(tempDir, "special.tar.gz")
		err = CompressDirectory(specialDir, archivePath)
		if err != nil {
			t.Errorf("CompressDirectory failed for special characters: %v", err)
		}

		// Verify archive contents
		err = verifyArchiveContents(archivePath, specialFiles)
		if err != nil {
			t.Errorf("Special characters archive verification failed: %v", err)
		}
	})
}

func TestCompressDirectory_Permissions(t *testing.T) {
	// Create temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "compress_perm_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	err = os.MkdirAll(sourceDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create files with different permissions
	testFile1 := filepath.Join(sourceDir, "readable.txt")
	testFile2 := filepath.Join(sourceDir, "executable.txt")

	err = os.WriteFile(testFile1, []byte("readable content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create readable file: %v", err)
	}

	err = os.WriteFile(testFile2, []byte("executable content"), 0755)
	if err != nil {
		t.Fatalf("Failed to create executable file: %v", err)
	}

	// Compress directory
	archivePath := filepath.Join(tempDir, "perm_test.tar.gz")
	err = CompressDirectory(sourceDir, archivePath)
	if err != nil {
		t.Errorf("CompressDirectory failed: %v", err)
	}

	// Verify archive was created
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Error("Archive file was not created")
	}

	// Verify permissions are preserved in archive
	err = verifyArchivePermissions(archivePath)
	if err != nil {
		t.Errorf("Permission verification failed: %v", err)
	}
}

// Helper function to verify archive contents
func verifyArchiveContents(archivePath string, expectedFiles map[string]string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %v", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	foundFiles := make(map[string]string)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %v", err)
		}

		if header.FileInfo().IsDir() {
			continue
		}

		content, err := io.ReadAll(tarReader)
		if err != nil {
			return fmt.Errorf("failed to read file content: %v", err)
		}

		foundFiles[header.Name] = string(content)
	}

	// Check if all expected files are present
	for expectedFile, expectedContent := range expectedFiles {
		if foundContent, exists := foundFiles[expectedFile]; !exists {
			return fmt.Errorf("expected file %s not found in archive", expectedFile)
		} else if foundContent != expectedContent {
			return fmt.Errorf("content mismatch for file %s: expected %s, got %s",
				expectedFile, expectedContent, foundContent)
		}
	}

	// Check for unexpected files
	for foundFile := range foundFiles {
		if _, expected := expectedFiles[foundFile]; !expected {
			return fmt.Errorf("unexpected file %s found in archive", foundFile)
		}
	}

	return nil
}

// Helper function to verify archive permissions
func verifyArchivePermissions(archivePath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %v", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %v", err)
		}

		if header.FileInfo().IsDir() {
			continue
		}

		// Check that permissions are reasonable
		mode := header.FileInfo().Mode()
		if mode == 0 {
			return fmt.Errorf("file %s has zero permissions", header.Name)
		}

		// Check specific files
		if header.Name == "readable.txt" {
			expectedMode := os.FileMode(0644)
			if mode.Perm() != expectedMode {
				return fmt.Errorf("file %s has wrong permissions: expected %v, got %v",
					header.Name, expectedMode, mode.Perm())
			}
		}
	}

	return nil
}

func BenchmarkCompressDirectory(b *testing.B) {
	// Create temporary directory structure for benchmarking
	tempDir, err := os.MkdirTemp("", "compress_bench")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	err = os.MkdirAll(sourceDir, 0755)
	if err != nil {
		b.Fatalf("Failed to create source directory: %v", err)
	}

	// Create multiple files for realistic benchmark
	for i := 0; i < 100; i++ {
		fileName := fmt.Sprintf("file_%d.txt", i)
		filePath := filepath.Join(sourceDir, fileName)
		content := fmt.Sprintf("This is test file number %d with some content to compress", i)
		err = os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			b.Fatalf("Failed to create benchmark file: %v", err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		archivePath := filepath.Join(tempDir, fmt.Sprintf("bench_%d.tar.gz", i))
		err := CompressDirectory(sourceDir, archivePath)
		if err != nil {
			b.Errorf("CompressDirectory failed: %v", err)
		}

		// Clean up archive for next iteration
		os.Remove(archivePath)
	}
}

func TestCompressDirectory_EdgeCases(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "compress_edge_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test with existing target file
	t.Run("Existing target file", func(t *testing.T) {
		sourceDir := filepath.Join(tempDir, "source_existing")
		err = os.MkdirAll(sourceDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create source directory: %v", err)
		}

		// Create a test file
		testFile := filepath.Join(sourceDir, "test.txt")
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		archivePath := filepath.Join(tempDir, "existing.tar.gz")

		// Create existing archive file
		err = os.WriteFile(archivePath, []byte("existing content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create existing archive: %v", err)
		}

		// Compress should overwrite existing file
		err = CompressDirectory(sourceDir, archivePath)
		if err != nil {
			t.Errorf("CompressDirectory failed with existing target: %v", err)
		}

		// Verify the file was overwritten (should be larger and be a valid archive)
		info, err := os.Stat(archivePath)
		if err != nil {
			t.Errorf("Failed to stat overwritten archive: %v", err)
		}

		if info.Size() <= int64(len("existing content")) {
			t.Error("Archive file doesn't appear to have been overwritten")
		}
	})

	// Test with zero-byte file
	t.Run("Zero-byte file", func(t *testing.T) {
		sourceDir := filepath.Join(tempDir, "source_zero")
		err = os.MkdirAll(sourceDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create source directory: %v", err)
		}

		// Create a zero-byte file
		zeroFile := filepath.Join(sourceDir, "zero.txt")
		err = os.WriteFile(zeroFile, []byte{}, 0644)
		if err != nil {
			t.Fatalf("Failed to create zero-byte file: %v", err)
		}

		archivePath := filepath.Join(tempDir, "zero.tar.gz")
		err = CompressDirectory(sourceDir, archivePath)
		if err != nil {
			t.Errorf("CompressDirectory failed with zero-byte file: %v", err)
		}

		// Verify archive contents
		testFiles := map[string]string{
			"zero.txt": "",
		}
		err = verifyArchiveContents(archivePath, testFiles)
		if err != nil {
			t.Errorf("Zero-byte file archive verification failed: %v", err)
		}
	})
}

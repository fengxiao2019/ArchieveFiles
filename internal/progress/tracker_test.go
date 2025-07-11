package progress

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewProgressTracker(t *testing.T) {
	// Test enabled tracker
	t.Run("Enabled tracker", func(t *testing.T) {
		tracker := NewProgressTracker(true)
		if !tracker.enabled {
			t.Error("Expected tracker to be enabled")
		}
		if tracker.startTime.IsZero() {
			t.Error("Expected start time to be set")
		}
	})

	// Test disabled tracker
	t.Run("Disabled tracker", func(t *testing.T) {
		tracker := NewProgressTracker(false)
		if tracker.enabled {
			t.Error("Expected tracker to be disabled")
		}
	})
}

func TestProgressTracker_Init(t *testing.T) {
	// Test enabled tracker
	t.Run("Enabled tracker", func(t *testing.T) {
		tracker := NewProgressTracker(true)
		totalItems := 10
		totalSize := int64(1024)

		tracker.Init(totalItems, totalSize)

		if tracker.totalItems != totalItems {
			t.Errorf("Expected totalItems to be %d, got %d", totalItems, tracker.totalItems)
		}
		if tracker.totalSize != totalSize {
			t.Errorf("Expected totalSize to be %d, got %d", totalSize, tracker.totalSize)
		}
		if tracker.currentItem != 0 {
			t.Errorf("Expected currentItem to be 0, got %d", tracker.currentItem)
		}
		if tracker.processedSize != 0 {
			t.Errorf("Expected processedSize to be 0, got %d", tracker.processedSize)
		}
	})

	// Test disabled tracker
	t.Run("Disabled tracker", func(t *testing.T) {
		tracker := NewProgressTracker(false)
		originalItems := tracker.totalItems
		originalSize := tracker.totalSize

		tracker.Init(100, 2048)

		// Should not change values when disabled
		if tracker.totalItems != originalItems {
			t.Error("Disabled tracker should not update totalItems")
		}
		if tracker.totalSize != originalSize {
			t.Error("Disabled tracker should not update totalSize")
		}
	})
}

func TestProgressTracker_SetCurrentFile(t *testing.T) {
	// Test enabled tracker
	t.Run("Enabled tracker", func(t *testing.T) {
		tracker := NewProgressTracker(true)
		filename := "test_file.db"

		tracker.SetCurrentFile(filename)

		if tracker.currentFile != filename {
			t.Errorf("Expected currentFile to be %s, got %s", filename, tracker.currentFile)
		}
	})

	// Test disabled tracker
	t.Run("Disabled tracker", func(t *testing.T) {
		tracker := NewProgressTracker(false)
		originalFile := tracker.currentFile

		tracker.SetCurrentFile("test_file.db")

		// Should not change when disabled
		if tracker.currentFile != originalFile {
			t.Error("Disabled tracker should not update currentFile")
		}
	})
}

func TestProgressTracker_CompleteItem(t *testing.T) {
	// Test enabled tracker
	t.Run("Enabled tracker", func(t *testing.T) {
		tracker := NewProgressTracker(true)
		tracker.Init(3, 1024)

		size1 := int64(100)
		size2 := int64(200)
		size3 := int64(300)

		// Complete first item
		tracker.CompleteItem(size1)
		if tracker.currentItem != 1 {
			t.Errorf("Expected currentItem to be 1, got %d", tracker.currentItem)
		}
		if tracker.processedSize != size1 {
			t.Errorf("Expected processedSize to be %d, got %d", size1, tracker.processedSize)
		}

		// Complete second item
		tracker.CompleteItem(size2)
		if tracker.currentItem != 2 {
			t.Errorf("Expected currentItem to be 2, got %d", tracker.currentItem)
		}
		if tracker.processedSize != size1+size2 {
			t.Errorf("Expected processedSize to be %d, got %d", size1+size2, tracker.processedSize)
		}

		// Complete third item
		tracker.CompleteItem(size3)
		if tracker.currentItem != 3 {
			t.Errorf("Expected currentItem to be 3, got %d", tracker.currentItem)
		}
		if tracker.processedSize != size1+size2+size3 {
			t.Errorf("Expected processedSize to be %d, got %d", size1+size2+size3, tracker.processedSize)
		}
	})

	// Test disabled tracker
	t.Run("Disabled tracker", func(t *testing.T) {
		tracker := NewProgressTracker(false)
		originalItem := tracker.currentItem
		originalSize := tracker.processedSize

		tracker.CompleteItem(100)

		// Should not change when disabled
		if tracker.currentItem != originalItem {
			t.Error("Disabled tracker should not update currentItem")
		}
		if tracker.processedSize != originalSize {
			t.Error("Disabled tracker should not update processedSize")
		}
	})
}

func TestProgressTracker_UpdateRocksDBProgress(t *testing.T) {
	// Test enabled tracker
	t.Run("Enabled tracker", func(t *testing.T) {
		tracker := NewProgressTracker(true)
		processed := int64(500)
		total := int64(1000)

		// This should not panic and should handle the update gracefully
		tracker.UpdateRocksDBProgress(processed, total)

		// The method should handle the progress update without errors
		// We can't easily test the display output, but we can ensure no panic
	})

	// Test disabled tracker
	t.Run("Disabled tracker", func(t *testing.T) {
		tracker := NewProgressTracker(false)

		// This should not panic and should be a no-op
		tracker.UpdateRocksDBProgress(100, 200)
	})

	// Test edge cases
	t.Run("Zero total", func(t *testing.T) {
		tracker := NewProgressTracker(true)

		// Should handle zero total gracefully
		tracker.UpdateRocksDBProgress(0, 0)
	})

	t.Run("Processed greater than total", func(t *testing.T) {
		tracker := NewProgressTracker(true)

		// Should handle edge case where processed > total
		tracker.UpdateRocksDBProgress(150, 100)
	})
}

func TestProgressTracker_Concurrency(t *testing.T) {
	// Test concurrent access to the progress tracker
	tracker := NewProgressTracker(true)
	tracker.Init(100, 10240)

	var wg sync.WaitGroup
	numGoroutines := 10
	itemsPerGoroutine := 10

	// Start multiple goroutines that update progress concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < itemsPerGoroutine; j++ {
				tracker.SetCurrentFile(fmt.Sprintf("file_%d_%d.db", id, j))
				tracker.CompleteItem(int64(j + 1))
				time.Sleep(time.Millisecond) // Small delay to increase chance of concurrency
			}
		}(i)
	}

	wg.Wait()

	// Verify final state
	expectedItems := numGoroutines * itemsPerGoroutine
	if tracker.currentItem != expectedItems {
		t.Errorf("Expected currentItem to be %d, got %d", expectedItems, tracker.currentItem)
	}

	// Verify that the processedSize is reasonable (sum of all individual sizes)
	expectedMinSize := int64(numGoroutines * (1 + itemsPerGoroutine) * itemsPerGoroutine / 2)
	if tracker.processedSize != expectedMinSize {
		t.Errorf("Expected processedSize to be %d, got %d", expectedMinSize, tracker.processedSize)
	}
}

func TestProgressTracker_RaceConditions(t *testing.T) {
	// Test for race conditions using go test -race
	tracker := NewProgressTracker(true)
	tracker.Init(10, 1024)

	var wg sync.WaitGroup

	// Start goroutines that call different methods concurrently
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			tracker.SetCurrentFile(fmt.Sprintf("file_%d.db", id))
		}(i)
	}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			tracker.CompleteItem(int64(id * 10))
		}(i)
	}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			tracker.UpdateRocksDBProgress(int64(id*100), int64(id*200))
		}(i)
	}

	wg.Wait()

	// If we get here without data races, the test passes
}

func TestProgressTracker_StateConsistency(t *testing.T) {
	tracker := NewProgressTracker(true)
	totalItems := 5
	totalSize := int64(500)

	tracker.Init(totalItems, totalSize)

	// Verify initial state
	if tracker.totalItems != totalItems {
		t.Errorf("Expected totalItems to be %d, got %d", totalItems, tracker.totalItems)
	}
	if tracker.totalSize != totalSize {
		t.Errorf("Expected totalSize to be %d, got %d", totalSize, tracker.totalSize)
	}
	if tracker.currentItem != 0 {
		t.Errorf("Expected currentItem to be 0, got %d", tracker.currentItem)
	}
	if tracker.processedSize != 0 {
		t.Errorf("Expected processedSize to be 0, got %d", tracker.processedSize)
	}

	// Update progress step by step
	for i := 1; i <= totalItems; i++ {
		tracker.SetCurrentFile(fmt.Sprintf("file_%d.db", i))
		tracker.CompleteItem(int64(i * 100))

		// Verify state after each update
		if tracker.currentItem != i {
			t.Errorf("After item %d: expected currentItem to be %d, got %d", i, i, tracker.currentItem)
		}

		expectedSize := int64(i * (i + 1) * 100 / 2)
		if tracker.processedSize != expectedSize {
			t.Errorf("After item %d: expected processedSize to be %d, got %d", i, expectedSize, tracker.processedSize)
		}
	}
}

func TestProgressTracker_EdgeCases(t *testing.T) {
	// Test initialization with zero values
	t.Run("Zero initialization", func(t *testing.T) {
		tracker := NewProgressTracker(true)
		tracker.Init(0, 0)

		if tracker.totalItems != 0 {
			t.Errorf("Expected totalItems to be 0, got %d", tracker.totalItems)
		}
		if tracker.totalSize != 0 {
			t.Errorf("Expected totalSize to be 0, got %d", tracker.totalSize)
		}
	})

	// Test with negative values
	t.Run("Negative values", func(t *testing.T) {
		tracker := NewProgressTracker(true)
		tracker.Init(-1, -100)

		// Should handle negative values gracefully
		if tracker.totalItems != -1 {
			t.Errorf("Expected totalItems to be -1, got %d", tracker.totalItems)
		}
		if tracker.totalSize != -100 {
			t.Errorf("Expected totalSize to be -100, got %d", tracker.totalSize)
		}
	})

	// Test with very large values
	t.Run("Large values", func(t *testing.T) {
		tracker := NewProgressTracker(true)
		largeItems := 1000000
		largeSize := int64(1024 * 1024 * 1024) // 1GB

		tracker.Init(largeItems, largeSize)

		if tracker.totalItems != largeItems {
			t.Errorf("Expected totalItems to be %d, got %d", largeItems, tracker.totalItems)
		}
		if tracker.totalSize != largeSize {
			t.Errorf("Expected totalSize to be %d, got %d", largeSize, tracker.totalSize)
		}
	})

	// Test empty file name
	t.Run("Empty file name", func(t *testing.T) {
		tracker := NewProgressTracker(true)
		tracker.SetCurrentFile("")

		if tracker.currentFile != "" {
			t.Errorf("Expected currentFile to be empty, got %s", tracker.currentFile)
		}
	})

	// Test very long file name
	t.Run("Long file name", func(t *testing.T) {
		tracker := NewProgressTracker(true)
		longName := string(make([]byte, 1000))
		for i := range longName {
			longName = longName[:i] + "a" + longName[i+1:]
		}

		tracker.SetCurrentFile(longName)

		if tracker.currentFile != longName {
			t.Error("Expected currentFile to handle long names")
		}
	})
}

func TestProgressTracker_TimingBehavior(t *testing.T) {
	tracker := NewProgressTracker(true)

	// Record start time
	startTime := time.Now()
	tracker.Init(1, 100)

	// The start time should be close to when we started
	if time.Since(tracker.startTime) > time.Second {
		t.Error("Start time should be recent")
	}

	// Start time should be after our recorded time
	if tracker.startTime.Before(startTime) {
		t.Error("Start time should be after test start")
	}
}

func TestProgressTracker_MultipleInit(t *testing.T) {
	tracker := NewProgressTracker(true)

	// First initialization
	tracker.Init(5, 500)
	tracker.CompleteItem(100)

	// Second initialization should reset everything
	tracker.Init(10, 1000)

	if tracker.totalItems != 10 {
		t.Errorf("Expected totalItems to be 10 after re-init, got %d", tracker.totalItems)
	}
	if tracker.totalSize != 1000 {
		t.Errorf("Expected totalSize to be 1000 after re-init, got %d", tracker.totalSize)
	}
	if tracker.currentItem != 0 {
		t.Errorf("Expected currentItem to be 0 after re-init, got %d", tracker.currentItem)
	}
	if tracker.processedSize != 0 {
		t.Errorf("Expected processedSize to be 0 after re-init, got %d", tracker.processedSize)
	}
}

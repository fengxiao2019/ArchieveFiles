package progress

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"archiveFiles/internal/constants"
	"archiveFiles/internal/utils"
)

// ProgressTracker tracks progress of backup operations
type ProgressTracker struct {
	mu            sync.Mutex
	totalItems    int
	currentItem   int
	totalSize     int64
	processedSize int64
	startTime     time.Time
	currentFile   string
	enabled       bool
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(enabled bool) *ProgressTracker {
	return &ProgressTracker{
		startTime: time.Now(),
		enabled:   enabled,
	}
}

// Init initializes progress tracking
func (p *ProgressTracker) Init(totalItems int, totalSize int64) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.totalItems = totalItems
	p.totalSize = totalSize
	p.currentItem = 0
	p.processedSize = 0
	p.startTime = time.Now()
}

// SetCurrentFile updates the current processing file
func (p *ProgressTracker) SetCurrentFile(filename string) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentFile = filename
	p.displayProgress()
}

// CompleteItem marks an item as completed
func (p *ProgressTracker) CompleteItem(size int64) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentItem++
	p.processedSize += size
	p.displayProgress()
}

// UpdateRocksDBProgress updates progress for RocksDB copying (by record count)
func (p *ProgressTracker) UpdateRocksDBProgress(processed, total int64) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	// For RocksDB, we use record count as a proxy for progress
	if total > 0 {
		p.displayRocksDBProgress(processed, total)
	}
}

// displayProgress displays overall progress
func (p *ProgressTracker) displayProgress() {
	if p.totalItems == 0 {
		return
	}

	percentage := float64(p.currentItem) / float64(p.totalItems) * 100
	elapsed := time.Since(p.startTime)

	// Calculate speed and ETA
	var eta time.Duration
	var speed string
	if p.processedSize > 0 && elapsed.Seconds() > 0 {
		bytesPerSecond := float64(p.processedSize) / elapsed.Seconds()
		speed = utils.FormatBytes(int64(bytesPerSecond)) + "/s"

		if bytesPerSecond > 0 {
			remainingBytes := p.totalSize - p.processedSize
			etaSeconds := float64(remainingBytes) / bytesPerSecond
			eta = time.Duration(etaSeconds) * time.Second
		}
	}

	// Create progress bar
	filled := int(percentage / 100 * float64(constants.ProgressBarWidth))
	if filled < 0 {
		filled = 0
	}
	if filled > constants.ProgressBarWidth {
		filled = constants.ProgressBarWidth
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", constants.ProgressBarWidth-filled)

	// Format output
	fmt.Printf("\r[%s] %.1f%% (%d/%d) | %s | %s",
		bar,
		percentage,
		p.currentItem,
		p.totalItems,
		utils.FormatBytes(p.processedSize)+"/"+utils.FormatBytes(p.totalSize),
		speed,
	)

	if eta > 0 {
		fmt.Printf(" | ETA: %s", utils.FormatDuration(eta))
	}

	if p.currentFile != "" {
		fmt.Printf(" | %s", utils.TruncateString(p.currentFile, constants.ProgressFileNameMaxLength))
	}

	fmt.Print("   ") // Clear any remaining characters
}

// displayRocksDBProgress displays RocksDB specific progress
func (p *ProgressTracker) displayRocksDBProgress(processed, total int64) {
	percentage := float64(processed) / float64(total) * 100
	elapsed := time.Since(p.startTime)

	// Calculate records per second
	var speed string
	if elapsed.Seconds() > 0 {
		recordsPerSecond := float64(processed) / elapsed.Seconds()
		speed = fmt.Sprintf("%.0f rec/s", recordsPerSecond)
	}

	// Create progress bar
	filled := int(percentage / 100 * float64(constants.ProgressBarWidth))
	if filled < 0 {
		filled = 0
	}
	if filled > constants.ProgressBarWidth {
		filled = constants.ProgressBarWidth
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", constants.ProgressBarWidth-filled)

	fmt.Printf("\r  [%s] %.1f%% (%d/%d records) | %s | %s   ",
		bar,
		percentage,
		processed,
		total,
		speed,
		p.currentFile,
	)
}

// Finish completes progress tracking
func (p *ProgressTracker) Finish() {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	elapsed := time.Since(p.startTime)
	fmt.Printf("\nCompleted %d item(s) in %s (%s total)\n",
		p.totalItems,
		utils.FormatDuration(elapsed),
		utils.FormatBytes(p.totalSize))
}

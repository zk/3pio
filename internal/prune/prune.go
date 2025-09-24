package prune

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/zk/3pio/internal/logger"
)

type Pruner struct {
	logger  *logger.FileLogger
	dryRun  bool
	force   bool
	baseDir string
}

type RunInfo struct {
	Path      string
	Timestamp time.Time
	Size      int64
}

func New(logger *logger.FileLogger, dryRun, force bool) *Pruner {
	return &Pruner{
		logger:  logger,
		dryRun:  dryRun,
		force:   force,
		baseDir: ".3pio",
	}
}

func (p *Pruner) Run() error {
	p.logger.Info("Starting prune operation (dry-run: %v, force: %v)", p.dryRun, p.force)

	// Get all runs
	runs, err := p.getRuns()
	if err != nil {
		return fmt.Errorf("failed to get runs: %w", err)
	}

	if len(runs) == 0 {
		fmt.Println("No test runs found - nothing to prune")
		return nil
	}

	// Sort runs by timestamp (newest first)
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].Timestamp.After(runs[j].Timestamp)
	})

	// Identify runs to delete (all except the most recent)
	var toDelete []RunInfo
	if len(runs) > 1 {
		toDelete = runs[1:]
	}

	// Calculate total size
	var totalSize int64
	for _, run := range toDelete {
		totalSize += run.Size
	}

	// Get debug log size
	debugLogPath := filepath.Join(p.baseDir, "debug.log")
	var debugLogSize int64
	if info, err := os.Stat(debugLogPath); err == nil {
		debugLogSize = info.Size()
	}

	// Display summary
	fmt.Println("\n3pio Prune Summary:")
	fmt.Println("===================")
	fmt.Printf("Total runs found: %d\n", len(runs))
	if len(runs) > 0 {
		fmt.Printf("Most recent run: %s\n", filepath.Base(runs[0].Path))
	}
	fmt.Printf("Runs to delete: %d\n", len(toDelete))
	fmt.Printf("Space to be freed: %s\n", formatSize(totalSize+debugLogSize))

	if p.dryRun {
		fmt.Println("\n[DRY RUN MODE - No files will be deleted]")
		if len(toDelete) > 0 {
			fmt.Println("\nRuns that would be deleted:")
			for _, run := range toDelete {
				fmt.Printf("  - %s (%s)\n", filepath.Base(run.Path), formatSize(run.Size))
			}
		}
		return nil
	}

	// If nothing to delete, exit early
	if len(toDelete) == 0 && debugLogSize == 0 {
		fmt.Println("\nNothing to prune - keeping the single existing run")
		return nil
	}

	// Confirm with user unless force flag is set
	if !p.force {
		fmt.Print("\nProceed with deletion? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Prune cancelled")
			return nil
		}
	}

	// Perform deletion
	fmt.Println("\nDeleting old runs...")
	deletedCount := 0
	for _, run := range toDelete {
		p.logger.Info("Deleting run: %s", run.Path)
		if err := os.RemoveAll(run.Path); err != nil {
			p.logger.Warn("Failed to delete run %s: %v", run.Path, err)
			fmt.Fprintf(os.Stderr, "Warning: Failed to delete %s: %v\n", filepath.Base(run.Path), err)
		} else {
			deletedCount++
		}
	}
	fmt.Printf("Deleted %d/%d runs\n", deletedCount, len(toDelete))

	// Clear debug log
	if debugLogSize > 0 {
		fmt.Println("Clearing debug log...")
		p.logger.Info("Clearing debug log file")
		// Close the current logger before clearing
		p.logger.Close()

		// Clear the file by truncating it
		if err := os.Truncate(debugLogPath, 0); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to clear debug log: %v\n", err)
		} else {
			fmt.Printf("Cleared debug log (%s freed)\n", formatSize(debugLogSize))
		}

		// Reopen logger
		newLogger, err := logger.NewFileLogger()
		if err == nil {
			p.logger = newLogger
			p.logger.Info("Debug log cleared and reopened after prune")
		}
	}

	fmt.Println("\nPrune completed successfully!")
	return nil
}

func (p *Pruner) getRuns() ([]RunInfo, error) {
	runsDir := filepath.Join(p.baseDir, "runs")
	if _, err := os.Stat(runsDir); os.IsNotExist(err) {
		return []RunInfo{}, nil
	}

	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return nil, err
	}

	var runs []RunInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		runPath := filepath.Join(runsDir, entry.Name())
		timestamp := p.parseTimestamp(entry.Name())
		size := p.getDirSize(runPath)

		runs = append(runs, RunInfo{
			Path:      runPath,
			Timestamp: timestamp,
			Size:      size,
		})
	}

	return runs, nil
}

func (p *Pruner) parseTimestamp(dirname string) time.Time {
	// Format: YYYYMMDD_HHMMSS_memorable-name
	parts := strings.Split(dirname, "_")
	if len(parts) < 2 {
		return time.Time{}
	}

	dateStr := parts[0]
	timeStr := parts[1]

	// Parse the timestamp
	layout := "20060102150405"
	timestamp, err := time.Parse(layout, dateStr+timeStr)
	if err != nil {
		p.logger.Warn("Failed to parse timestamp from %s: %v", dirname, err)
		return time.Time{}
	}

	return timestamp
}

func (p *Pruner) getDirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

func formatSize(bytes int64) string {
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

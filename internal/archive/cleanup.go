// internal/archive/cleanup.go
package archive

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

func CleanOldArchives(dir string, retention time.Duration, logger *log.Logger) (int, error) {
	pattern := filepath.Join(dir, "*.zip")
	files, err := filepath.Glob(pattern)
	if err != nil {
		logger.Printf("archive cleanup error: %v", err)
		return 0, err
	}

	cutoff := time.Now().Add(-retention)
	cleaned := 0

	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(f); err != nil {
				logger.Printf("failed to remove archive %s: %v", f, err)
			} else {
				cleaned++
			}
		}
	}

	if cleaned > 0 {
		logger.Printf("cleaned up %d old archives", cleaned)
	}
	return cleaned, nil
}

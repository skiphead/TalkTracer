package bot

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// CleanupOldTempFiles удаляет устаревшие временные файлы, пропуская активные.
func (b *Bot) CleanupOldTempFiles(maxAge time.Duration) error {
	logger := b.logger.With(slog.String("operation", "cleanup"))

	entries, err := os.ReadDir(b.tempDir)
	if err != nil {
		return fmt.Errorf("failed to read temp dir: %w", err)
	}

	stats := b.processCleanupEntries(entries, maxAge, logger)
	logger.Info("Cleanup completed",
		slog.Int("deleted", stats.deleted),
		slog.Int("skipped", stats.skipped),
	)
	return nil
}

// cleanupStats хранит статистику операции очистки.
type cleanupStats struct {
	deleted int
	skipped int
}

// processCleanupEntries обрабатывает список файлов и возвращает статистику.
func (b *Bot) processCleanupEntries(entries []os.DirEntry, maxAge time.Duration, logger *slog.Logger) cleanupStats {
	stats := cleanupStats{}
	now := time.Now()

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		b.processSingleFile(filepath.Join(b.tempDir, entry.Name()), now, maxAge, logger, &stats)
	}
	return stats
}

// processSingleFile обрабатывает один файл: проверяет активность, возраст и удаляет при необходимости.
func (b *Bot) processSingleFile(filePath string, now time.Time, maxAge time.Duration, logger *slog.Logger, stats *cleanupStats) {
	if b.isFileActive(filePath) {
		logger.Debug("Skipping active file", slog.String("path", filePath))
		stats.skipped++
		return
	}

	info, err := os.Stat(filePath)
	if err != nil {
		logger.Warn("Failed to get file info",
			slog.String("path", filePath),
			slog.String("error", err.Error()),
		)
		return
	}

	if now.Sub(info.ModTime()) > maxAge {
		b.tryRemoveOldFile(filePath, info, now, logger, stats)
	}
}

// tryRemoveOldFile пытается удалить устаревший файл и обновляет статистику.
func (b *Bot) tryRemoveOldFile(filePath string, info os.FileInfo, now time.Time, logger *slog.Logger, stats *cleanupStats) {
	if err := os.Remove(filePath); err != nil {
		if !os.IsNotExist(err) {
			logger.Debug("File is in use or cannot be removed",
				slog.String("path", filePath),
				slog.String("error", err.Error()),
			)
			stats.skipped++
		}
	} else {
		logger.Info("Removed old temp file",
			slog.String("path", filePath),
			slog.Duration("age", now.Sub(info.ModTime())),
		)
		stats.deleted++
	}
}

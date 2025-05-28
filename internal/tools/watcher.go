package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dudykr/syncai/internal/config"
	"github.com/dudykr/syncai/internal/types"
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

// Watcher handles file system watching for cursor rules changes
type Watcher struct {
	rootDir   string
	converter *Converter
	parser    *config.Parser
	targets   []types.TargetTool
	logger    *logrus.Logger
	watcher   *fsnotify.Watcher
}

// NewWatcher creates a new file system watcher
func NewWatcher(rootDir string, converter *Converter, parser *config.Parser, targets []types.TargetTool, logger *logrus.Logger) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		rootDir:   rootDir,
		converter: converter,
		parser:    parser,
		targets:   targets,
		logger:    logger,
		watcher:   watcher,
	}, nil
}

// Start starts watching for file changes
func (w *Watcher) Start(ctx context.Context) error {
	// Add paths to watch
	if err := w.addWatchPaths(); err != nil {
		return err
	}

	w.logger.Info("Starting file system watcher...")

	// Debounce timer to avoid multiple rapid rebuilds
	var debounceTimer *time.Timer
	const debounceDelay = 500 * time.Millisecond

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}

			if w.shouldProcessEvent(event) {
				w.logger.Debugf("File change detected: %s (%s)", event.Name, event.Op)

				// Reset or create debounce timer
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(debounceDelay, func() {
					if err := w.rebuild(); err != nil {
						w.logger.Errorf("Failed to rebuild: %v", err)
					}
				})
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return nil
			}
			w.logger.Errorf("Watcher error: %v", err)

		case <-ctx.Done():
			w.logger.Info("Stopping file system watcher...")
			return w.watcher.Close()
		}
	}
}

// addWatchPaths adds all relevant paths to the watcher
func (w *Watcher) addWatchPaths() error {
	// Watch the root directory for .cursorrules
	if err := w.watcher.Add(w.rootDir); err != nil {
		return err
	}

	// Walk the directory tree to find .cursor directories
	return filepath.WalkDir(w.rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip directories we can't access
		}

		if d.IsDir() {
			// Watch .cursor directories and their rules subdirectories
			if d.Name() == ".cursor" {
				if err := w.watcher.Add(path); err != nil {
					w.logger.Warnf("Failed to watch %s: %v", path, err)
				}

				rulesDir := filepath.Join(path, "rules")
				if err := w.watcher.Add(rulesDir); err != nil {
					w.logger.Debugf("Rules directory doesn't exist or can't be watched: %s", rulesDir)
				}
			}
		}

		return nil
	})
}

// shouldProcessEvent determines if an event should trigger a rebuild
func (w *Watcher) shouldProcessEvent(event fsnotify.Event) bool {
	// Only process write and create events
	if event.Op&fsnotify.Write == 0 && event.Op&fsnotify.Create == 0 {
		return false
	}

	fileName := filepath.Base(event.Name)

	// Check for .cursorrules file
	if fileName == ".cursorrules" {
		return true
	}

	// Check for files in .cursor/rules directories
	if strings.Contains(event.Name, ".cursor/rules/") {
		return true
	}

	// Check for .mdc files
	if strings.HasSuffix(fileName, ".mdc") {
		return true
	}

	return false
}

// rebuild parses rules and converts them to target formats
func (w *Watcher) rebuild() error {
	w.logger.Info("Rebuilding target configurations...")

	start := time.Now()

	// Parse cursor rules
	rules, err := w.parser.ParseCursorRules()
	if err != nil {
		return err
	}

	// Convert to target formats
	if err := w.converter.ConvertRules(rules, w.targets); err != nil {
		return err
	}

	duration := time.Since(start)
	w.logger.Infof("Rebuild completed in %v", duration)

	return nil
}

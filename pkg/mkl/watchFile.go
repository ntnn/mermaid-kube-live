package mkl

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// watchFile watches a file for changes and calls the provided function
// whenever a change occurs.
func (m *MKL) watchFile(ctx context.Context, filePath string, fn func() error) error { //nolint:cyclop
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Watching the directory instead of the file itself to handle cases
	// where the file is replaced (e.g., by an editor).
	parentDir := filepath.Dir(filePath)
	if err := watcher.Add(parentDir); err != nil {
		return fmt.Errorf("failed to watch parent directory %q of %q: %w", parentDir, filePath, err)
	}

	if fn == nil {
		return errors.New("file hook function cannot be nil")
	}

	if err := fn(); err != nil {
		return fmt.Errorf("initial file hook run failed %q: %w", filePath, err)
	}

	go func() {
		defer func() {
			if err := watcher.Close(); err != nil {
				m.opts.Logger.Error(err, "failed to close watcher")
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case e, ok := <-watcher.Events:
				if !ok {
					return
				}
				if filepath.Clean(e.Name) != filepath.Clean(filePath) {
					continue
				}
				m.opts.Logger.V(2).Info("file event", "event", e, "file", filePath)

				if err := fn(); err != nil {
					m.opts.Logger.Error(err, "file hook function failed", "file", filePath)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				m.opts.Logger.Error(err, "error watching file", "file", filePath)
			}
		}
	}()

	return nil
}

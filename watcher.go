package lugo

import (
	"context"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

// WatcherConfig contains settings for the configuration watcher
type WatcherConfig struct {
	// Paths to watch for changes
	Paths []string
	// How often to poll for changes (for non-inotify systems)
	PollInterval time.Duration
	// Debounce period to avoid multiple reloads
	DebounceInterval time.Duration
	// Callback function when configuration is reloaded
	OnReload func(error)
}

// ConfigWatcher watches for configuration changes and reloads automatically
type ConfigWatcher struct {
	cfg      *Config
	watcher  *fsnotify.Watcher
	paths    map[string]bool
	stopChan chan struct{}
	mu       sync.RWMutex
	wcfg     WatcherConfig
}

// NewWatcher creates a new configuration watcher
func (c *Config) NewWatcher(wcfg WatcherConfig) (*ConfigWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if wcfg.PollInterval == 0 {
		wcfg.PollInterval = 5 * time.Second
	}
	if wcfg.DebounceInterval == 0 {
		wcfg.DebounceInterval = 100 * time.Millisecond
	}

	w := &ConfigWatcher{
		cfg:      c,
		watcher:  watcher,
		paths:    make(map[string]bool),
		stopChan: make(chan struct{}),
		wcfg:     wcfg,
	}

	for _, path := range wcfg.Paths {
		if err := w.AddPath(path); err != nil {
			w.Close()
			return nil, err
		}
	}

	go w.watch()
	return w, nil
}

// AddPath adds a path to watch
func (w *ConfigWatcher) AddPath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.paths[absPath] {
		return nil
	}

	if err := w.watcher.Add(absPath); err != nil {
		return err
	}

	w.paths[absPath] = true
	return nil
}

// RemovePath removes a path from watching
func (w *ConfigWatcher) RemovePath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.paths[absPath] {
		return nil
	}

	if err := w.watcher.Remove(absPath); err != nil {
		return err
	}

	delete(w.paths, absPath)
	return nil
}

// Close stops watching for changes
func (w *ConfigWatcher) Close() error {
	close(w.stopChan)
	return w.watcher.Close()
}

func (w *ConfigWatcher) watch() {
	var (
		debounceTimer *time.Timer
		reloadMu      sync.Mutex
	)

	reload := func() {
		reloadMu.Lock()
		defer reloadMu.Unlock()

		w.mu.RLock()
		paths := make([]string, 0, len(w.paths))
		for path := range w.paths {
			paths = append(paths, path)
		}
		w.mu.RUnlock()

		var err error
		for _, path := range paths {
			if err = w.cfg.LoadFile(context.Background(), path); err != nil {
				w.cfg.logger.Error("failed to reload config",
					zap.String("path", path),
					zap.Error(err))
				break
			}
		}

		if w.wcfg.OnReload != nil {
			w.wcfg.OnReload(err)
		}
	}

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(w.wcfg.DebounceInterval, reload)
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.cfg.logger.Error("watcher error", zap.Error(err))

		case <-w.stopChan:
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return
		}
	}
}

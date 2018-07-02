package conf

import (
	"errors"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

type watchingInfo struct {
	BindAddr string
	Updater
}
type Updater interface {
	ReplaceFrontend(*Frontend) error
	RemoveFrontend(string)
}

// ConfigWatcher struct
type ConfigWatcher struct {
	// dir     string
	watcher  *fsnotify.Watcher
	watching map[string]watchingInfo
}

// NewConfigWatcher creates new file watcher for config files
func NewConfigWatcher() (*ConfigWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &ConfigWatcher{
		watcher:  watcher,
		watching: make(map[string]watchingInfo),
	}, nil
}

// Add starts watching the named file or directory (non-recursively).
func (cw *ConfigWatcher) Add(dir string, bindAddr string, updater Updater) error {
	// ensure directory has trailing foreward slash
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	if _, ok := cw.watching[dir]; ok {
		return errors.New("Already watching dir")
	}
	cw.watching[dir] = watchingInfo{
		BindAddr: bindAddr,
		Updater:  updater,
	}
	zap.L().Debug("Watching",
		zap.String("dir", dir),
		zap.Any("watching", cw.watching),
	)
	return cw.watcher.Add(dir)
}

// Start config directory watching
func (cw *ConfigWatcher) Start() {
	cw.updateAll()

	go func() {
		defer cw.watcher.Close()
		for {
			select {
			case event, ok := <-cw.watcher.Events:
				if !ok {
					return
				}
				zap.L().Debug("Watcher event", zap.Any("event", event))
				if event.Op&fsnotify.Write == fsnotify.Write {
					time.Sleep(time.Millisecond) // ???? EOF if we don't wait
					cw.updateFrontend(event.Name, false)
				} else if event.Op&fsnotify.Create == fsnotify.Create {
					time.Sleep(time.Millisecond) // ???? EOF if we don't wait
					cw.updateFrontend(event.Name, false)
				} else if event.Op&fsnotify.Rename == fsnotify.Rename {
					cw.updateFrontend(event.Name, true)
				}
			case err := <-cw.watcher.Errors:
				zap.L().Error("Config watcher", zap.Error(err))
			}
		}
	}()
}
func (cw *ConfigWatcher) updateAll() {
	for dir := range cw.watching {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			zap.L().Warn("Failed to read files in directory",
				zap.String("dir", dir),
				zap.Error(err),
			)
			continue
		}

		for _, f := range files {
			if f.IsDir() {
				continue
			}
			cw.updateFrontend(path.Join(dir, f.Name()), false)
		}
	}
}

func (cw *ConfigWatcher) updateFrontend(filePath string, delete bool) {
	dir, file := filepath.Split(filePath)
	name := strings.TrimSuffix(file, path.Ext(file))
	zap.L().Debug("Updating Frontend",
		zap.String("dir", dir),
		// zap.String("file", file),
		zap.String("name", name),
	)

	info, ok := cw.watching[dir]
	if !ok {
		zap.L().Warn("No watching info found",
			zap.String("dir", dir),
			zap.Any("watching", cw.watching),
		)
		return
	}

	if delete {
		info.RemoveFrontend(name)
		return
	}

	frontend := NewFrontend(info.BindAddr, name, nil)
	err := frontend.ParseFile(filePath)
	if err != nil {
		zap.L().Error("Failed to read config",
			zap.String("name", filePath),
			zap.Error(err),
		)
		return
	}
	if err := info.ReplaceFrontend(frontend); err != nil {
		zap.L().Error("Failed to replace frontend",
			zap.String("name", filePath),
			zap.Error(err),
		)
		return
	}
	zap.L().Info("New frontend", zap.Any("frontend", frontend))
}

// Stop config directory watching
func (cw *ConfigWatcher) Stop() error {
	return cw.watcher.Close()
}

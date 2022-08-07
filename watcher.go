package gaper

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	zglob "github.com/mattn/go-zglob"
)

// Watcher is a interface for the watch process
type Watcher interface {
	Watch()
	Errors() chan error
	Events() chan string
}

// watcher is a interface for the watch process
type watcher struct {
	defaultIgnore     bool
	pollInterval      int
	watchItems        map[string]bool
	ignoreItems       map[string]bool
	allowedExtensions map[string]bool
	events            chan string
	errors            chan error
}

// WatcherConfig defines the settings available for the watcher
type WatcherConfig struct {
	DefaultIgnore bool
	PollInterval  int
	WatchItems    []string
	IgnoreItems   []string
	Extensions    []string
}

// NewWatcher creates a new watcher
func NewWatcher(cfg WatcherConfig) (Watcher, error) {
	if cfg.PollInterval == 0 {
		cfg.PollInterval = DefaultPoolInterval
	}

	if len(cfg.Extensions) == 0 {
		cfg.Extensions = DefaultExtensions
	}

	allowedExts := make(map[string]bool)
	for _, ext := range cfg.Extensions {
		allowedExts["."+ext] = true
	}

	watchPaths, err := resolvePaths(cfg.WatchItems, allowedExts)
	if err != nil {
		return nil, err
	}

	ignorePaths, err := resolvePaths(cfg.IgnoreItems, allowedExts)
	if err != nil {
		return nil, err
	}

	logger.Debugf("Resolved watch paths: %v", watchPaths)
	logger.Debugf("Resolved ignore paths: %v", ignorePaths)
	return &watcher{
		events:            make(chan string),
		errors:            make(chan error),
		defaultIgnore:     cfg.DefaultIgnore,
		pollInterval:      cfg.PollInterval,
		watchItems:        watchPaths,
		ignoreItems:       ignorePaths,
		allowedExtensions: allowedExts,
	}, nil
}

var startTime = time.Now()
var errDetectedChange = errors.New("done")

// Watch starts watching for file changes
func (w *watcher) Watch() {
	for {
		for watchPath := range w.watchItems {
			fileChanged, err := w.scanChange(watchPath)
			if err != nil {
				w.errors <- err
				return
			}

			if fileChanged != "" {
				w.events <- fileChanged
				startTime = time.Now()
			}
		}

		time.Sleep(time.Duration(w.pollInterval) * time.Millisecond)
	}
}

// Events get events occurred during the watching
// these events are emitted only a file changing is detected
func (w *watcher) Events() chan string {
	return w.events
}

// Errors get errors occurred during the watching
func (w *watcher) Errors() chan error {
	return w.errors
}

func (w *watcher) scanChange(watchPath string) (string, error) {
	logger.Debug("Watching ", watchPath)

	var fileChanged string

	err := filepath.Walk(watchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Ignore attempt to acess go temporary unmask
			if strings.Contains(err.Error(), "-go-tmp-umask") {
				return filepath.SkipDir
			}

			return fmt.Errorf("couldn't walk to path \"%s\": %v", path, err)
		}

		if w.ignoreFile(path, info) {
			return skipFile(info)
		}

		ext := filepath.Ext(path)
		if _, ok := w.allowedExtensions[ext]; ok && info.ModTime().After(startTime) {
			fileChanged = path
			return errDetectedChange
		}

		return nil
	})

	if err != nil && err != errDetectedChange {
		return "", err
	}

	return fileChanged, nil
}

func (w *watcher) ignoreFile(path string, info os.FileInfo) bool {
	// if a file has been deleted after gaper was watching it
	// info will be nil in the other iterations
	if info == nil {
		return true
	}

	// check if preset ignore is enabled
	if w.defaultIgnore {
		// check for hidden files and directories
		if name := info.Name(); name[0] == '.' && name != "." {
			return true
		}

		// check if it is a Go testing file
		if strings.HasSuffix(path, "_test.go") {
			return true
		}

		// check if it is the vendor folder
		if info.IsDir() && info.Name() == "vendor" {
			return true
		}
	}

	if _, ignored := w.ignoreItems[path]; ignored {
		return true
	}

	return false
}

func resolvePaths(paths []string, extensions map[string]bool) (map[string]bool, error) {
	result := map[string]bool{}

	for _, path := range paths {
		matches := []string{path}

		isGlob := strings.Contains(path, "*")
		if isGlob {
			var err error
			matches, err = zglob.Glob(path)
			if err != nil {
				return nil, fmt.Errorf("couldn't resolve glob path \"%s\": %v", path, err)
			}
		}

		for _, match := range matches {
			// ignore existing files that don't match the allowed extensions
			if f, err := os.Stat(match); !os.IsNotExist(err) && !f.IsDir() {
				if ext := filepath.Ext(match); ext != "" {
					if _, ok := extensions[ext]; !ok {
						continue
					}
				}
			}

			if _, ok := result[match]; !ok {
				result[match] = true
			}
		}
	}

	removeOverlappedPaths(result)

	return result, nil
}

// remove overlapped paths so it makes the scan for changes later faster and simpler
func removeOverlappedPaths(mapPaths map[string]bool) {
	startDot := regexp.MustCompile(`^\./`)

	for p1 := range mapPaths {
		p1 = startDot.ReplaceAllString(p1, "")

		// skip to next item if this path has already been checked
		if v, ok := mapPaths[p1]; ok && !v {
			continue
		}

		for p2 := range mapPaths {
			p2 = startDot.ReplaceAllString(p2, "")

			if p1 == p2 {
				continue
			}

			if strings.HasPrefix(p2, p1) {
				mapPaths[p2] = false
			} else if strings.HasPrefix(p1, p2) {
				mapPaths[p1] = false
			}
		}
	}

	// cleanup path list
	for p := range mapPaths {
		if !mapPaths[p] {
			delete(mapPaths, p)
		}
	}
}

func skipFile(info os.FileInfo) error {
	if info.IsDir() {
		return filepath.SkipDir
	}
	return nil
}

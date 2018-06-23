package gaper

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	zglob "github.com/mattn/go-zglob"
)

// Watcher is a interface for the watch process
type Watcher struct {
	PollInterval      int
	WatchItems        map[string]bool
	IgnoreItems       map[string]bool
	AllowedExtensions map[string]bool
	Events            chan string
	Errors            chan error
}

// NewWatcher creates a new watcher
func NewWatcher(pollInterval int, watchItems []string, ignoreItems []string, extensions []string) (*Watcher, error) {
	if pollInterval == 0 {
		pollInterval = DefaultPoolInterval
	}

	if len(extensions) == 0 {
		extensions = DefaultExtensions
	}

	allowedExts := make(map[string]bool)
	for _, ext := range extensions {
		allowedExts["."+ext] = true
	}

	watchPaths, err := resolvePaths(watchItems, allowedExts)
	if err != nil {
		return nil, err
	}

	ignorePaths, err := resolvePaths(ignoreItems, allowedExts)
	if err != nil {
		return nil, err
	}

	logger.Debugf("Resolved watch paths: %v", watchPaths)
	logger.Debugf("Resolved ignore paths: %v", ignorePaths)
	return &Watcher{
		Events:            make(chan string),
		Errors:            make(chan error),
		PollInterval:      pollInterval,
		WatchItems:        watchPaths,
		IgnoreItems:       ignorePaths,
		AllowedExtensions: allowedExts,
	}, nil
}

var startTime = time.Now()
var errDetectedChange = errors.New("done")

// Watch starts watching for file changes
func (w *Watcher) Watch() {
	for {
		for watchPath := range w.WatchItems {
			fileChanged, err := w.scanChange(watchPath)
			if err != nil {
				w.Errors <- err
				return
			}

			if fileChanged != "" {
				w.Events <- fileChanged
				startTime = time.Now()
			}
		}

		time.Sleep(time.Duration(w.PollInterval) * time.Millisecond)
	}
}

func (w *Watcher) scanChange(watchPath string) (string, error) {
	logger.Debug("Watching ", watchPath)

	var fileChanged string

	err := filepath.Walk(watchPath, func(path string, info os.FileInfo, err error) error {
		// always ignore hidden files and directories
		if filepath.Base(path)[0] == '.' {
			return nil
		}

		if _, ignored := w.IgnoreItems[path]; ignored {
			return filepath.SkipDir
		}

		ext := filepath.Ext(path)
		if _, ok := w.AllowedExtensions[ext]; ok && info.ModTime().After(startTime) {
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
			// don't care for extension filter right now for non glob paths
			// since they could be a directory
			if isGlob {
				if _, ok := extensions[filepath.Ext(path)]; !ok {
					continue
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
	for p1 := range mapPaths {
		for p2 := range mapPaths {
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

package main

import (
	"errors"
	"os"
	"path/filepath"
	"time"
)

// Watcher ...
type Watcher struct {
	PollInterval      int
	WatchItems        []string
	IgnoreItems       []string
	AllowedExtensions map[string]bool
	Events            chan string
	Errors            chan error
}

// NewWatcher ...
func NewWatcher(pollInterval int, watchItems []string, ignoreItems []string, extensions []string) *Watcher {
	allowedExts := make(map[string]bool)
	for _, ext := range extensions {
		allowedExts["."+ext] = true
	}

	return &Watcher{
		Events:            make(chan string),
		Errors:            make(chan error),
		PollInterval:      pollInterval,
		WatchItems:        watchItems,
		IgnoreItems:       ignoreItems,
		AllowedExtensions: allowedExts,
	}
}

var startTime = time.Now()
var errDetectedChange = errors.New("done")

// Watch ...
func (w *Watcher) Watch() { // nolint: gocyclo
	for {
		for i := range w.WatchItems {
			fileChanged, err := w.scanChange(w.WatchItems[i])
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
		if path == ".git" && info.IsDir() {
			return filepath.SkipDir
		}

		for _, x := range w.IgnoreItems {
			if x == path {
				return filepath.SkipDir
			}
		}

		// ignore hidden files
		if filepath.Base(path)[0] == '.' {
			return nil
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

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	zglob "github.com/mattn/go-zglob"
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
func NewWatcher(pollInterval int, watchItems []string, ignoreItems []string, extensions []string) (*Watcher, error) {
	allowedExts := make(map[string]bool)
	for _, ext := range extensions {
		allowedExts["."+ext] = true
	}

	watchMatches, err := resolveGlobMatches(watchItems)
	if err != nil {
		return nil, err
	}

	ignoreMatches, err := resolveGlobMatches(ignoreItems)
	if err != nil {
		return nil, err
	}

	return &Watcher{
		Events:            make(chan string),
		Errors:            make(chan error),
		PollInterval:      pollInterval,
		WatchItems:        watchMatches,
		IgnoreItems:       ignoreMatches,
		AllowedExtensions: allowedExts,
	}, nil
}

var startTime = time.Now()
var errDetectedChange = errors.New("done")

// Watch ...
func (w *Watcher) Watch() {
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

func resolveGlobMatches(paths []string) ([]string, error) {
	var result []string

	for _, path := range paths {
		matches, err := zglob.Glob(path)
		if err != nil {
			return nil, fmt.Errorf("couldn't resolve glob path %s: %v", path, err)
		}

		logger.Debugf("Resolved glob path %s: %v", path, matches)
		result = append(result, matches...)
	}

	return result, nil
}

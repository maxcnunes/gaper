package main

import (
	"errors"
	"os"
	"path/filepath"
	"time"
)

// Watcher ...
type Watcher struct {
	WatchItems        []string
	IgnoreItems       []string
	AllowedExtensions map[string]bool
	Events            chan string
	Errors            chan error
}

// NewWatcher ...
func NewWatcher(watchItems []string, ignoreItems []string, extensions []string) *Watcher {
	allowedExts := make(map[string]bool)
	for _, ext := range extensions {
		allowedExts["."+ext] = true
	}

	return &Watcher{
		Events:            make(chan string),
		Errors:            make(chan error),
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
		err := filepath.Walk(w.WatchItems[0], func(path string, info os.FileInfo, err error) error {
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
				w.Events <- path
				startTime = time.Now()
				return errDetectedChange
			}

			return nil
		})

		if err != nil && err != errDetectedChange {
			w.Errors <- err
		}

		time.Sleep(500 * time.Millisecond)
	}
}

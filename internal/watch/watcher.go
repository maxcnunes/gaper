// Package watch provides a mechanism for watching file(s) for changes.
// This package is adapted from https://github.com/gohugoio/hugo/blob/master/watcher Apache-2.0 License.
package watch

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	zglob "github.com/mattn/go-zglob"

	"github.com/maxcnunes/gaper/v2/internal/log"
	"github.com/maxcnunes/gaper/v2/internal/watch/fsmonitor"
)

// Time to gather changes and handle them in batches
const intervalBatcher = 500 * time.Millisecond

// Watcher is a interface for the watch process
type Watcher interface {
	Watch()
	Errors() chan error
	Events() chan []fsnotify.Event
}

// WatcherConfig defines the settings available for the watcher
type WatcherConfig struct {
	DefaultIgnore bool
	Poll          bool
	PollInterval  time.Duration
	WatchItems    []string
	IgnoreItems   []string
	Extensions    []string
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

// watcher batches file watch events in a given interval.
type watcher struct {
	fsmonitor.FileWatcher
	ticker *time.Ticker
	done   chan struct{}
	errors chan error

	events chan []fsnotify.Event // Events are returned on this channel

	defaultIgnore     bool
	watchItems        map[string]bool
	ignoreItems       map[string]bool
	allowedExtensions map[string]bool
}

// NewWatcher creates and starts a watcher with the given time interval.
// It will fall back to a poll based watcher if native isn's supported.
// To always use polling, set poll to true.
func NewWatcher(cfg WatcherConfig) (Watcher, error) {
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

	log.Logger.Debugf("Resolved watch paths: %v", watchPaths)
	log.Logger.Debugf("Resolved ignore paths: %v", ignorePaths)

	var fw fsmonitor.FileWatcher

	if cfg.Poll {
		fw = fsmonitor.NewPollingWatcher(cfg.PollInterval)
	} else {
		fw, err = fsmonitor.NewEventWatcherWithPollFallback(cfg.PollInterval)
	}

	if err != nil {
		return nil, err
	}

	w := &watcher{
		defaultIgnore:     cfg.DefaultIgnore,
		watchItems:        watchPaths,
		ignoreItems:       ignorePaths,
		allowedExtensions: allowedExts,

		FileWatcher: fw,
		ticker:      time.NewTicker(intervalBatcher),
		done:        make(chan struct{}, 1),
		events:      make(chan []fsnotify.Event, 1),
	}

	for fpath := range watchPaths {
		log.Logger.Debug("Add file", fpath)
		if err := w.Add(fpath); err != nil {
			log.Logger.Error("Error adding path ", fpath)
		}
	}

	return w, nil
}

// Watch starts watching for file changes
func (w *watcher) Watch() {
	evs := make([]fsnotify.Event, 0)
OuterLoop:
	for {
		select {
		case ev := <-w.FileWatcher.Events():
			evs = append(evs, ev)
		case <-w.ticker.C:
			if len(evs) == 0 {
				continue
			}

			filtered := w.filterEvents(evs)
			evs = make([]fsnotify.Event, 0)

			if len(filtered) > 0 {
				w.events <- filtered
			}
		case <-w.done:
			break OuterLoop
		}
	}
	close(w.done)
}

func (w *watcher) filterEvents(evs []fsnotify.Event) []fsnotify.Event {
	filtered := []fsnotify.Event{}

	for _, ev := range evs {
		log.Logger.Debug("Filter event", ev.Op, ev.Name)

		if w.ignoreFile(ev.Name) {
			log.Logger.Debug("Ignored based on the configuration")
			continue
		}

		// Sometimes during rm -rf operations a '"": REMOVE' is triggered. Just ignore these
		if ev.Name == "" {
			log.Logger.Debug("Ignored because event name is empty")
			continue
		}

		// Write and rename operations are often followed by CHMOD.
		// There may be valid use cases for rebuilding the site on CHMOD,
		// but that will require more complex logic than this simple conditional.
		// On OS X this seems to be related to Spotlight, see:
		// https://github.com/go-fsnotify/fsnotify/issues/15
		// A workaround is to put your site(s) on the Spotlight exception list,
		// but that may be a little mysterious for most end users.
		// So, for now, we skip reload on CHMOD.
		// We do have to check for WRITE though. On slower laptops a Chmod
		// could be aggregated with other important events, and we still want
		// to rebuild on those
		if ev.Op&(fsnotify.Chmod|fsnotify.Write|fsnotify.Create) == fsnotify.Chmod {
			log.Logger.Debug("Ignored because it is a CHMOD event")
			continue
		}

		walkAdder := func(path string, f os.FileInfo, err error) error {
			if err != nil {
				// Ignore attempt to access go temporary unmask
				if strings.Contains(err.Error(), "-go-tmp-umask") {
					return filepath.SkipDir
				}

				return fmt.Errorf("couldn't walk to path \"%s\": %v", path, err)
			}

			if f.IsDir() {
				log.Logger.Debugf("Adding created directory to watchlist %s", path)
				if err := w.FileWatcher.Add(path); err != nil {
					return err
				}
			} else {
				filtered = append(filtered, fsnotify.Event{Name: path, Op: fsnotify.Create})
			}

			return nil
		}

		// recursively add new directories to watch list
		// When mkdir -p is used, only the top directory triggers an event (at least on OSX)
		if ev.Op&fsnotify.Create == fsnotify.Create {
			info, err := os.Stat(ev.Name)
			if err != nil {
				log.Logger.Errorf("Error reading created file/dir %s: %v", ev.Name, err)
			}

			if info.Mode().IsDir() {
				if err = filepath.Walk(ev.Name, walkAdder); err != nil {
					log.Logger.Errorf("Error walking to created file/dir %s: %v", ev.Name, err)
				}
			}
		}

		log.Logger.Debug("Accepted")
		filtered = append(filtered, ev)
	}

	return filtered
}

// TODO: Support gitignore rules https://github.com/sabhiram/go-gitignore
func (w *watcher) ignoreFile(filename string) bool {
	ext := filepath.Ext(filename)
	baseName := filepath.Base(filename)

	istemp := strings.HasSuffix(ext, "~") ||
		(ext == ".swp") || // vim
		(ext == ".swx") || // vim
		(ext == ".tmp") || // generic temp file
		(ext == ".DS_Store") || // OSX Thumbnail
		baseName == "4913" || // vim
		strings.HasPrefix(ext, ".goutputstream") || // gnome
		strings.HasSuffix(ext, "jb_old___") || // intelliJ
		strings.HasSuffix(ext, "jb_tmp___") || // intelliJ
		strings.HasSuffix(ext, "jb_bak___") || // intelliJ
		strings.HasPrefix(ext, ".sb-") || // byword
		strings.HasPrefix(baseName, ".#") || // emacs
		strings.HasPrefix(baseName, "#") || // emacs
		strings.Contains(baseName, "-go-tmp-umask") // golang

	if istemp {
		log.Logger.Debug("Ignored file: temp")
		return true
	}

	info, err := os.Stat(filename)
	if err != nil {
		log.Logger.Debugf("Ignored file: stats failure: %v", err)
		return true
	}

	// if a file has been deleted after gaper was watching it
	// info will be nil in the other iterations
	if info == nil {
		log.Logger.Debug("Ignored file: reason not info")
		return true
	}

	// check if preset ignore is enabled
	if w.defaultIgnore {
		// check for hidden files and directories
		if name := info.Name(); name[0] == '.' && name != "." {
			log.Logger.Debug("Ignored file: hidden")
			return true
		}

		// check if it is a Go testing file
		if strings.HasSuffix(filename, "_test.go") {
			log.Logger.Debug("Ignored file: go test file")
			return true
		}

		// check if it is the vendor folder
		if info.IsDir() && info.Name() == "vendor" {
			log.Logger.Debug("Ignored file: vendor")
			return true
		}
	}

	if _, ignored := w.ignoreItems[filename]; ignored {
		log.Logger.Debug("Ignored file: ignored list")
		return true
	}

	if ext != "" && len(w.allowedExtensions) > 0 {
		if _, allowed := w.allowedExtensions[ext]; !allowed {
			log.Logger.Debugf("Ignored file: extension not allowed '%s'", ext)
			return true
		}
	}

	return false
}

// Close stops the watching of the files.
func (w *watcher) Close() {
	w.done <- struct{}{}
	w.FileWatcher.Close()
	w.ticker.Stop()
}

// Events get events occurred during the watching
// these events are emitted only a file changing is detected
func (w *watcher) Events() chan []fsnotify.Event {
	return w.events
}

// Errors get errors occurred during the watching
func (w *watcher) Errors() chan error {
	return w.errors
}

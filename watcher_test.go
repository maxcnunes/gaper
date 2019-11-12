package gaper

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWatcherDefaultValues(t *testing.T) {
	pollInterval := 0
	watchItems := []string{filepath.Join("testdata", "server")}
	var ignoreItems []string
	var extensions []string

	wCfg := WatcherConfig{
		DefaultIgnore: true,
		PollInterval:  pollInterval,
		WatchItems:    watchItems,
		IgnoreItems:   ignoreItems,
		Extensions:    extensions,
	}
	wt, err := NewWatcher(wCfg)

	expectedPath := "testdata/server"
	if runtime.GOOS == OSWindows {
		expectedPath = "testdata\\server"
	}

	w := wt.(*watcher)
	assert.Nil(t, err, "wacher error")
	assert.Equal(t, 500, w.pollInterval)
	assert.Equal(t, map[string]bool{expectedPath: true}, w.watchItems)
	assert.Len(t, w.ignoreItems, 0)
	assert.Equal(t, map[string]bool{".go": true}, w.allowedExtensions)
}

func TestWatcherGlobPath(t *testing.T) {
	pollInterval := 0
	watchItems := []string{filepath.Join("testdata", "server")}
	ignoreItems := []string{"./testdata/**/*_test.go"}
	var extensions []string

	wCfg := WatcherConfig{
		DefaultIgnore: true,
		PollInterval:  pollInterval,
		WatchItems:    watchItems,
		IgnoreItems:   ignoreItems,
		Extensions:    extensions,
	}
	wt, err := NewWatcher(wCfg)
	assert.Nil(t, err, "wacher error")
	w := wt.(*watcher)
	assert.Equal(t, map[string]bool{"testdata/server/main_test.go": true}, w.ignoreItems)
}

func TestWatcherRemoveOverlapdPaths(t *testing.T) {
	pollInterval := 0
	watchItems := []string{filepath.Join("testdata", "server")}
	ignoreItems := []string{"./testdata/server/**/*", "./testdata/server"}
	var extensions []string

	wCfg := WatcherConfig{
		DefaultIgnore: true,
		PollInterval:  pollInterval,
		WatchItems:    watchItems,
		IgnoreItems:   ignoreItems,
		Extensions:    extensions,
	}
	wt, err := NewWatcher(wCfg)
	assert.Nil(t, err, "wacher error")
	w := wt.(*watcher)
	assert.Equal(t, map[string]bool{"./testdata/server": true}, w.ignoreItems)
}

func TestWatcherWatchChange(t *testing.T) {
	srvdir := filepath.Join("testdata", "server")
	hiddendir := filepath.Join("testdata", "hidden-test")

	hiddenfile1 := filepath.Join("testdata", ".hidden-file")
	hiddenfile2 := filepath.Join("testdata", ".hidden-folder", ".gitkeep")
	mainfile := filepath.Join("testdata", "server", "main.go")
	testfile := filepath.Join("testdata", "server", "main_test.go")

	pollInterval := 0
	watchItems := []string{srvdir, hiddendir}
	ignoreItems := []string{testfile}
	extensions := []string{"go"}

	wCfg := WatcherConfig{
		DefaultIgnore: true,
		PollInterval:  pollInterval,
		WatchItems:    watchItems,
		IgnoreItems:   ignoreItems,
		Extensions:    extensions,
	}
	w, err := NewWatcher(wCfg)
	assert.Nil(t, err, "wacher error")

	go w.Watch()
	time.Sleep(time.Millisecond * 500)

	// update hidden files and dirs to check builtin hidden ignore is working
	os.Chtimes(hiddenfile1, time.Now(), time.Now())
	os.Chtimes(hiddenfile2, time.Now(), time.Now())

	// update testfile first to check ignore is working
	os.Chtimes(testfile, time.Now(), time.Now())

	time.Sleep(time.Millisecond * 500)
	os.Chtimes(mainfile, time.Now(), time.Now())

	select {
	case event := <-w.Events():
		assert.Equal(t, mainfile, event)
	case err := <-w.Errors():
		assert.Nil(t, err, "wacher event error")
	}
}

func TestWatcherIgnoreFile(t *testing.T) {
	testCases := []struct {
		name, file, ignoreFile      string
		defaultIgnore, expectIgnore bool
	}{
		{
			name:          "with default ignore enabled it ignores vendor folder",
			file:          "vendor",
			defaultIgnore: true,
			expectIgnore:  true,
		},
		{
			name:          "without default ignore enabled it does not ignore vendor folder",
			file:          "vendor",
			defaultIgnore: false,
			expectIgnore:  false,
		},
		{
			name:          "with default ignore enabled it ignores test file",
			file:          filepath.Join("testdata", "server", "main_test.go"),
			defaultIgnore: true,
			expectIgnore:  true,
		},
		{
			name:          "with default ignore enabled it does no ignore non test files which have test in the name",
			file:          filepath.Join("testdata", "ignore-test-name.txt"),
			defaultIgnore: true,
			expectIgnore:  false,
		},
		{
			name:          "without default ignore enabled it does not ignore test file",
			file:          filepath.Join("testdata", "server", "main_test.go"),
			defaultIgnore: false,
			expectIgnore:  false,
		},
		{
			name:          "with default ignore enabled it ignores ignored items",
			file:          filepath.Join("testdata", "server", "main.go"),
			ignoreFile:    filepath.Join("testdata", "server", "main.go"),
			defaultIgnore: true,
			expectIgnore:  true,
		},
		{
			name:          "without default ignore enabled it ignores ignored items",
			file:          filepath.Join("testdata", "server", "main.go"),
			ignoreFile:    filepath.Join("testdata", "server", "main.go"),
			defaultIgnore: false,
			expectIgnore:  true,
		},
	}

	// create vendor folder for testing
	if err := os.MkdirAll("vendor", os.ModePerm); err != nil {
		t.Fatal(err)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			srvdir := "."

			watchItems := []string{srvdir}
			ignoreItems := []string{}
			if len(tc.ignoreFile) > 0 {
				ignoreItems = append(ignoreItems, tc.ignoreFile)
			}
			extensions := []string{"go"}

			wCfg := WatcherConfig{
				DefaultIgnore: tc.defaultIgnore,
				WatchItems:    watchItems,
				IgnoreItems:   ignoreItems,
				Extensions:    extensions,
			}
			w, err := NewWatcher(wCfg)
			assert.Nil(t, err, "wacher error")

			wt := w.(*watcher)

			filePath := tc.file
			file, err := os.Open(filePath)
			if err != nil {
				t.Fatal(err)
			}

			fileInfo, err := file.Stat()
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tc.expectIgnore, wt.ignoreFile(filePath, fileInfo))
		})
	}
}

func TestWatcherResolvePaths(t *testing.T) {
	testCases := []struct {
		name                    string
		paths                   []string
		extensions, expectPaths map[string]bool
		err                     error
	}{
		{
			name:        "remove duplicated paths",
			paths:       []string{"testdata/test-duplicated-paths", "testdata/test-duplicated-paths"},
			extensions:  map[string]bool{".txt": true},
			expectPaths: map[string]bool{"testdata/test-duplicated-paths": true},
		},
		{
			name:        "remove duplicated paths from glob",
			paths:       []string{"testdata/test-duplicated-paths", "testdata/test-duplicated-paths/**/*"},
			extensions:  map[string]bool{".txt": true},
			expectPaths: map[string]bool{"testdata/test-duplicated-paths": true},
		},
		{
			name:        "remove duplicated paths from glob with inverse order",
			paths:       []string{"testdata/test-duplicated-paths/**/*", "testdata/test-duplicated-paths"},
			extensions:  map[string]bool{".txt": true},
			expectPaths: map[string]bool{"testdata/test-duplicated-paths": true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			paths, err := resolvePaths(tc.paths, tc.extensions)
			if tc.err == nil {
				assert.Nil(t, err, "resolve path error")
				assert.Equal(t, tc.expectPaths, paths)
			} else {
				assert.Equal(t, tc.err, err)
			}
		})
	}
}

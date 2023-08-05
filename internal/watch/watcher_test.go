package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testdataPath(paths ...string) string {
	return filepath.Join("..", "..", "testdata", filepath.Join(paths...))
}

func TestWatcherDefaultValues(t *testing.T) {
	watchItems := []string{testdataPath("server")}
	var ignoreItems []string

	wCfg := WatcherConfig{
		DefaultIgnore: true,
		Poll:          true,
		PollInterval:  500 * time.Millisecond,
		WatchItems:    watchItems,
		IgnoreItems:   ignoreItems,
		Extensions:    []string{"go"},
	}
	wt, err := NewWatcher(wCfg)

	expectedPath := testdataPath("server")

	w := wt.(*watcher)
	assert.Nil(t, err, "wacher error")
	assert.Equal(t, map[string]bool{expectedPath: true}, w.watchItems)
	assert.Len(t, w.ignoreItems, 0)
	assert.Equal(t, map[string]bool{".go": true}, w.allowedExtensions)
}

func TestWatcherGlobPath(t *testing.T) {
	watchItems := []string{testdataPath("server")}
	ignoreItems := []string{"../../testdata/**/*_test.go"}

	wCfg := WatcherConfig{
		DefaultIgnore: true,
		Poll:          true,
		PollInterval:  500 * time.Millisecond,
		WatchItems:    watchItems,
		IgnoreItems:   ignoreItems,
		Extensions:    []string{"go"},
	}
	wt, err := NewWatcher(wCfg)
	assert.Nil(t, err, "wacher error")
	w := wt.(*watcher)
	assert.Equal(t, map[string]bool{"../../testdata/server/main_test.go": true}, w.ignoreItems)
}

func TestWatcherRemoveOverlapdPaths(t *testing.T) {
	watchItems := []string{testdataPath("server")}
	ignoreItems := []string{"../../testdata/server/**/*", "../../testdata/server"}

	wCfg := WatcherConfig{
		DefaultIgnore: true,
		Poll:          true,
		PollInterval:  500 * time.Millisecond,
		WatchItems:    watchItems,
		IgnoreItems:   ignoreItems,
		Extensions:    []string{"go"},
	}
	wt, err := NewWatcher(wCfg)
	assert.Nil(t, err, "wacher error")
	w := wt.(*watcher)
	assert.Equal(t, map[string]bool{"../../testdata/server": true}, w.ignoreItems)
}

func TestWatcherWatchChange(t *testing.T) {
	srvdir := testdataPath("server")
	hiddendir := testdataPath("hidden-test")

	hiddenfile1 := testdataPath(".hidden-file")
	hiddenfile2 := testdataPath(".hidden-folder", ".gitkeep")
	mainfile := testdataPath("server", "main.go")
	testfile := testdataPath("server", "main_test.go")

	watchItems := []string{srvdir, hiddendir}
	ignoreItems := []string{testfile}

	wCfg := WatcherConfig{
		DefaultIgnore: true,
		Poll:          true,
		PollInterval:  500 * time.Millisecond,
		WatchItems:    watchItems,
		IgnoreItems:   ignoreItems,
		Extensions:    []string{"go"},
	}
	w, err := NewWatcher(wCfg)
	assert.Nil(t, err, "wacher error")

	go w.Watch()
	time.Sleep(time.Millisecond * 500)

	// update hidden files and dirs to check builtin hidden ignore is working
	err = os.Chtimes(hiddenfile1, time.Now(), time.Now())
	assert.Nil(t, err, "chtimes error")

	err = os.Chtimes(hiddenfile2, time.Now(), time.Now())
	assert.Nil(t, err, "chtimes error")

	// update testfile first to check ignore is working
	err = os.Chtimes(testfile, time.Now(), time.Now())
	assert.Nil(t, err, "chtimes error")

	time.Sleep(time.Millisecond * 500)
	err = os.Chtimes(mainfile, time.Now(), time.Now())
	assert.Nil(t, err, "chtimes error")

	select {
	case events := <-w.Events():
		assert.Equal(t, 1, len(events))
		assert.Equal(t, mainfile, events[0].Name)
	case err := <-w.Errors():
		assert.Nil(t, err, "wacher event error")
	}
}

func TestWatcherIgnoreFile(t *testing.T) {
	vendorPath := filepath.Join("..", "..", "vendor")

	testCases := []struct {
		name, file, ignoreFile      string
		defaultIgnore, expectIgnore bool
	}{
		{
			name:          "with default ignore enabled it ignores vendor folder",
			file:          vendorPath,
			defaultIgnore: true,
			expectIgnore:  true,
		},
		{
			name:          "without default ignore enabled it does not ignore vendor folder",
			file:          vendorPath,
			defaultIgnore: false,
			expectIgnore:  false,
		},
		{
			name:          "with default ignore enabled it ignores test file",
			file:          testdataPath("server", "main_test.go"),
			defaultIgnore: true,
			expectIgnore:  true,
		},
		{
			name:          "with default ignore enabled it does not ignore non test files which have test in the name",
			file:          testdataPath("ignore-test-name.txt"),
			defaultIgnore: true,
			expectIgnore:  false,
		},
		{
			name:          "without default ignore enabled it does not ignore test file",
			file:          testdataPath("server", "main_test.go"),
			defaultIgnore: false,
			expectIgnore:  false,
		},
		{
			name:          "with default ignore enabled it ignores ignored items",
			file:          testdataPath("server", "main.go"),
			ignoreFile:    testdataPath("server", "main.go"),
			defaultIgnore: true,
			expectIgnore:  true,
		},
		{
			name:          "without default ignore enabled it ignores ignored items",
			file:          testdataPath("server", "main.go"),
			ignoreFile:    testdataPath("server", "main.go"),
			defaultIgnore: false,
			expectIgnore:  true,
		},
	}

	// create vendor folder for testing
	if err := os.MkdirAll(vendorPath, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(vendorPath)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			srvdir := filepath.Join("..", "..")

			watchItems := []string{srvdir}
			ignoreItems := []string{}
			if len(tc.ignoreFile) > 0 {
				ignoreItems = append(ignoreItems, tc.ignoreFile)
			}

			wCfg := WatcherConfig{
				DefaultIgnore: tc.defaultIgnore,
				WatchItems:    watchItems,
				IgnoreItems:   ignoreItems,
				Extensions:    []string{"go", "txt"},
			}
			w, err := NewWatcher(wCfg)
			assert.Nil(t, err, "wacher error")

			wt := w.(*watcher)

			assert.Equal(t, tc.expectIgnore, wt.ignoreFile(tc.file))
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

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

	w, err := NewWatcher(pollInterval, watchItems, ignoreItems, extensions)

	expectedPath := "testdata/server"
	if runtime.GOOS == OSWindows {
		expectedPath = "testdata\\server"
	}

	assert.Nil(t, err, "wacher error")
	assert.Equal(t, 500, w.PollInterval)
	assert.Equal(t, map[string]bool{expectedPath: true}, w.WatchItems)
	assert.Len(t, w.IgnoreItems, 0)
	assert.Equal(t, map[string]bool{".go": true}, w.AllowedExtensions)
}

func TestWatcherGlobPath(t *testing.T) {
	pollInterval := 0
	watchItems := []string{filepath.Join("testdata", "server")}
	ignoreItems := []string{"./testdata/**/*_test.go"}
	var extensions []string

	w, err := NewWatcher(pollInterval, watchItems, ignoreItems, extensions)
	assert.Nil(t, err, "wacher error")
	assert.Equal(t, map[string]bool{"testdata/server/main_test.go": true}, w.IgnoreItems)
}

func TestWatcherRemoveOverlapdPaths(t *testing.T) {
	pollInterval := 0
	watchItems := []string{filepath.Join("testdata", "server")}
	ignoreItems := []string{"./testdata/**/*", "./testdata/server"}
	var extensions []string

	w, err := NewWatcher(pollInterval, watchItems, ignoreItems, extensions)
	assert.Nil(t, err, "wacher error")
	assert.Equal(t, map[string]bool{"./testdata/server": true}, w.IgnoreItems)
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

	w, err := NewWatcher(pollInterval, watchItems, ignoreItems, extensions)
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
	case event := <-w.Events:
		assert.Equal(t, mainfile, event)
	case err := <-w.Errors:
		assert.Nil(t, err, "wacher event error")
	}
}

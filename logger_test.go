package gaper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoggerDefault(t *testing.T) {
	l := NewLogger("gaper-test")
	assert.Equal(t, l.verbose, false)
}

func TestLoggerEnableVerbose(t *testing.T) {
	l := NewLogger("gaper-test")
	l.Verbose(true)
	assert.Equal(t, l.verbose, true)
}

func TestLoggerRunAllLogsWithoutVerbose(t *testing.T) {
	// no asserts, just checking it doesn't crash
	l := NewLogger("gaper-test")
	l.Debug("debug")
	l.Debugf("%s", "debug")
	l.Info("info")
	l.Error("error")
	l.Errorf("%s", "error")
}

func TestLoggerRunAllLogsWithVerbose(t *testing.T) {
	// no asserts, just checking it doesn't crash
	l := NewLogger("gaper-test")
	l.Verbose(true)
	l.Debug("debug")
	l.Debugf("%s", "debug")
	l.Info("info")
	l.Error("error")
	l.Errorf("%s", "error")
}

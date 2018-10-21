package gaper

import (
	"log"
	"os"

	"github.com/fatih/color"
)

// logger use by the whole package
var logger = newLogger("gaper")

// Logger give access to external packages to use gaper logger
func Logger() *LoggerEntity {
	return logger
}

// LoggerEntity used by gaper
type LoggerEntity struct {
	verbose  bool
	logDebug *log.Logger
	logInfo  *log.Logger
	logError *log.Logger
}

// newLogger creates a new logger
func newLogger(prefix string) *LoggerEntity {
	prefix = "[" + prefix + "] "
	return &LoggerEntity{
		verbose:  false,
		logDebug: log.New(os.Stdout, prefix, 0),
		logInfo:  log.New(os.Stdout, color.CyanString(prefix), 0),
		logError: log.New(os.Stdout, color.RedString(prefix), 0),
	}
}

// Verbose toggle this logger verbosity
func (l *LoggerEntity) Verbose(verbose bool) {
	l.verbose = verbose
}

// Debug logs a debug message
func (l *LoggerEntity) Debug(v ...interface{}) {
	if l.verbose {
		l.logDebug.Println(v...)
	}
}

// Debugf logs a debug message with format
func (l *LoggerEntity) Debugf(format string, v ...interface{}) {
	if l.verbose {
		l.logDebug.Printf(format, v...)
	}
}

// Info logs a info message
func (l *LoggerEntity) Info(v ...interface{}) {
	l.logInfo.Println(v...)
}

// Error logs an error message
func (l *LoggerEntity) Error(v ...interface{}) {
	l.logError.Println(v...)
}

// Errorf logs and error message with format
func (l *LoggerEntity) Errorf(format string, v ...interface{}) {
	l.logError.Printf(format, v...)
}

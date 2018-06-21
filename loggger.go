package main

import (
	"log"
	"os"

	"github.com/fatih/color"
)

// Logger used by gaper
type Logger struct {
	verbose  bool
	logDebug *log.Logger
	logInfo  *log.Logger
	logError *log.Logger
}

// NewLogger creates a new logger
func NewLogger(prefix string) *Logger {
	prefix = "[" + prefix + "] "
	return &Logger{
		verbose:  false,
		logDebug: log.New(os.Stdout, prefix, 0),
		logInfo:  log.New(os.Stdout, color.CyanString(prefix), 0),
		logError: log.New(os.Stdout, color.RedString(prefix), 0),
	}
}

// Verbose toggle this logger verbosity
func (l *Logger) Verbose(verbose bool) {
	l.verbose = verbose
}

// Debug logs a debug message
func (l *Logger) Debug(v ...interface{}) {
	if l.verbose {
		l.logDebug.Println(v...)
	}
}

// Debugf logs a debug message with format
func (l *Logger) Debugf(format string, v ...interface{}) {
	if l.verbose {
		l.logDebug.Printf(format, v...)
	}
}

// Info logs a info message
func (l *Logger) Info(v ...interface{}) {
	l.logInfo.Println(v...)
}

// Error logs an error message
func (l *Logger) Error(v ...interface{}) {
	l.logError.Println(v...)
}

// Errorf logs and error message with format
func (l *Logger) Errorf(format string, v ...interface{}) {
	l.logError.Printf(format, v...)
}

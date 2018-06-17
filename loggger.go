package main

import (
	"log"
	"os"

	"github.com/fatih/color"
)

// Logger ..
type Logger struct {
	verbose  bool
	logDebug *log.Logger
	logWarn  *log.Logger
	logInfo  *log.Logger
	logError *log.Logger
}

// NewLogger ...
func NewLogger(prefix string) *Logger {
	prefix = "[" + prefix + "] "
	return &Logger{
		verbose:  false,
		logDebug: log.New(os.Stdout, prefix, 0),
		logWarn:  log.New(os.Stdout, color.YellowString(prefix), 0),
		logInfo:  log.New(os.Stdout, color.CyanString(prefix), 0),
		logError: log.New(os.Stdout, color.RedString(prefix), 0),
	}
}

// Verbose ...
func (l *Logger) Verbose(verbose bool) {
	l.verbose = verbose
}

// Debug ...
func (l *Logger) Debug(v ...interface{}) {
	if l.verbose {
		l.logDebug.Println(v...)
	}
}

// Warn ...
func (l *Logger) Warn(v ...interface{}) {
	l.logWarn.Println(v...)
}

// Info ...
func (l *Logger) Info(v ...interface{}) {
	l.logInfo.Println(v...)
}

// Error ...
func (l *Logger) Error(v ...interface{}) {
	l.logError.Println(v...)
}

// Errorf ...
func (l *Logger) Errorf(format string, v ...interface{}) {
	l.logError.Printf(format, v...)
}

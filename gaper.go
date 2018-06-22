// Package gaper implements a supervisor restarts a go project
// when it crashes or a watched file changes
package gaper

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	shellwords "github.com/mattn/go-shellwords"
)

// DefaultBuildPath is the default build and watched path
var DefaultBuildPath = "."

// DefaultExtensions is the default watched extension
var DefaultExtensions = []string{"go"}

// DefaultPoolInterval is the time in ms used by the watcher to wait between scans
var DefaultPoolInterval = 500

var logger = NewLogger("gaper")

// exit statuses
var exitStatusSuccess = 0
var exitStatusError = 1

// Config contains all settings supported by gaper
type Config struct {
	BinName           string
	BuildPath         string
	BuildArgs         []string
	BuildArgsMerged   string
	ProgramArgs       []string
	ProgramArgsMerged string
	WatchItems        []string
	IgnoreItems       []string
	PollInterval      int
	Extensions        []string
	NoRestartOn       string
	Verbose           bool
	ExitOnSIGINT      bool
}

// Run in the gaper high level API
// It starts the whole gaper process watching for file changes or exit codes
// and restarting the program
func Run(cfg *Config) error { // nolint: gocyclo
	var err error
	logger.Verbose(cfg.Verbose)
	logger.Debug("Starting gaper")

	if len(cfg.BuildPath) == 0 {
		cfg.BuildPath = DefaultBuildPath
	}

	cfg.BuildArgs, err = parseInnerArgs(cfg.BuildArgs, cfg.BuildArgsMerged)
	if err != nil {
		return err
	}

	cfg.ProgramArgs, err = parseInnerArgs(cfg.ProgramArgs, cfg.ProgramArgsMerged)
	if err != nil {
		return err
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	if len(cfg.WatchItems) == 0 {
		cfg.WatchItems = append(cfg.WatchItems, cfg.BuildPath)
	}

	builder := NewBuilder(cfg.BuildPath, cfg.BinName, wd, cfg.BuildArgs)
	runner := NewRunner(os.Stdout, os.Stderr, filepath.Join(wd, builder.Binary()), cfg.ProgramArgs)

	if err = builder.Build(); err != nil {
		return fmt.Errorf("build error: %v", err)
	}

	shutdown(runner, cfg.ExitOnSIGINT)

	if _, err = runner.Run(); err != nil {
		return fmt.Errorf("run error: %v", err)
	}

	watcher, err := NewWatcher(cfg.PollInterval, cfg.WatchItems, cfg.IgnoreItems, cfg.Extensions)
	if err != nil {
		return fmt.Errorf("watcher error: %v", err)
	}

	var changeRestart bool

	go watcher.Watch()
	for {
		select {
		case event := <-watcher.Events:
			logger.Debug("Detected new changed file: ", event)
			changeRestart = true
			if err := restart(builder, runner); err != nil {
				return err
			}
		case err := <-watcher.Errors:
			return fmt.Errorf("error on watching files: %v", err)
		case err := <-runner.Errors():
			if changeRestart {
				changeRestart = false
			} else {
				logger.Debug("Detected program exit: ", err)
				if err = handleProgramExit(builder, runner, err, cfg.NoRestartOn); err != nil {
					return err
				}
			}
		default:
			time.Sleep(time.Duration(cfg.PollInterval) * time.Millisecond)
		}
	}
}

func restart(builder Builder, runner Runner) error {
	logger.Debug("Restarting program")

	// kill process if it is running
	if !runner.Exited() {
		if err := runner.Kill(); err != nil {
			return fmt.Errorf("kill error: %v", err)
		}
	}

	if err := builder.Build(); err != nil {
		return fmt.Errorf("build error: %v", err)
	}

	if _, err := runner.Run(); err != nil {
		return fmt.Errorf("run error: %v", err)
	}

	return nil
}

func handleProgramExit(builder Builder, runner Runner, err error, noRestartOn string) error {
	exiterr, ok := err.(*exec.ExitError)
	if !ok {
		return fmt.Errorf("couldn't handle program crash restart: %v", err)
	}

	status, oks := exiterr.Sys().(syscall.WaitStatus)
	if !oks {
		return fmt.Errorf("couldn't resolve exit status: %v", err)
	}

	exitStatus := status.ExitStatus()

	// if "error", an exit code of 0 will still restart.
	if noRestartOn == "error" && exitStatus == exitStatusError {
		return nil
	}

	// if "success", no restart only if exit code is 0.
	if noRestartOn == "success" && exitStatus == exitStatusSuccess {
		return nil
	}

	// if "exit", no restart regardless of exit code.
	if noRestartOn == "exit" {
		return nil
	}

	return restart(builder, runner)
}

func shutdown(runner Runner, exitOnSIGINT bool) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		s := <-c
		logger.Debug("Got signal: ", s)

		if err := runner.Kill(); err != nil {
			logger.Error("Error killing: ", err)
		}

		if exitOnSIGINT {
			os.Exit(0)
		}
	}()
}

func parseInnerArgs(args []string, argsm string) ([]string, error) {
	if len(args) > 0 || len(argsm) == 0 {
		return args, nil
	}

	return shellwords.Parse(argsm)
}

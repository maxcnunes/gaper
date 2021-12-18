// Package gaper implements a supervisor restarts a go project
// when it crashes or a watched file changes
package gaper

import (
	"fmt"
	"os"
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

// No restart types
var (
	NoRestartOnError   = "error"
	NoRestartOnSuccess = "success"
	NoRestartOnExit    = "exit"
)

// exit statuses
var exitStatusSuccess = 0
var exitStatusError = 1

// Config contains all settings supported by gaper
type Config struct {
	BinName              string
	BuildPath            string
	BuildArgs            []string
	BuildArgsMerged      string
	ProgramArgs          []string
	ProgramArgsMerged    string
	WatchItems           []string
	IgnoreItems          []string
	PollInterval         int
	Extensions           []string
	NoRestartOn          string
	DisableDefaultIgnore bool
	WorkingDirectory     string
}

// Run starts the whole gaper process watching for file changes or exit codes
// and restarting the program
func Run(cfg *Config, chOSSiginal chan os.Signal) error {
	logger.Debug("Starting gaper")

	if err := setupConfig(cfg); err != nil {
		return err
	}

	logger.Debugf("Config: %+v", cfg)

	wCfg := WatcherConfig{
		DefaultIgnore: !cfg.DisableDefaultIgnore,
		PollInterval:  cfg.PollInterval,
		WatchItems:    cfg.WatchItems,
		IgnoreItems:   cfg.IgnoreItems,
		Extensions:    cfg.Extensions,
	}

	builder := NewBuilder(cfg.BuildPath, cfg.BinName, cfg.WorkingDirectory, cfg.BuildArgs)
	runner := NewRunner(os.Stdout, os.Stderr, filepath.Join(cfg.WorkingDirectory, builder.Binary()), cfg.ProgramArgs)
	watcher, err := NewWatcher(wCfg)
	if err != nil {
		return fmt.Errorf("watcher error: %v", err)
	}

	return run(cfg, chOSSiginal, builder, runner, watcher)
}

// nolint: gocyclo
func run(cfg *Config, chOSSiginal chan os.Signal, builder Builder, runner Runner, watcher Watcher) error {
	if err := builder.Build(); err != nil {
		return fmt.Errorf("build error: %v", err)
	}

	// listen for OS signals
	signal.Notify(chOSSiginal, os.Interrupt, syscall.SIGTERM)

	if _, err := runner.Run(); err != nil {
		return fmt.Errorf("run error: %v", err)
	}

	// flag to know if an exit was caused by a restart from a file changing
	changeRestart := false

	go watcher.Watch()
	for {
		select {
		case event := <-watcher.Events():
			logger.Debug("Detected new changed file:", event)
			if changeRestart {
				logger.Debug("Skip restart due to existing on going restart")
				continue
			}

			changeRestart = runner.IsRunning()

			if err := restart(builder, runner); err != nil {
				return err
			}
		case err := <-watcher.Errors():
			return fmt.Errorf("error on watching files: %v", err)
		case err := <-runner.Errors():
			logger.Debug("Detected program exit:", err)

			// ignore exit by change
			if changeRestart {
				changeRestart = false
				continue
			}

			if err = handleProgramExit(builder, runner, err, cfg.NoRestartOn); err != nil {
				return err
			}
		case signal := <-chOSSiginal:
			logger.Debug("Got signal:", signal)

			if err := runner.Kill(); err != nil {
				logger.Error("Error killing:", err)
			}

			return fmt.Errorf("OS signal: %v", signal)
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
		logger.Error("Error building binary during a restart:", err)
		return nil
	}

	if _, err := runner.Run(); err != nil {
		logger.Error("Error starting process during a restart:", err)
		return nil
	}

	return nil
}

func handleProgramExit(builder Builder, runner Runner, err error, noRestartOn string) error {
	exitStatus := runner.ExitStatus(err)

	// if "error", an exit code of 0 will still restart.
	if noRestartOn == NoRestartOnError && exitStatus == exitStatusError {
		return nil
	}

	// if "success", no restart only if exit code is 0.
	if noRestartOn == NoRestartOnSuccess && exitStatus == exitStatusSuccess {
		return nil
	}

	// if "exit", no restart regardless of exit code.
	if noRestartOn == NoRestartOnExit {
		return nil
	}

	return restart(builder, runner)
}

func setupConfig(cfg *Config) error {
	var err error

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

	cfg.WorkingDirectory, err = os.Getwd()
	if err != nil {
		return err
	}

	if len(cfg.WatchItems) == 0 {
		cfg.WatchItems = append(cfg.WatchItems, cfg.BuildPath)
	}

	return nil
}

func parseInnerArgs(args []string, argsm string) ([]string, error) {
	if len(args) > 0 || len(argsm) == 0 {
		return args, nil
	}

	return shellwords.Parse(argsm)
}

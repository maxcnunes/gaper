package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	shellwords "github.com/mattn/go-shellwords"
	"github.com/urfave/cli"
)

var logger = NewLogger("gaper")

// default values
var defaultExtensions = cli.StringSlice{"go"}
var defaultPoolInterval = 500

// exit statuses
var exitStatusSuccess = 0
var exitStatusError = 1

// Config ...
type Config struct {
	BinName         string
	BuildPath       string
	BuildArgs       []string
	BuildArgsMerged string
	ProgramArgs     []string
	Verbose         bool
	WatchItems      []string
	IgnoreItems     []string
	PollInterval    int
	Extensions      []string
	NoRestartOn     string
}

func main() {
	parseArgs := func(c *cli.Context) *Config {
		return &Config{
			BinName:         c.String("bin-name"),
			BuildPath:       c.String("build-path"),
			BuildArgsMerged: c.String("build-args"),
			ProgramArgs:     c.Args(),
			Verbose:         c.Bool("verbose"),
			WatchItems:      c.StringSlice("watch"),
			IgnoreItems:     c.StringSlice("ignore"),
			PollInterval:    c.Int("poll-interval"),
			Extensions:      c.StringSlice("extensions"),
			NoRestartOn:     c.String("no-restart-on"),
		}
	}

	app := cli.NewApp()
	app.Name = "gaper"
	app.Usage = "Used to restart programs when they crash or a watched file changes"

	app.Action = func(c *cli.Context) {
		args := parseArgs(c)
		if err := runGaper(args); err != nil {
			logger.Error(err)
			os.Exit(1)
		}
	}

	// supported arguments
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "bin-name",
			Usage: "name for the binary built by Gaper for the executed program",
		},
		cli.StringFlag{
			Name:  "build-path",
			Usage: "path to the program source code",
		},
		cli.StringFlag{
			Name:  "build-args",
			Usage: "build arguments passed to the program",
		},
		cli.BoolFlag{
			Name:  "verbose",
			Usage: "turns on the verbose messages from Gaper",
		},
		cli.StringSliceFlag{
			Name:  "watch, w",
			Usage: "list of folders or files to watch for changes",
		},
		cli.StringSliceFlag{
			Name:  "ignore, i",
			Usage: "list of folders or files to ignore for changes",
		},
		cli.IntFlag{
			Name:  "poll-interval, p",
			Value: defaultPoolInterval,
			Usage: "how often in milliseconds to poll watched files for changes",
		},
		cli.StringSliceFlag{
			Name:  "extensions, e",
			Value: &defaultExtensions,
			Usage: "a comma-delimited list of file extensions to watch for changes",
		},
		cli.StringFlag{
			Name: "no-restart-on, n",
			Usage: "don't automatically restart the supervised program if it ends:\n" +
				"\t\tif \"error\", an exit code of 0 will still restart.\n" +
				"\t\tif \"exit\", no restart regardless of exit code.\n" +
				"\t\tif \"success\", no restart only if exit code is 0.",
		},
	}

	if err := app.Run(os.Args); err != nil {
		logger.Errorf("Error running gaper: %v", err)
		os.Exit(1)
	}
}

// nolint: gocyclo
func runGaper(cfg *Config) error {
	var err error
	logger.Verbose(cfg.Verbose)
	logger.Debug("Starting gaper")

	if len(cfg.BuildArgs) == 0 && len(cfg.BuildArgsMerged) > 0 {
		cfg.BuildArgs, err = shellwords.Parse(cfg.BuildArgsMerged)
		if err != nil {
			return err
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	// resolve bin name by current folder name
	if cfg.BinName == "" {
		cfg.BinName = filepath.Base(wd)
	}

	if len(cfg.WatchItems) == 0 {
		cfg.WatchItems = append(cfg.WatchItems, cfg.BuildPath)
	}

	logger.Debug("Settings: ")
	logger.Debug("    | bin: ", cfg.BinName)
	logger.Debug("    | build path: ", cfg.BuildPath)
	logger.Debug("    | build args: ", cfg.BuildArgs)
	logger.Debug("    | verbose: ", cfg.Verbose)
	logger.Debug("    | watch: ", cfg.WatchItems)
	logger.Debug("    | ignore: ", cfg.IgnoreItems)
	logger.Debug("    | poll interval: ", cfg.PollInterval)
	logger.Debug("    | extensions: ", cfg.Extensions)
	logger.Debug("    | no restart on: ", cfg.NoRestartOn)
	logger.Debug("    | working directory: ", wd)

	builder := NewBuilder(cfg.BuildPath, cfg.BinName, wd, cfg.BuildArgs)
	runner := NewRunner(os.Stdout, os.Stderr, filepath.Join(wd, builder.Binary()), cfg.ProgramArgs)

	if err = builder.Build(); err != nil {
		return fmt.Errorf("build error: %v", err)
	}

	shutdown(runner)

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
			restart(builder, runner)
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

	restart(builder, runner)
	return nil
}

func shutdown(runner Runner) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		s := <-c
		logger.Debug("Got signal: ", s)
		err := runner.Kill()
		if err != nil {
			logger.Error("Error killing: ", err)
		}
		os.Exit(1)
	}()
}

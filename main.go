package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	shellwords "github.com/mattn/go-shellwords"
	"github.com/urfave/cli"
)

var logger = NewLogger("gaper")

var defaultExtensions = cli.StringSlice{"go"}
var defaultPoolInterval = 500

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
		cli.StringSliceFlag{
			Name: "--no-restart-on, n",
			Usage: "don't automatically restart the supervised program if it ends.\n" +
				"\t\tIf \"error\", an exit code of 0 will still restart.\n" +
				"\t\tIf \"exit\", no restart regardless of exit code.\n" +
				"\t\tIf \"success\", no restart only if exit code is 0.",
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
	logger.Debug("    | poll-interval: ", cfg.PollInterval)
	logger.Debug("    | extensions: ", cfg.Extensions)
	logger.Debug("    | working directory: ", wd)

	builder := NewBuilder(cfg.BuildPath, cfg.BinName, wd, cfg.BuildArgs)
	runner := NewRunner(os.Stdout, filepath.Join(wd, builder.Binary()), cfg.ProgramArgs)

	if err = builder.Build(); err != nil {
		return fmt.Errorf("build error: %v", err)
	}

	shutdown(runner)

	if _, err = runner.Run(); err != nil {
		return fmt.Errorf("run error: %v", err)
	}

	watcher := NewWatcher(cfg.PollInterval, cfg.WatchItems, cfg.IgnoreItems, cfg.Extensions)

	go watcher.Watch()
	for {
		select {
		case event := <-watcher.Events:
			logger.Debug("Detected new changed file: ", event)
			if err = runner.Kill(); err != nil {
				return fmt.Errorf("kill error: %v", err)
			}
			if err = builder.Build(); err != nil {
				return fmt.Errorf("build error: %v", err)
			}
			if _, err = runner.Run(); err != nil {
				return fmt.Errorf("run error: %v", err)
			}
		case err := <-watcher.Errors:
			return fmt.Errorf("error on watching files: %v", err)
		default:
			logger.Debug("Waiting watch event")
			time.Sleep(time.Duration(cfg.PollInterval) * time.Millisecond)
		}
	}
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

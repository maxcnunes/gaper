package main

import (
	"os"

	"github.com/maxcnunes/gaper"
	"github.com/urfave/cli"
)

// build info
var (
	// keep the version hardcoded because on installing it through "go get/install"
	// it doesn't apply the build tags to override it. So, it is make easier for
	// people using in that case to find out which version they are using
	version = "1.0.1-dev"
)

var logger = gaper.NewLogger("gaper")

func main() {
	parseArgs := func(c *cli.Context) *gaper.Config {
		return &gaper.Config{
			BinName:              c.String("bin-name"),
			BuildPath:            c.String("build-path"),
			BuildArgsMerged:      c.String("build-args"),
			ProgramArgsMerged:    c.String("program-args"),
			Verbose:              c.Bool("verbose"),
			DisableDefaultIgnore: c.Bool("disable-default-ignore"),
			WatchItems:           c.StringSlice("watch"),
			IgnoreItems:          c.StringSlice("ignore"),
			PollInterval:         c.Int("poll-interval"),
			Extensions:           c.StringSlice("extensions"),
			NoRestartOn:          c.String("no-restart-on"),
		}
	}

	app := cli.NewApp()
	app.Name = "gaper"
	app.Usage = "Used to build and restart a Go project when it crashes or some watched file changes"
	app.Version = version

	app.Action = func(c *cli.Context) {
		args := parseArgs(c)
		chOSSiginal := make(chan os.Signal, 2)
		logger.Verbose(args.Verbose)

		if err := gaper.Run(args, chOSSiginal); err != nil {
			logger.Error(err)
			os.Exit(1)
		}
	}

	exts := make(cli.StringSlice, len(gaper.DefaultExtensions))
	for i := range gaper.DefaultExtensions {
		exts[i] = gaper.DefaultExtensions[i]
	}

	// supported arguments
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "bin-name",
			Usage: "name for the binary built by gaper for the executed program (default current directory name)",
		},
		cli.StringFlag{
			Name:  "build-path",
			Value: gaper.DefaultBuildPath,
			Usage: "path to the program source code",
		},
		cli.StringFlag{
			Name:  "build-args",
			Usage: "arguments used on building the program",
		},
		cli.StringFlag{
			Name:  "program-args",
			Usage: "arguments used on executing the program",
		},
		cli.BoolFlag{
			Name:  "verbose",
			Usage: "turns on the verbose messages from gaper",
		},
		cli.BoolFlag{
			Name:  "disable-default-ignore",
			Usage: "turns off default ignore for hidden files and folders, \"*_test.go\" files, and vendor folder",
		},
		cli.StringSliceFlag{
			Name:  "watch, w",
			Usage: "list of folders or files to watch for changes",
		},
		cli.StringSliceFlag{
			Name: "ignore, i",
			Usage: "list of folders or files to ignore for changes\n" +
				"\t\t(always ignores all hidden files and directories)",
		},
		cli.IntFlag{
			Name:  "poll-interval, p",
			Value: gaper.DefaultPoolInterval,
			Usage: "how often in milliseconds to poll watched files for changes",
		},
		cli.StringSliceFlag{
			Name:  "extensions, e",
			Value: &exts,
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

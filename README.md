<p align="center">
  <img width="200px" src="https://raw.githubusercontent.com/maxcnunes/gaper/master/gopher-gaper.png">
  <h3 align="center">gaper</h3>
  <p align="center">
   Used to build and restart a Go project when it crashes or some watched file changes
   <br />
   <b>Aimed to be used in development only.</b>
  </p>
</p>

---

[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)](LICENSE.md)
[![Linux - Build Status](https://travis-ci.org/maxcnunes/gaper.svg?branch=master)](https://travis-ci.org/maxcnunes/gaper)
[![Windows - Build status](https://ci.appveyor.com/api/projects/status/e0g00kmxwv44?svg=true)](https://ci.appveyor.com/project/maxcnunes/gaper)
[![Coverage Status](https://codecov.io/gh/maxcnunes/gaper/branch/master/graph/badge.svg)](https://codecov.io/gh/maxcnunes/gaper)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/maxcnunes/gaper)
[![Go Report Card](https://goreportcard.com/badge/github.com/maxcnunes/gaper)](https://goreportcard.com/report/github.com/maxcnunes/gaper)
[![Powered By: GoReleaser](https://img.shields.io/badge/powered%20by-goreleaser-green.svg?style=flat-square)](https://github.com/goreleaser)

## Changelog

See [Releases](https://github.com/maxcnunes/gaper/releases) for detailed history changes.

## Installation

Using go tooling:

```
go get -u github.com/maxcnunes/gaper/cmd/gaper
```

Or, downloading the binary instead (example for version 1.0.3, make sure you are using the latest version though):

```
curl -SL https://github.com/maxcnunes/gaper/releases/download/v1.0.3/gaper_1.0.3_linux_amd64.tar.gz | tar -xvzf - -C "${GOPATH}/bin"
```

## Usage

```
NAME:
   gaper - Used to build and restart a Go project when it crashes or some watched file changes

USAGE:
   gaper [global options] command [command options] [arguments...]

VERSION:
   version

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --bin-name value                 name for the binary built by gaper for the executed program (default current directory name)
   --build-path value               path to the program source code (default: ".")
   --build-args value               arguments used on building the program
   --program-args value             arguments used on executing the program
   --verbose                        turns on the verbose messages from gaper
   --disable-default-ignore         turns off default ignore for hidden files and folders, "*_test.go" files, and vendor folder
   --watch value, -w value          list of folders or files to watch for changes
   --ignore value, -i value         list of folders or files to ignore for changes
   --poll-interval value, -p value  how often in milliseconds to poll watched files for changes (default: 500)
   --extensions value, -e value     a comma-delimited list of file extensions to watch for changes (default: "go")
   --no-restart-on value, -n value  don't automatically restart the supervised program if it ends:
                                      if "error", an exit code of 0 will still restart.
                                      if "exit", no restart regardless of exit code.
                                      if "success", no restart only if exit code is 0.
   --help, -h                       show help
   --version, -v                    print the version
```

### Watch and Ignore paths

For those options Gaper supports static paths (e.g. `build/`, `seed.go`) or glob paths (e.g. `migrations/**/up.go`, `*_test.go`).

On using a path to a directory please add a `/` at the end (e.g. `build/`) to make sure Gaper won't include other matches that starts with that same value (e.g. `build/`, `build_settings.go`).

### Default ignore settings

Since in most projects there is no need to watch changes of:

* hidden files and folders
* test files (`*_test.go`)
* vendor folder

Gaper by default ignores those cases already. Although, if you need Gaper to watch those files anyway it is possible to disable this setting with `--disable-default-ignore` argument.

### Watch method

Currently Gaper uses polling to watch file changes. We have plans to [support fs events](https://github.com/maxcnunes/gaper/issues/12) though in a near future.

### Examples

Using all defaults provided by Gaper:

```
gaper
```

Ignore watch over all test files:

> no need for this if you have not disabled the default ignore settings `--disable-default-ignore`

```
--ignore './**/*_test.go'
```

## Contributing

See the [Contributing guide](/CONTRIBUTING.md) for steps on how to contribute to this project.

## Reference

This package was heavily inspired by [gin](https://github.com/codegangsta/gin) and [node-supervisor](https://github.com/petruisfan/node-supervisor).

Basically, Gaper is a mixing of those projects above. It started from **gin** code base and I rewrote it aiming to get
something similar to **node-supervisor** (but simpler). A big thanks for those projects and for the people behind it!
:clap::clap:

### How is Gaper different of Gin

The main difference is that Gaper removes a layer of complexity from Gin which has a proxy running on top of 
the executed server. It allows to postpone a build and reload the server when the first call hits it. With Gaper 
we don't care about that feature, it just restarts your server whenever a change is made.

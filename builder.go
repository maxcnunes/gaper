package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Builder ...
type Builder interface {
	Build() error
	Binary() string
	Errors() string
}

type builder struct {
	dir       string
	binary    string
	errors    string
	wd        string
	buildArgs []string
}

// NewBuilder ...
func NewBuilder(dir string, bin string, wd string, buildArgs []string) Builder {
	if len(bin) == 0 {
		bin = "bin"
	}

	// does not work on Windows without the ".exe" extension
	if runtime.GOOS == OSWindows {
		// check if it already has the .exe extension
		if !strings.HasSuffix(bin, ".exe") {
			bin += ".exe"
		}
	}

	return &builder{dir: dir, binary: bin, wd: wd, buildArgs: buildArgs}
}

// Binary ...
func (b *builder) Binary() string {
	return b.binary
}

// Errors ...
func (b *builder) Errors() string {
	return b.errors
}

// Build ...
func (b *builder) Build() error {
	logger.Info("Building program")
	args := append([]string{"go", "build", "-o", filepath.Join(b.wd, b.binary)}, b.buildArgs...)
	logger.Debug("Build command", args)

	command := exec.Command(args[0], args[1:]...) // nolint gas
	command.Dir = b.dir

	output, err := command.CombinedOutput()
	if err != nil {
		return err
	}

	if command.ProcessState.Success() {
		b.errors = ""
	} else {
		b.errors = string(output)
	}

	if len(b.errors) > 0 {
		return fmt.Errorf(b.errors)
	}

	return nil
}

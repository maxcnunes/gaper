package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// OSWindows ...
const OSWindows = "windows"

// Runner ...
type Runner interface {
	Run() (*exec.Cmd, error)
	Kill() error
	Errors() chan error
	Exited() bool
}

type runner struct {
	bin          string
	args         []string
	writerStdout io.Writer
	writerStderr io.Writer
	command      *exec.Cmd
	starttime    time.Time
	errors       chan error
}

// NewRunner ...
func NewRunner(wStdout io.Writer, wStderr io.Writer, bin string, args []string) Runner {
	return &runner{
		bin:          bin,
		args:         args,
		writerStdout: wStdout,
		writerStderr: wStderr,
		starttime:    time.Now(),
		errors:       make(chan error),
	}
}

// Run ...
func (r *runner) Run() (*exec.Cmd, error) {
	logger.Info("Starting program")

	if r.command == nil || r.Exited() {
		if err := r.runBin(); err != nil {
			return nil, fmt.Errorf("error running: %v", err)
		}

		time.Sleep(250 * time.Millisecond)
		return r.command, nil
	}

	return r.command, nil
}

// Kill ...
func (r *runner) Kill() error {
	if r.command == nil || r.command.Process == nil {
		return nil
	}

	done := make(chan error)
	go func() {
		r.command.Wait() // nolint errcheck
		close(done)
	}()

	// Trying a "soft" kill first
	if runtime.GOOS == OSWindows {
		if err := r.command.Process.Kill(); err != nil {
			return err
		}
	} else if err := r.command.Process.Signal(os.Interrupt); err != nil {
		return err
	}

	// Wait for our process to die before we return or hard kill after 3 sec
	select {
	case <-time.After(3 * time.Second):
		if err := r.command.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill: %v", err)
		}
	case <-done:
	}

	r.command = nil
	return nil
}

// Exited ...
func (r *runner) Exited() bool {
	return r.command != nil && r.command.ProcessState != nil && r.command.ProcessState.Exited()
}

// Errors ...
func (r *runner) Errors() chan error {
	return r.errors
}

func (r *runner) runBin() error {
	r.command = exec.Command(r.bin, r.args...) // nolint gas
	stdout, err := r.command.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := r.command.StderrPipe()
	if err != nil {
		return err
	}

	// TODO: handle or log errors
	go io.Copy(r.writerStdout, stdout) // nolint errcheck
	go io.Copy(r.writerStderr, stderr) // nolint errcheck

	err = r.command.Start()
	if err != nil {
		return err
	}

	r.starttime = time.Now()

	// wait for exit errors
	go func() {
		r.errors <- r.command.Wait()
	}()

	return nil
}

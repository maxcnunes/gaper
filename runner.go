package gaper

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"
)

// OSWindows is used to check if current OS is a Windows
const OSWindows = "windows"

// os errors
var errFinished = errors.New("os: process already finished")

// Runner is a interface for the run process
type Runner interface {
	Run() (*exec.Cmd, error)
	Kill() error
	Errors() chan error
	Exited() bool
	ExitStatus(err error) int
}

type runner struct {
	bin          string
	args         []string
	writerStdout io.Writer
	writerStderr io.Writer
	command      *exec.Cmd
	starttime    time.Time
	errors       chan error
	end          chan bool // used internally by Kill to wait a process die
}

// NewRunner creates a new runner
func NewRunner(wStdout io.Writer, wStderr io.Writer, bin string, args []string) Runner {
	return &runner{
		bin:          bin,
		args:         args,
		writerStdout: wStdout,
		writerStderr: wStderr,
		starttime:    time.Now(),
		errors:       make(chan error),
		end:          make(chan bool),
	}
}

// Run executes the project binary
func (r *runner) Run() (*exec.Cmd, error) {
	logger.Info("Starting program")

	if r.command != nil && !r.Exited() {
		return r.command, nil
	}

	if err := r.runBin(); err != nil {
		return nil, fmt.Errorf("error running: %v", err)
	}

	return r.command, nil
}

// Kill the current process running for the Golang project
func (r *runner) Kill() error { // nolint gocyclo
	if r.command == nil || r.command.Process == nil {
		return nil
	}

	done := make(chan error)
	go func() {
		<-r.end
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
			errMsg := err.Error()
			// ignore error if the processed has been killed already
			if errMsg != errFinished.Error() && errMsg != os.ErrInvalid.Error() {
				return fmt.Errorf("failed to kill: %v", err)
			}
		}
	case <-done:
	}

	r.command = nil
	return nil
}

// Exited checks if the process has exited
func (r *runner) Exited() bool {
	return r.command != nil && r.command.ProcessState != nil && r.command.ProcessState.Exited()
}

// Errors get errors occurred during the build
func (r *runner) Errors() chan error {
	return r.errors
}

// ExitStatus resolves the exit status
func (r *runner) ExitStatus(err error) int {
	var exitStatus int
	if exiterr, ok := err.(*exec.ExitError); ok {
		if status, oks := exiterr.Sys().(syscall.WaitStatus); oks {
			exitStatus = status.ExitStatus()
		}
	}

	return exitStatus
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
		r.end <- true
	}()

	return nil
}

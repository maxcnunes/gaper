package gaper

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunnerSuccessRun(t *testing.T) {
	stdout := bytes.NewBufferString("")
	stderr := bytes.NewBufferString("")
	pArgs := []string{}
	bin := filepath.Join("testdata", "print-gaper")
	if runtime.GOOS == OSWindows {
		bin += ".bat"
	}

	runner := NewRunner(stdout, stderr, bin, pArgs)

	cmd, err := runner.Run()
	assert.Nil(t, err, "error running binary")
	assert.NotNil(t, cmd.Process, "process has not started")

	errCmd := <-runner.Errors()
	assert.Nil(t, errCmd, "async error running binary")
	assert.Contains(t, stdout.String(), "Gaper Test Message")
	assert.Equal(t, stderr.String(), "")
}

func TestRunnerSuccessKill(t *testing.T) {
	bin := filepath.Join("testdata", "print-gaper")
	if runtime.GOOS == OSWindows {
		bin += ".bat"
	}

	runner := NewRunner(os.Stdout, os.Stderr, bin, nil)

	_, err := runner.Run()
	assert.Nil(t, err, "error running binary")

	err = runner.Kill()
	assert.Nil(t, err, "error killing program")

	errCmd := <-runner.Errors()
	assert.NotNil(t, errCmd, "kill program")
}

func TestRunnerExitedNotStarted(t *testing.T) {
	runner := NewRunner(os.Stdout, os.Stderr, "", nil)
	assert.Equal(t, runner.Exited(), false)
}

func TestRunnerExitStatusNonExitError(t *testing.T) {
	runner := NewRunner(os.Stdout, os.Stderr, "", nil)
	err := errors.New("non exec.ExitError")
	assert.Equal(t, runner.ExitStatus(err), 0)
}

func testExit() {
	os.Exit(1)
}

func TestRunnerExitStatusExitError(t *testing.T) {
	if os.Getenv("TEST_EXIT") == "1" {
		testExit()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRunnerExitStatusExitError")
	cmd.Env = append(os.Environ(), "TEST_EXIT=1")
	err := cmd.Run()

	runner := NewRunner(os.Stdout, os.Stderr, "", nil)
	assert.Equal(t, runner.ExitStatus(err), 1)
}

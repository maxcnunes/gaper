package main

import (
	"bytes"
	"os"
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

	if runtime.GOOS == OSWindows {
		assert.Equal(t, "Gaper\r\n", stdout.String())
	} else {
		assert.Equal(t, "Gaper\n", stdout.String())
	}
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

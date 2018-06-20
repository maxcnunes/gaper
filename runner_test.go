package main

import (
	"bytes"
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

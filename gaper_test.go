package gaper

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/maxcnunes/gaper/testdata"
	"github.com/stretchr/testify/assert"
)

func TestGaperRunStopOnSGINT(t *testing.T) {
	args := &Config{
		BuildPath: filepath.Join("testdata", "server"),
	}

	chOSSiginal := make(chan os.Signal, 2)
	go func() {
		time.Sleep(1 * time.Second)
		chOSSiginal <- syscall.SIGINT
	}()

	err := Run(args, chOSSiginal)
	assert.NotNil(t, err, "build error")
	assert.Equal(t, "OS signal: interrupt", err.Error())
}

func TestGaperSetupConfigNoParams(t *testing.T) {
	cwd, _ := os.Getwd()
	args := &Config{}
	err := setupConfig(args)
	assert.Nil(t, err, "build error")
	assert.Equal(t, args.BuildPath, ".")
	assert.Equal(t, args.WorkingDirectory, cwd)
	assert.Equal(t, args.WatchItems, []string{"."})
}

func TestGaperBuildError(t *testing.T) {
	mockBuilder := new(testdata.MockBuilder)
	mockBuilder.On("Build").Return(errors.New("build-error"))
	mockRunner := new(testdata.MockRunner)
	mockWatcher := new(testdata.MockWacther)

	cfg := &Config{}

	chOSSiginal := make(chan os.Signal, 2)
	err := run(cfg, chOSSiginal, mockBuilder, mockRunner, mockWatcher)
	assert.NotNil(t, err, "build error")
	assert.Equal(t, "build error: build-error", err.Error())
}

func TestGaperRunError(t *testing.T) {
	mockBuilder := new(testdata.MockBuilder)
	mockBuilder.On("Build").Return(nil)
	mockRunner := new(testdata.MockRunner)
	mockRunner.On("Run").Return(nil, errors.New("runner-error"))
	mockWatcher := new(testdata.MockWacther)

	cfg := &Config{}

	chOSSiginal := make(chan os.Signal, 2)
	err := run(cfg, chOSSiginal, mockBuilder, mockRunner, mockWatcher)
	assert.NotNil(t, err, "runner error")
	assert.Equal(t, "run error: runner-error", err.Error())
}

func TestGaperWatcherError(t *testing.T) {
	mockBuilder := new(testdata.MockBuilder)
	mockBuilder.On("Build").Return(nil)

	mockRunner := new(testdata.MockRunner)
	cmd := &exec.Cmd{}
	runnerErrorsChan := make(chan error)
	mockRunner.On("Run").Return(cmd, nil)
	mockRunner.On("Errors").Return(runnerErrorsChan)

	mockWatcher := new(testdata.MockWacther)
	watcherErrorsChan := make(chan error)
	watcherEvetnsChan := make(chan string)
	mockWatcher.On("Errors").Return(watcherErrorsChan)
	mockWatcher.On("Events").Return(watcherEvetnsChan)

	dir := filepath.Join("testdata", "server")

	cfg := &Config{
		BinName:   "test-srv",
		BuildPath: dir,
	}

	go func() {
		time.Sleep(3 * time.Second)
		watcherErrorsChan <- errors.New("watcher-error")
	}()
	chOSSiginal := make(chan os.Signal, 2)
	err := run(cfg, chOSSiginal, mockBuilder, mockRunner, mockWatcher)
	assert.NotNil(t, err, "build error")
	assert.Equal(t, "error on watching files: watcher-error", err.Error())
	mockBuilder.AssertExpectations(t)
	mockRunner.AssertExpectations(t)
	mockWatcher.AssertExpectations(t)
}

func TestGaperProgramExit(t *testing.T) {
	testCases := []struct {
		name        string
		exitStatus  int
		noRestartOn string
		restart     bool
	}{
		{
			name:        "no restart on exit error with no-restart-on=error",
			exitStatus:  exitStatusError,
			noRestartOn: NoRestartOnError,
			restart:     false,
		},
		{
			name:        "no restart on exit success with no-restart-on=success",
			exitStatus:  exitStatusSuccess,
			noRestartOn: NoRestartOnSuccess,
			restart:     false,
		},
		{
			name:        "no restart on exit error with no-restart-on=exit",
			exitStatus:  exitStatusError,
			noRestartOn: NoRestartOnExit,
			restart:     false,
		},
		{
			name:        "no restart on exit success with no-restart-on=exit",
			exitStatus:  exitStatusSuccess,
			noRestartOn: NoRestartOnExit,
			restart:     false,
		},
		{
			name:       "restart on exit error with disabled no-restart-on",
			exitStatus: exitStatusError,
			restart:    true,
		},
		{
			name:       "restart on exit success with disabled no-restart-on",
			exitStatus: exitStatusSuccess,
			restart:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockBuilder := new(testdata.MockBuilder)
			mockBuilder.On("Build").Return(nil)

			mockRunner := new(testdata.MockRunner)
			cmd := &exec.Cmd{}
			runnerErrorsChan := make(chan error)
			mockRunner.On("Run").Return(cmd, nil)
			mockRunner.On("Kill").Return(nil)
			mockRunner.On("Errors").Return(runnerErrorsChan)
			mockRunner.On("ExitStatus").Return(tc.exitStatus)
			if tc.restart {
				mockRunner.On("Exited").Return(true)
			}

			mockWatcher := new(testdata.MockWacther)
			watcherErrorsChan := make(chan error)
			watcherEvetnsChan := make(chan string)
			mockWatcher.On("Errors").Return(watcherErrorsChan)
			mockWatcher.On("Events").Return(watcherEvetnsChan)

			dir := filepath.Join("testdata", "server")

			cfg := &Config{
				BinName:     "test-srv",
				BuildPath:   dir,
				NoRestartOn: tc.noRestartOn,
			}

			chOSSiginal := make(chan os.Signal, 2)
			go func() {
				time.Sleep(1 * time.Second)
				runnerErrorsChan <- errors.New("runner-error")
				time.Sleep(1 * time.Second)
				chOSSiginal <- syscall.SIGINT
			}()
			err := run(cfg, chOSSiginal, mockBuilder, mockRunner, mockWatcher)
			assert.NotNil(t, err, "build error")
			assert.Equal(t, "OS signal: interrupt", err.Error())
			mockBuilder.AssertExpectations(t)
			mockRunner.AssertExpectations(t)
			mockWatcher.AssertExpectations(t)
		})
	}
}

func TestGaperRestartExited(t *testing.T) {
	mockBuilder := new(testdata.MockBuilder)
	mockBuilder.On("Build").Return(nil)

	mockRunner := new(testdata.MockRunner)
	cmd := &exec.Cmd{}
	mockRunner.On("Run").Return(cmd, nil)
	mockRunner.On("Exited").Return(true)

	err := restart(mockBuilder, mockRunner)
	assert.Nil(t, err, "restart error")
	mockBuilder.AssertExpectations(t)
	mockRunner.AssertExpectations(t)
}

func TestGaperRestartNotExited(t *testing.T) {
	mockBuilder := new(testdata.MockBuilder)
	mockBuilder.On("Build").Return(nil)

	mockRunner := new(testdata.MockRunner)
	cmd := &exec.Cmd{}
	mockRunner.On("Run").Return(cmd, nil)
	mockRunner.On("Kill").Return(nil)
	mockRunner.On("Exited").Return(false)

	err := restart(mockBuilder, mockRunner)
	assert.Nil(t, err, "restart error")
	mockBuilder.AssertExpectations(t)
	mockRunner.AssertExpectations(t)
}

func TestGaperRestartNotExitedKillFail(t *testing.T) {
	mockBuilder := new(testdata.MockBuilder)

	mockRunner := new(testdata.MockRunner)
	mockRunner.On("Kill").Return(errors.New("kill-error"))
	mockRunner.On("Exited").Return(false)

	err := restart(mockBuilder, mockRunner)
	assert.NotNil(t, err, "restart error")
	assert.Equal(t, "kill error: kill-error", err.Error())
	mockBuilder.AssertExpectations(t)
	mockRunner.AssertExpectations(t)
}

func TestGaperRestartBuildFail(t *testing.T) {
	mockBuilder := new(testdata.MockBuilder)
	mockBuilder.On("Build").Return(errors.New("build-error"))

	mockRunner := new(testdata.MockRunner)
	mockRunner.On("Exited").Return(true)

	err := restart(mockBuilder, mockRunner)
	assert.NotNil(t, err, "restart error")
	assert.Equal(t, "build error: build-error", err.Error())
	mockBuilder.AssertExpectations(t)
	mockRunner.AssertExpectations(t)
}

func TestGaperRestartRunFail(t *testing.T) {
	mockBuilder := new(testdata.MockBuilder)
	mockBuilder.On("Build").Return(nil)

	mockRunner := new(testdata.MockRunner)
	cmd := &exec.Cmd{}
	mockRunner.On("Run").Return(cmd, errors.New("run-error"))
	mockRunner.On("Exited").Return(true)

	err := restart(mockBuilder, mockRunner)
	assert.NotNil(t, err, "restart error")
	assert.Equal(t, "run error: run-error", err.Error())
	mockBuilder.AssertExpectations(t)
	mockRunner.AssertExpectations(t)
}

package testdata

import (
	"os/exec"

	"github.com/stretchr/testify/mock"
)

// MockBuilder ...
type MockBuilder struct {
	mock.Mock
}

// Build ...
func (m *MockBuilder) Build() error {
	args := m.Called()
	return args.Error(0)
}

// Binary ...
func (m *MockBuilder) Binary() string {
	args := m.Called()
	return args.String(0)
}

// MockRunner ...
type MockRunner struct {
	mock.Mock
}

// Run ...
func (m *MockRunner) Run() (*exec.Cmd, error) {
	args := m.Called()
	cmdArg := args.Get(0)
	if cmdArg == nil {
		return nil, args.Error(1)
	}

	return cmdArg.(*exec.Cmd), args.Error(1)
}

// Kill ...
func (m *MockRunner) Kill() error {
	args := m.Called()
	return args.Error(0)
}

// Errors ...
func (m *MockRunner) Errors() chan error {
	args := m.Called()
	return args.Get(0).(chan error)
}

// Exited ...
func (m *MockRunner) Exited() bool {
	args := m.Called()
	return args.Bool(0)
}

// ExitStatus ...
func (m *MockRunner) ExitStatus(err error) int {
	args := m.Called()
	return args.Int(0)
}

// MockWacther ...
type MockWacther struct {
	mock.Mock
}

// Watch ...
func (m *MockWacther) Watch() {}

// Events ...
func (m *MockWacther) Events() chan string {
	args := m.Called()
	return args.Get(0).(chan string)
}

// Errors ...
func (m *MockWacther) Errors() chan error {
	args := m.Called()
	return args.Get(0).(chan error)
}

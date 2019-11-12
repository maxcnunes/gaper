package gaper

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuilderSuccessBuild(t *testing.T) {
	bArgs := []string{}
	bin := resolveBinNameByOS("srv")
	dir := filepath.Join("testdata", "server")
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("couldn't get current working directory: %v", err)
	}

	b := NewBuilder(dir, bin, wd, bArgs)
	err = b.Build()
	assert.Nil(t, err, "build error")

	file, err := os.Open(filepath.Join(wd, bin))
	if err != nil {
		t.Fatalf("couldn't open open built binary: %v", err)
	}
	assert.NotNil(t, file, "binary not written properly")
}

func TestBuilderFailureBuild(t *testing.T) {
	bArgs := []string{}
	bin := "srv"
	dir := filepath.Join("testdata", "build-failure")
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("couldn't get current working directory: %v", err)
	}

	b := NewBuilder(dir, bin, wd, bArgs)
	err = b.Build()
	assert.NotNil(t, err, "build error")
	assert.Equal(t, err.Error(), "build failed with exit status 2\n"+
		"# github.com/maxcnunes/gaper/testdata/build-failure\n"+
		"./main.go:4:6: func main must have no arguments and no return values\n"+
		"./main.go:5:1: missing return at end of function\n")
}

func TestBuilderDefaultBinName(t *testing.T) {
	bin := ""
	dir := filepath.Join("testdata", "server")
	wd := "/src/projects/project-name"
	b := NewBuilder(dir, bin, wd, nil)
	assert.Equal(t, b.Binary(), resolveBinNameByOS("project-name"))
}

func resolveBinNameByOS(name string) string {
	if runtime.GOOS == OSWindows {
		name += ".exe"
	}
	return name
}

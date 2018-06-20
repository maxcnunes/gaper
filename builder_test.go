package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuilderSuccessBuild(t *testing.T) {
	bArgs := []string{}
	bin := "srv"
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

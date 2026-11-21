package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestVersionFlagPrintsTheVersionAndExits(t *testing.T) {
	var stdout, stderr bytes.Buffer

	if err := run(context.Background(), []string{"-version"}, &stdout, &stderr); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	if got := strings.TrimSpace(stdout.String()); got != version {
		t.Errorf("printed version %q, want %q", got, version)
	}
}

func TestIconDirPrefersTheEnvironment(t *testing.T) {
	t.Setenv(iconDirVariable, "/tmp/my-icons")

	if got := iconDir(); got != "/tmp/my-icons" {
		t.Errorf("iconDir() = %q, want the %s directory", got, iconDirVariable)
	}
}

func TestIconDirFallsBackToTheWorkingDirectoryDefault(t *testing.T) {
	t.Setenv(iconDirVariable, "")

	if got := iconDir(); got != defaultIconDir {
		t.Errorf("iconDir() = %q, want %q", got, defaultIconDir)
	}
}

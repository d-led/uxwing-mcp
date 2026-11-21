package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
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

func TestInstallCommandRegistersTheServerInTheWorkspaceConfig(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv(iconDirVariable, filepath.Join(dir, "icons"))

	var stdout, stderr bytes.Buffer
	if err := run(context.Background(), []string{"install", "--client", "workspace"}, &stdout, &stderr); err != nil {
		t.Fatalf("install returned error: %v (%s)", err, stderr.String())
	}

	raw, err := os.ReadFile(filepath.Join(dir, ".vscode", "mcp.json"))
	if err != nil {
		t.Fatalf("read workspace config: %v", err)
	}
	config := string(raw)
	for _, want := range []string{`"uxwing"`, iconDirVariable, filepath.Join(dir, "icons")} {
		if !strings.Contains(config, want) {
			t.Errorf("workspace config %s does not mention %q", config, want)
		}
	}
	if !strings.Contains(stdout.String(), filepath.Join(".vscode", "mcp.json")) {
		t.Errorf("install said %q, want the config path", stdout.String())
	}
}

func TestInstallCommandRejectsAnUnknownClient(t *testing.T) {
	var stdout, stderr bytes.Buffer

	if err := run(context.Background(), []string{"install", "--client", "cursor"}, &stdout, &stderr); err == nil {
		t.Error("install accepted an unknown client")
	}
}

func TestUninstallCommandRemovesTheServerAgain(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv(iconDirVariable, filepath.Join(dir, "icons"))

	var stdout, stderr bytes.Buffer
	if err := run(context.Background(), []string{"install", "--client", "workspace"}, &stdout, &stderr); err != nil {
		t.Fatalf("install returned error: %v", err)
	}
	stdout.Reset()
	if err := run(context.Background(), []string{"uninstall", "--client", "workspace"}, &stdout, &stderr); err != nil {
		t.Fatalf("uninstall returned error: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, ".vscode", "mcp.json"))
	if err != nil {
		t.Fatalf("read workspace config: %v", err)
	}
	if strings.Contains(string(raw), "uxwing-mcp") || strings.Contains(string(raw), `"uxwing"`) {
		t.Errorf("config still registers the server: %s", raw)
	}
	if !strings.Contains(stdout.String(), "removed") {
		t.Errorf("uninstall said %q, want a removal message", stdout.String())
	}
}

func TestAbsoluteIconDirResolvesRelativePaths(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	got, err := absoluteIconDir(filepath.Join("assets", "icons"))
	if err != nil {
		t.Fatalf("absoluteIconDir returned error: %v", err)
	}
	if want := filepath.Join(dir, "assets", "icons"); got != want {
		t.Errorf("absoluteIconDir() = %q, want %q", got, want)
	}
}

package install

import (
	"path/filepath"
	"testing"
)

const home = "/home/user"

func TestConfigPathFindsTheVSCodeUserProfile(t *testing.T) {
	for goos, want := range map[string]string{
		"darwin":  filepath.Join(home, "Library", "Application Support", "Code", "User", "mcp.json"),
		"linux":   filepath.Join(home, ".config", "Code", "User", "mcp.json"),
		"windows": filepath.Join(home, "AppData", "Roaming", "Code", "User", "mcp.json"),
	} {
		t.Setenv("APPDATA", "")
		got, err := ConfigPath(TargetVSCodeUser, home, goos)
		if err != nil {
			t.Fatalf("ConfigPath(vscode, %s) returned error: %v", goos, err)
		}
		if got != want {
			t.Errorf("ConfigPath(vscode, %s) = %q, want %q", goos, got, want)
		}
	}
}

func TestConfigPathFindsTheClaudeDesktopConfig(t *testing.T) {
	for goos, want := range map[string]string{
		"darwin":  filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"),
		"linux":   filepath.Join(home, ".config", "Claude", "claude_desktop_config.json"),
		"windows": filepath.Join(home, "AppData", "Roaming", "Claude", "claude_desktop_config.json"),
	} {
		t.Setenv("APPDATA", "")
		got, err := ConfigPath(TargetClaudeDesktop, home, goos)
		if err != nil {
			t.Fatalf("ConfigPath(claude, %s) returned error: %v", goos, err)
		}
		if got != want {
			t.Errorf("ConfigPath(claude, %s) = %q, want %q", goos, got, want)
		}
	}
}

func TestConfigPathUsesTheWorkspaceFolder(t *testing.T) {
	got, err := ConfigPath(TargetVSCodeWorkspace, home, "darwin")
	if err != nil {
		t.Fatalf("ConfigPath(workspace) returned error: %v", err)
	}
	if want := filepath.Join(".vscode", "mcp.json"); got != want {
		t.Errorf("ConfigPath(workspace) = %q, want %q", got, want)
	}
}

func TestConfigPathPrefersAppDataOnWindows(t *testing.T) {
	t.Setenv("APPDATA", "/appdata")

	got, err := ConfigPath(TargetVSCodeUser, home, "windows")
	if err != nil {
		t.Fatalf("ConfigPath returned error: %v", err)
	}
	if want := filepath.Join("/appdata", "Code", "User", "mcp.json"); got != want {
		t.Errorf("ConfigPath(vscode, windows) = %q, want %q", got, want)
	}
}

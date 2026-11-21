package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var uxwing = Server{
	Name:    "uxwing",
	Command: "/opt/bin/uxwing-mcp",
	Env:     map[string]string{"UXWING_ICON_DIR": "/tmp/icons"},
}

const otherServers = `{
  "servers": {
    "other": {"command": "other-server", "type": "stdio"}
  },
  "inputs": []
}
`

func newConfig(t *testing.T, existing string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "mcp.json")
	if existing != "" {
		if err := os.WriteFile(path, []byte(existing), 0o600); err != nil {
			t.Fatalf("write existing config: %v", err)
		}
	}
	return path
}

func readConfigFile(t *testing.T, path string) map[string]any {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatalf("config is not JSON: %v", err)
	}
	return root
}

func serverEntry(t *testing.T, root map[string]any, section, name string) map[string]any {
	t.Helper()

	servers, _ := root[section].(map[string]any)
	entry, ok := servers[name].(map[string]any)
	if !ok {
		t.Fatalf("%q is not registered under %q: %v", name, section, root)
	}
	return entry
}

func TestInstallRegistersAStdioServerNextToTheExistingOnes(t *testing.T) {
	path := newConfig(t, otherServers)

	changed, err := InstallAt(TargetVSCodeUser, path, uxwing)
	if err != nil {
		t.Fatalf("InstallAt returned error: %v", err)
	}
	if !changed {
		t.Fatal("InstallAt reported no change although the server was missing")
	}

	root := readConfigFile(t, path)
	entry := serverEntry(t, root, "servers", "uxwing")
	if entry["command"] != uxwing.Command || entry["type"] != "stdio" {
		t.Errorf("registered %v, want a stdio entry running %q", entry, uxwing.Command)
	}
	env, _ := entry["env"].(map[string]any)
	if env["UXWING_ICON_DIR"] != "/tmp/icons" {
		t.Errorf("registered env %v, want the icon directory", entry["env"])
	}
	serverEntry(t, root, "servers", "other")
	if _, ok := root["inputs"]; !ok {
		t.Error("install dropped the other top level keys of the config")
	}
}

func TestInstallWritesClaudeDesktopEntriesWithoutTheTypeField(t *testing.T) {
	path := newConfig(t, "")

	if _, err := InstallAt(TargetClaudeDesktop, path, uxwing); err != nil {
		t.Fatalf("InstallAt returned error: %v", err)
	}

	entry := serverEntry(t, readConfigFile(t, path), "mcpServers", "uxwing")
	if _, ok := entry["type"]; ok {
		t.Errorf("registered %v, want no type field for Claude Desktop", entry)
	}
	if entry["command"] != uxwing.Command {
		t.Errorf("registered command %v, want %q", entry["command"], uxwing.Command)
	}
}

func TestInstallWithoutAnIconDirectoryWritesNoEnv(t *testing.T) {
	path := newConfig(t, "")

	if _, err := InstallAt(TargetVSCodeUser, path, Server{Name: "uxwing", Command: "uxwing-mcp"}); err != nil {
		t.Fatalf("InstallAt returned error: %v", err)
	}

	if entry := serverEntry(t, readConfigFile(t, path), "servers", "uxwing"); entry["env"] != nil {
		t.Errorf("registered env %v, want none", entry["env"])
	}
}

func TestInstallTwiceLeavesTheFileAlone(t *testing.T) {
	path := newConfig(t, otherServers)
	if _, err := InstallAt(TargetVSCodeUser, path, uxwing); err != nil {
		t.Fatalf("first InstallAt returned error: %v", err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	changed, err := InstallAt(TargetVSCodeUser, path, uxwing)
	if err != nil {
		t.Fatalf("second InstallAt returned error: %v", err)
	}
	if changed {
		t.Error("InstallAt reported a change although the entry was already there")
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if string(before) != string(after) {
		t.Error("InstallAt rewrote an unchanged config")
	}
}

func TestInstallCreatesTheConfigDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Code", "User", "mcp.json")

	if _, err := InstallAt(TargetVSCodeUser, path, uxwing); err != nil {
		t.Fatalf("InstallAt returned error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("InstallAt did not create %s: %v", path, err)
	}
}

func TestInstallRefusesToOverwriteAConfigItCannotRead(t *testing.T) {
	path := newConfig(t, "{ this is not json")

	if _, err := InstallAt(TargetVSCodeUser, path, uxwing); err == nil {
		t.Fatal("InstallAt accepted a config it cannot parse")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(raw), "not json") {
		t.Error("InstallAt damaged the config it could not parse")
	}
}

func TestUninstallRemovesOnlyTheNamedServer(t *testing.T) {
	path := newConfig(t, otherServers)
	if _, err := InstallAt(TargetVSCodeUser, path, uxwing); err != nil {
		t.Fatalf("InstallAt returned error: %v", err)
	}

	changed, err := UninstallAt(TargetVSCodeUser, path, "uxwing")
	if err != nil {
		t.Fatalf("UninstallAt returned error: %v", err)
	}
	if !changed {
		t.Fatal("UninstallAt reported no change although the entry was there")
	}

	root := readConfigFile(t, path)
	serverEntry(t, root, "servers", "other")
	if servers, _ := root["servers"].(map[string]any); servers["uxwing"] != nil {
		t.Errorf("servers still hold %v after uninstall", servers["uxwing"])
	}
}

func TestUninstallLeavesAConfigWithoutTheServerAlone(t *testing.T) {
	path := newConfig(t, otherServers)

	changed, err := UninstallAt(TargetVSCodeUser, path, "uxwing")
	if err != nil {
		t.Fatalf("UninstallAt returned error: %v", err)
	}
	if changed {
		t.Error("UninstallAt reported a change although the server was not registered")
	}
}

func TestUninstallDropsTheSectionWhenItBecomesEmpty(t *testing.T) {
	path := newConfig(t, "")
	if _, err := InstallAt(TargetClaudeDesktop, path, uxwing); err != nil {
		t.Fatalf("InstallAt returned error: %v", err)
	}

	if _, err := UninstallAt(TargetClaudeDesktop, path, "uxwing"); err != nil {
		t.Fatalf("UninstallAt returned error: %v", err)
	}

	if _, ok := readConfigFile(t, path)["mcpServers"]; ok {
		t.Error("uninstall left an empty mcpServers section behind")
	}
}

func TestInstallFindsTheWorkspaceConfigOnItsOwn(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	path, changed, err := Install(TargetVSCodeWorkspace, uxwing)
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if want := filepath.Join(".vscode", "mcp.json"); path != want {
		t.Errorf("Install wrote %q, want %q", path, want)
	}
	if !changed {
		t.Error("Install reported no change although it created the config")
	}
	if _, err := os.Stat(filepath.Join(dir, ".vscode", "mcp.json")); err != nil {
		t.Errorf("Install did not write the workspace config: %v", err)
	}
}

func TestUninstallFindsTheWorkspaceConfigOnItsOwn(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, _, err := Install(TargetVSCodeWorkspace, uxwing); err != nil {
		t.Fatalf("Install returned error: %v", err)
	}

	path, changed, err := Uninstall(TargetVSCodeWorkspace, "uxwing")
	if err != nil {
		t.Fatalf("Uninstall returned error: %v", err)
	}
	if !changed {
		t.Errorf("Uninstall reported no change for %s", path)
	}
}

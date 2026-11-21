// Package install registers the uxwing MCP server with an MCP client: the
// VS Code user profile, the workspace's .vscode/mcp.json, or Claude Desktop.
// It merges with the servers already in the config file instead of replacing
// them.
package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Target selects which client config to write.
type Target string

const (
	// TargetVSCodeUser is the VS Code user profile: available in all workspaces.
	TargetVSCodeUser Target = "vscode"
	// TargetVSCodeWorkspace is the .vscode/mcp.json of the current folder.
	TargetVSCodeWorkspace Target = "workspace"
	// TargetClaudeDesktop is the Claude Desktop app config.
	TargetClaudeDesktop Target = "claude"
)

// ParseTarget maps a command line value onto a Target.
func ParseTarget(raw string) (Target, error) {
	target := Target(strings.ToLower(strings.TrimSpace(raw)))
	switch target {
	case TargetVSCodeUser, TargetVSCodeWorkspace, TargetClaudeDesktop:
		return target, nil
	default:
		return "", fmt.Errorf("unknown client %q: use vscode, workspace or claude", raw)
	}
}

// Server is the stdio server entry to write.
type Server struct {
	Name    string
	Command string
	Env     map[string]string
}

// ConfigPath locates the config file of a target. home and goos are arguments
// so the platform rules can be tested without touching a real home directory.
func ConfigPath(target Target, home, goos string) (string, error) {
	switch target {
	case TargetVSCodeUser:
		return filepath.Join(vscodeUserDir(home, goos), "mcp.json"), nil
	case TargetVSCodeWorkspace:
		return filepath.Join(".vscode", "mcp.json"), nil
	case TargetClaudeDesktop:
		return claudeConfigPath(home, goos), nil
	default:
		return "", fmt.Errorf("unknown client %q", target)
	}
}

// Install registers the server in the client's config and returns the path it
// wrote.
func Install(target Target, server Server) (string, bool, error) {
	path, err := configPath(target)
	if err != nil {
		return "", false, err
	}
	changed, err := InstallAt(target, path, server)
	return path, changed, err
}

// InstallAt registers the server in the config file at path.
func InstallAt(target Target, path string, server Server) (bool, error) {
	root, err := readConfig(path)
	if err != nil {
		return false, err
	}

	known, ok := servers(root, target)
	entry := entryFor(target, server)
	if !ok || !jsonEqual(known[server.Name], entry) {
		known[server.Name] = entry
		root[sectionKey(target)] = known
		return true, writeConfig(path, root)
	}
	return false, nil
}

// Uninstall removes the server from the config of the client and returns the
// path it touched.
func Uninstall(target Target, name string) (string, bool, error) {
	path, err := configPath(target)
	if err != nil {
		return "", false, err
	}
	changed, err := UninstallAt(target, path, name)
	return path, changed, err
}

// UninstallAt removes the server from the config file at path.
func UninstallAt(target Target, path, name string) (bool, error) {
	root, err := readConfig(path)
	if err != nil {
		return false, err
	}

	known, ok := servers(root, target)
	if _, registered := known[name]; !ok || !registered {
		return false, nil
	}

	delete(known, name)
	if len(known) == 0 {
		delete(root, sectionKey(target))
	} else {
		root[sectionKey(target)] = known
	}
	return true, writeConfig(path, root)
}

func configPath(target Target) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return ConfigPath(target, home, runtime.GOOS)
}

// sectionKey is the JSON property that holds the server map.
func sectionKey(target Target) string {
	if target == TargetClaudeDesktop {
		return "mcpServers"
	}
	return "servers"
}

// servers is the writable server map of a config, which InstallAt and
// UninstallAt edit in place.
func servers(root map[string]any, target Target) (map[string]any, bool) {
	switch known := root[sectionKey(target)].(type) {
	case map[string]any:
		return known, true
	default:
		return map[string]any{}, false
	}
}

func entryFor(target Target, server Server) map[string]any {
	entry := map[string]any{"command": server.Command}
	if target != TargetClaudeDesktop {
		entry["type"] = "stdio"
	}
	if len(server.Env) > 0 {
		env := make(map[string]any, len(server.Env))
		for key, value := range server.Env {
			env[key] = value
		}
		entry["env"] = env
	}
	return entry
}

func readConfig(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	root := map[string]any{}
	if len(raw) == 0 {
		return root, nil
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("%s is not a JSON object: %w", path, err)
	}
	return root, nil
}

func writeConfig(path string, root map[string]any) error {
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	if err := os.WriteFile(path, append(out, '\n'), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func jsonEqual(a, b any) bool {
	left, err := json.Marshal(a)
	if err != nil {
		return false
	}
	right, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(left) == string(right)
}

func vscodeUserDir(home, goos string) string {
	switch goos {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Code", "User")
	case "windows":
		return filepath.Join(appDataDir(home), "Code", "User")
	default:
		return filepath.Join(home, ".config", "Code", "User")
	}
}

func claudeConfigPath(home, goos string) string {
	switch goos {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	case "windows":
		return filepath.Join(appDataDir(home), "Claude", "claude_desktop_config.json")
	default:
		return filepath.Join(home, ".config", "Claude", "claude_desktop_config.json")
	}
}

func appDataDir(home string) string {
	if appData := os.Getenv("APPDATA"); appData != "" {
		return appData
	}
	return filepath.Join(home, "AppData", "Roaming")
}

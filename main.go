// Command uxwing-mcp serves UXWing icons over MCP on stdin and stdout.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/d-led/uxwing-mcp/internal/install"
	"github.com/d-led/uxwing-mcp/internal/mcpserver"
	"github.com/d-led/uxwing-mcp/internal/uxwing"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// version is set at build time; see scripts/build.sh and .goreleaser.yaml.
var version = "dev"

const (
	uxwingSite      = "https://uxwing.com"
	defaultIconDir  = "icons"
	iconDirVariable = "UXWING_ICON_DIR"
	serverName      = "uxwing"
	clientFlagHelp  = "client config to write: vscode, workspace or claude"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "uxwing-mcp: %v\n", err)
		os.Exit(1)
	}
}

// run dispatches the subcommands and otherwise serves MCP over stdin and
// stdout until the client disconnects.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) > 0 {
		switch args[0] {
		case "install":
			return runInstall(args[1:], stdout, stderr)
		case "uninstall":
			return runUninstall(args[1:], stdout, stderr)
		}
	}
	return serve(ctx, args, stdout, stderr)
}

func serve(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("uxwing-mcp", flag.ContinueOnError)
	flags.SetOutput(stderr)
	showVersion := flags.Bool("version", false, "print the version and exit")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		_, err := fmt.Fprintln(stdout, version)
		return err
	}

	client, err := uxwing.NewClient(uxwing.NewHTTPGetter(), uxwingSite)
	if err != nil {
		return err
	}

	server := mcpserver.NewServer(client, uxwing.NewStore(), iconDir(), version)
	return server.Run(ctx, &mcp.StdioTransport{})
}

// runInstall writes this binary into an MCP client's config.
func runInstall(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("install", flag.ContinueOnError)
	flags.SetOutput(stderr)
	client := flags.String("client", string(install.TargetVSCodeUser), clientFlagHelp)
	name := flags.String("name", serverName, "server name in the config")
	dir := flags.String("dir", "", "icon directory to record (default the "+iconDirVariable+" value)")
	if err := flags.Parse(args); err != nil {
		return err
	}

	target, err := install.ParseTarget(*client)
	if err != nil {
		return err
	}
	iconsDir, err := absoluteIconDir(*dir)
	if err != nil {
		return err
	}
	command, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate this executable: %w", err)
	}

	path, changed, err := install.Install(target, install.Server{
		Name:    *name,
		Command: command,
		Env:     map[string]string{iconDirVariable: iconsDir},
	})
	if err != nil {
		return err
	}
	return report(stdout, path, changed, fmt.Sprintf("registered %q", *name), fmt.Sprintf("%q was already registered", *name))
}

// runUninstall removes this server from an MCP client's config.
func runUninstall(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	flags.SetOutput(stderr)
	client := flags.String("client", string(install.TargetVSCodeUser), clientFlagHelp)
	name := flags.String("name", serverName, "server name in the config")
	if err := flags.Parse(args); err != nil {
		return err
	}

	target, err := install.ParseTarget(*client)
	if err != nil {
		return err
	}

	path, changed, err := install.Uninstall(target, *name)
	if err != nil {
		return err
	}
	return report(stdout, path, changed, fmt.Sprintf("removed %q", *name), fmt.Sprintf("%q was not registered", *name))
}

// report says what happened to the config file.
func report(stdout io.Writer, path string, changed bool, done, unchanged string) error {
	message := unchanged
	if changed {
		message = done
	}
	_, err := fmt.Fprintf(stdout, "%s in %s\n", message, path)
	return err
}

// absoluteIconDir makes the recorded download directory independent of the
// directory the client happens to start the server in.
func absoluteIconDir(requested string) (string, error) {
	dir := requested
	if dir == "" {
		dir = iconDir()
	}
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve icon directory %q: %w", dir, err)
	}
	return absolute, nil
}

// iconDir is where downloads land when a call names no directory.
func iconDir() string {
	if dir := os.Getenv(iconDirVariable); dir != "" {
		return dir
	}
	return defaultIconDir
}

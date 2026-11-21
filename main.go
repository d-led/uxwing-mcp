// Command uxwing-mcp serves UXWing icons over MCP on stdin and stdout.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"

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
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "uxwing-mcp: %v\n", err)
		os.Exit(1)
	}
}

// run serves MCP over stdin and stdout until the client disconnects.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
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

// iconDir is where downloads land when a call names no directory.
func iconDir() string {
	if dir := os.Getenv(iconDirVariable); dir != "" {
		return dir
	}
	return defaultIconDir
}

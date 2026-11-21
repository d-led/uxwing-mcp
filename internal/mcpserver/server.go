// Package mcpserver exposes uxwing.com icons as MCP tools: search with
// previews, inspect one icon with its variations, and download the files.
package mcpserver

import (
	"context"

	"github.com/d-led/uxwing-mcp/internal/uxwing"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Service is the icon source the tools read from.
type Service interface {
	Search(ctx context.Context, query string, limit int) ([]uxwing.Icon, error)
	Page(ctx context.Context, slug uxwing.Slug) (uxwing.Page, error)
	Download(ctx context.Context, slug uxwing.Slug, format uxwing.Format) ([]byte, uxwing.Asset, error)
	Preview(ctx context.Context, previewURL string) ([]byte, error)
}

// Saver stores a downloaded file and returns the path it wrote.
type Saver interface {
	Save(dir string, asset uxwing.Asset, data []byte) (string, error)
}

const instructions = `UXWing icon tools. search_icons lists free icons with their uxwing.com page
addresses, preview images and links the user can open. get_icon shows one icon: its page,
the formats UXWing serves and the variations of the same idea. download_icon writes the
SVG and PNG files into a directory. Icons are free under the UXWing licence
(https://uxwing.com/license/).`

// NewServer builds the MCP server with every icon tool registered. Downloads
// land in defaultDir unless a call asks for another directory.
func NewServer(service Service, saver Saver, defaultDir, version string) *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{Name: "uxwing-mcp", Version: version},
		&mcp.ServerOptions{Instructions: instructions},
	)

	tools := &tools{service: service, saver: saver, defaultDir: defaultDir}
	mcp.AddTool(server, searchTool(), tools.search)
	mcp.AddTool(server, getIconTool(), tools.getIcon)
	mcp.AddTool(server, downloadTool(), tools.download)
	return server
}

package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/d-led/uxwing-mcp/internal/uxwing"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	svgFile = []byte(`<svg xmlns="http://www.w3.org/2000/svg"><path d="M0 0h10v10H0z"/></svg>`)
	pngFile = []byte("\x89PNG\r\n\x1a\n512x512 preview")
)

// fakeService stands in for uxwing.com and records what the tools asked for.
type fakeService struct {
	icons      []uxwing.Icon
	page       uxwing.Page
	pageErr    error
	files      map[uxwing.Format][]byte
	preview    []byte
	previewErr error

	queries    []string
	limits     []int
	downloads  []uxwing.Format
	previewURL []string
}

func (f *fakeService) Search(_ context.Context, query string, limit int) ([]uxwing.Icon, error) {
	f.queries = append(f.queries, query)
	f.limits = append(f.limits, limit)
	if limit > len(f.icons) {
		limit = len(f.icons)
	}
	return f.icons[:limit], nil
}

func (f *fakeService) Page(context.Context, uxwing.Slug) (uxwing.Page, error) {
	return f.page, f.pageErr
}

func (f *fakeService) Download(_ context.Context, _ uxwing.Slug, format uxwing.Format) ([]byte, uxwing.Asset, error) {
	data, ok := f.files[format]
	if !ok {
		return nil, uxwing.Asset{}, errors.New("no such file: " + format.String())
	}
	f.downloads = append(f.downloads, format)
	return data, uxwing.Asset{Format: format, Filename: "robot-icon." + format.String(), URL: "https://uxwing.com/robot-icon." + format.String()}, nil
}

func (f *fakeService) Preview(_ context.Context, previewURL string) ([]byte, error) {
	f.previewURL = append(f.previewURL, previewURL)
	return f.preview, f.previewErr
}

func robotIcons() []uxwing.Icon {
	return []uxwing.Icon{
		{
			Slug:       "robot-icon",
			Title:      "Robot Icon",
			PageURL:    "https://uxwing.com/robot-icon/",
			PreviewURL: "https://uxwing.com/wp-content/themes/uxwing/download/robotics/robot-icon.png",
		},
		{
			Slug:    "robot-line-icon",
			Title:   "Robot Line Icon",
			PageURL: "https://uxwing.com/robot-line-icon/",
		},
	}
}

func robotPage() uxwing.Page {
	return uxwing.Page{
		Slug:  "robot-icon",
		Title: "Robot Icon",
		URL:   "https://uxwing.com/robot-icon/",
		Assets: []uxwing.Asset{
			{Format: uxwing.FormatPNG, Filename: "robot-icon.png", URL: "https://uxwing.com/robot-icon.png"},
			{Format: uxwing.FormatSVG, Filename: "robot-icon.svg", URL: "https://uxwing.com/robot-icon.svg"},
		},
		Related: []uxwing.Icon{{Slug: "robot-bot-icon", Title: "Robot Bot Icon", PageURL: "https://uxwing.com/robot-bot-icon/"}},
	}
}

func newTestSession(t *testing.T, service Service, dir string) *mcp.ClientSession {
	t.Helper()

	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	server := NewServer(service, uxwing.NewStore(), dir, "test")
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("connect server: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect client: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func callTool(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	return result
}

func structured[T any](t *testing.T, result *mcp.CallToolResult) T {
	t.Helper()

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode structured content %s: %v", raw, err)
	}
	return out
}

func imagesOf(result *mcp.CallToolResult) []*mcp.ImageContent {
	var images []*mcp.ImageContent
	for _, content := range result.Content {
		if image, ok := content.(*mcp.ImageContent); ok {
			images = append(images, image)
		}
	}
	return images
}

func textOf(result *mcp.CallToolResult) string {
	var parts []string
	for _, content := range result.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			parts = append(parts, text.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func TestServerTellsClientsToReadTheUXWingLicence(t *testing.T) {
	session := newTestSession(t, &fakeService{}, t.TempDir())

	instructions := session.InitializeResult().Instructions
	if !strings.Contains(instructions, "https://uxwing.com/license/") {
		t.Errorf("server instructions %q do not point at the UXWing licence", instructions)
	}
}

func TestEveryAnswerPointsAtTheUXWingLicence(t *testing.T) {
	service := &fakeService{
		icons:   robotIcons(),
		page:    robotPage(),
		preview: pngFile,
		files:   map[uxwing.Format][]byte{uxwing.FormatSVG: svgFile},
	}
	session := newTestSession(t, service, t.TempDir())

	invocations := map[string]map[string]any{
		"search_icons":  {"query": "robot"},
		"get_icon":      {"slug": "robot-icon"},
		"download_icon": {"slug": "robot-icon"},
	}
	for name, args := range invocations {
		if text := textOf(callTool(t, session, name, args)); !strings.Contains(text, "https://uxwing.com/license/") {
			t.Errorf("%s answered %q without the licence link", name, text)
		}
	}
}

func TestServerOffersSearchPreviewAndDownload(t *testing.T) {
	session := newTestSession(t, &fakeService{}, t.TempDir())

	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}

	found := make(map[string]bool)
	for _, tool := range tools.Tools {
		found[tool.Name] = true
	}
	for _, want := range []string{"search_icons", "get_icon", "download_icon"} {
		if !found[want] {
			t.Errorf("tool %q is missing from %v", want, tools.Tools)
		}
	}
}

func TestSearchShowsIconsTheirPagesAndPreviews(t *testing.T) {
	service := &fakeService{icons: robotIcons(), preview: pngFile}
	session := newTestSession(t, service, t.TempDir())

	result := callTool(t, session, "search_icons", map[string]any{"query": "robot"})

	output := structured[SearchOutput](t, result)
	if len(output.Icons) != 2 {
		t.Fatalf("search returned %d icons, want 2", len(output.Icons))
	}
	if images := imagesOf(result); len(images) != 1 || string(images[0].Data) != string(pngFile) {
		t.Errorf("search returned %d preview images, want the PNG preview of the icon with one", len(images))
	}
	if text := textOf(result); !strings.Contains(text, "https://uxwing.com/robot-icon/") {
		t.Errorf("search text %q does not give the icon's page address", text)
	}
	if service.queries[0] != "robot" || service.limits[0] != 15 {
		t.Errorf("search asked for query %q with limit %d, want robot with the default limit 15", service.queries[0], service.limits[0])
	}
}

func TestSearchRejectsAnEmptyQuery(t *testing.T) {
	session := newTestSession(t, &fakeService{}, t.TempDir())

	result := callTool(t, session, "search_icons", map[string]any{"query": "  "})

	if !result.IsError || !strings.Contains(textOf(result), "query") {
		t.Errorf("search of an empty query returned %+v, want an error about the query", result)
	}
}

func TestGetIconShowsFormatsVariationsAndPreview(t *testing.T) {
	service := &fakeService{page: robotPage(), preview: pngFile}
	session := newTestSession(t, service, t.TempDir())

	result := callTool(t, session, "get_icon", map[string]any{"slug": "robot-icon"})

	output := structured[GetIconOutput](t, result)
	if output.Title != "Robot Icon" || output.URL != "https://uxwing.com/robot-icon/" {
		t.Errorf("got icon %+v, want the robot icon with its page address", output)
	}
	if len(output.Formats) != 2 || len(output.Related) != 1 {
		t.Errorf("got formats %v and variations %v, want png, svg and one variation", output.Formats, output.Related)
	}
	if images := imagesOf(result); len(images) != 1 {
		t.Errorf("get_icon returned %d preview images, want one", len(images))
	}
	if service.previewURL[0] != "https://uxwing.com/robot-icon.png" {
		t.Errorf("previewed %q, want the icon's PNG download", service.previewURL[0])
	}
}

func TestGetIconRejectsAnAddressThatIsNotAnIconPage(t *testing.T) {
	session := newTestSession(t, &fakeService{}, t.TempDir())

	result := callTool(t, session, "get_icon", map[string]any{"slug": "https://example.com/robot-icon/"})

	if !result.IsError {
		t.Errorf("get_icon accepted a foreign address: %+v", result)
	}
}

func TestDownloadSavesEveryRequestedFormat(t *testing.T) {
	dir := t.TempDir()
	service := &fakeService{files: map[uxwing.Format][]byte{uxwing.FormatPNG: pngFile, uxwing.FormatSVG: svgFile}}
	session := newTestSession(t, service, dir)

	result := callTool(t, session, "download_icon", map[string]any{
		"slug":    "robot-icon",
		"formats": []any{"png", "svg"},
	})

	output := structured[DownloadOutput](t, result)
	if len(output.Files) != 2 {
		t.Fatalf("downloaded %d files, want the PNG and the SVG", len(output.Files))
	}
	for _, file := range output.Files {
		saved, err := os.ReadFile(filepath.Join(dir, file.Filename))
		if err != nil {
			t.Fatalf("read saved %s: %v", file.Filename, err)
		}
		if len(saved) != file.Bytes {
			t.Errorf("%s holds %d bytes, reported %d", file.Filename, len(saved), file.Bytes)
		}
	}
	if images := imagesOf(result); len(images) != 1 || string(images[0].Data) != string(pngFile) {
		t.Errorf("download returned %d images, want the downloaded PNG as preview", len(images))
	}
}

func TestDownloadDefaultsToSVG(t *testing.T) {
	dir := t.TempDir()
	service := &fakeService{files: map[uxwing.Format][]byte{uxwing.FormatSVG: svgFile}}
	session := newTestSession(t, service, dir)

	result := callTool(t, session, "download_icon", map[string]any{"slug": "robot-icon"})

	output := structured[DownloadOutput](t, result)
	if len(output.Files) != 1 || output.Files[0].Format != uxwing.FormatSVG {
		t.Fatalf("download returned %+v, want the SVG only", output.Files)
	}
	if _, err := os.Stat(filepath.Join(dir, "robot-icon.svg")); err != nil {
		t.Errorf("SVG was not saved into the default directory: %v", err)
	}
	if images := imagesOf(result); len(images) != 0 {
		t.Errorf("download returned %d images for an SVG only download", len(images))
	}
}

func TestDownloadRejectsAnUnknownFormat(t *testing.T) {
	service := &fakeService{files: map[uxwing.Format][]byte{uxwing.FormatSVG: svgFile}}
	session := newTestSession(t, service, t.TempDir())

	result := callTool(t, session, "download_icon", map[string]any{"slug": "robot-icon", "formats": []any{"gif"}})

	if !result.IsError || !strings.Contains(textOf(result), "svg") {
		t.Errorf("download of an unknown format returned %+v, want an error naming the served formats", result)
	}
	if len(service.downloads) != 0 {
		t.Errorf("download fetched %v although the format was rejected", service.downloads)
	}
}

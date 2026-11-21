package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/d-led/uxwing-mcp/internal/uxwing"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultLimit    = 15
	maxLimit        = 30
	defaultPreviews = 5
	maxPreviews     = 10
)

// SearchArgs asks for icons matching some keywords.
type SearchArgs struct {
	Query    string `json:"query" jsonschema:"what to search icons for, for example robot"`
	Limit    int    `json:"limit,omitempty" jsonschema:"how many icons to list, 1 to 30 (default 15)"`
	Previews *int   `json:"previews,omitempty" jsonschema:"how many preview images to attach, 0 to 10 (default 5)"`
}

// SearchOutput is the machine readable half of a search answer.
type SearchOutput struct {
	Icons []uxwing.Icon `json:"icons" jsonschema:"icons UXWing found, in result order"`
}

// GetIconArgs names one icon.
type GetIconArgs struct {
	Slug string `json:"slug" jsonschema:"icon slug or uxwing.com page address, for example robot-icon"`
}

// GetIconOutput describes one icon page.
type GetIconOutput struct {
	Slug    uxwing.Slug   `json:"slug"`
	Title   string        `json:"title" jsonschema:"human readable icon name"`
	URL     string        `json:"url" jsonschema:"uxwing.com page showing the icon"`
	Formats []string      `json:"formats" jsonschema:"formats UXWing serves for this icon"`
	Related []uxwing.Icon `json:"related" jsonschema:"variations of the same idea, for example black or outline drawings"`
}

// DownloadArgs asks for the files of one icon.
type DownloadArgs struct {
	Slug    string   `json:"slug" jsonschema:"icon slug or uxwing.com page address, for example robot-icon"`
	Formats []string `json:"formats,omitempty" jsonschema:"formats to download, svg and/or png (default svg)"`
	Dir     string   `json:"dir,omitempty" jsonschema:"directory that receives the files (default the server's icon directory)"`
}

// DownloadOutput lists the files that reached the disk.
type DownloadOutput struct {
	Slug  uxwing.Slug `json:"slug"`
	Files []SavedFile `json:"files" jsonschema:"files written to disk"`
}

// SavedFile is one file written by the download tool.
type SavedFile struct {
	Format   uxwing.Format `json:"format"`
	Filename string        `json:"filename"`
	Path     string        `json:"path"`
	Bytes    int           `json:"bytes"`
}

type tools struct {
	service    Service
	saver      Saver
	defaultDir string
}

func searchTool() *mcp.Tool {
	return &mcp.Tool{
		Name: "search_icons",
		Description: "Search uxwing.com for free icons. Returns each icon's uxwing.com page " +
			"address, a 512x512 PNG preview the user can look at, and the values to pass to " +
			"get_icon or download_icon.",
	}
}

func getIconTool() *mcp.Tool {
	return &mcp.Tool{
		Name: "get_icon",
		Description: "Show one uxwing.com icon: its page address, the formats UXWing serves, " +
			"a PNG preview, and the variations UXWing offers for the same idea.",
	}
}

func downloadTool() *mcp.Tool {
	return &mcp.Tool{
		Name: "download_icon",
		Description: "Download an uxwing.com icon's SVG and/or PNG files into a directory and " +
			"report the saved paths. Returns the PNG as a preview so the user sees what was saved.",
	}
}

func (t *tools) search(ctx context.Context, _ *mcp.CallToolRequest, args SearchArgs) (*mcp.CallToolResult, SearchOutput, error) {
	query := strings.TrimSpace(args.Query)
	if query == "" {
		return nil, SearchOutput{}, errors.New("a search query is required, for example robot")
	}

	icons, err := t.service.Search(ctx, query, clampLimit(args.Limit))
	if err != nil {
		return nil, SearchOutput{}, err
	}

	content := answer(describeIcons(icons))
	content = append(content, t.previews(ctx, icons, previewCount(args.Previews))...)
	return &mcp.CallToolResult{Content: content}, SearchOutput{Icons: icons}, nil
}

func (t *tools) getIcon(ctx context.Context, _ *mcp.CallToolRequest, args GetIconArgs) (*mcp.CallToolResult, GetIconOutput, error) {
	slug, err := uxwing.ParseSlug(args.Slug)
	if err != nil {
		return nil, GetIconOutput{}, err
	}

	page, err := t.service.Page(ctx, slug)
	if err != nil {
		return nil, GetIconOutput{}, err
	}

	output := GetIconOutput{
		Slug:    page.Slug,
		Title:   page.Title,
		URL:     page.URL,
		Formats: uxwing.FormatNames(page.Assets),
		Related: page.Related,
	}

	content := answer(describeIcon(output))
	if png, ok := uxwing.FindAsset(page.Assets, uxwing.FormatPNG); ok {
		content = append(content, t.preview(ctx, png.URL)...)
	}
	return &mcp.CallToolResult{Content: content}, output, nil
}

func (t *tools) download(ctx context.Context, _ *mcp.CallToolRequest, args DownloadArgs) (*mcp.CallToolResult, DownloadOutput, error) {
	slug, err := uxwing.ParseSlug(args.Slug)
	if err != nil {
		return nil, DownloadOutput{}, err
	}

	formats, err := parseFormats(args.Formats)
	if err != nil {
		return nil, DownloadOutput{}, err
	}

	saved, err := t.saveAll(ctx, slug, formats, dirOrDefault(args.Dir, t.defaultDir))
	if err != nil {
		return nil, DownloadOutput{}, err
	}

	output := DownloadOutput{Slug: slug, Files: savedFiles(saved)}
	content := answer(describeSaved(output))
	content = append(content, savedPNG(saved)...)
	return &mcp.CallToolResult{Content: content}, output, nil
}

// previews attaches PNG previews of the first icons. A preview that cannot be
// fetched is reported in the text instead of failing the search.
func (t *tools) previews(ctx context.Context, icons []uxwing.Icon, count int) []mcp.Content {
	content := make([]mcp.Content, 0, count)

	for _, icon := range icons[:min(count, len(icons))] {
		if icon.PreviewURL == "" {
			continue
		}
		content = append(content, t.preview(ctx, icon.PreviewURL)...)
	}

	return content
}

func (t *tools) preview(ctx context.Context, previewURL string) []mcp.Content {
	data, err := t.service.Preview(ctx, previewURL)
	if err != nil {
		return []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("preview unavailable at %s: %v", previewURL, err)}}
	}
	return []mcp.Content{&mcp.ImageContent{Data: data, MIMEType: "image/png"}}
}

type savedIcon struct {
	asset uxwing.Asset
	path  string
	data  []byte
}

func (t *tools) saveAll(ctx context.Context, slug uxwing.Slug, formats []uxwing.Format, dir string) ([]savedIcon, error) {
	saved := make([]savedIcon, 0, len(formats))

	for _, format := range formats {
		data, asset, err := t.service.Download(ctx, slug, format)
		if err != nil {
			return nil, err
		}
		path, err := t.saver.Save(dir, asset, data)
		if err != nil {
			return nil, err
		}
		saved = append(saved, savedIcon{asset: asset, path: path, data: data})
	}

	return saved, nil
}

func savedFiles(saved []savedIcon) []SavedFile {
	files := make([]SavedFile, len(saved))
	for i, icon := range saved {
		files[i] = SavedFile{
			Format:   icon.asset.Format,
			Filename: icon.asset.Filename,
			Path:     icon.path,
			Bytes:    len(icon.data),
		}
	}
	return files
}

// savedPNG hands the downloaded PNG back as a preview, so the user sees the
// file that was just written.
func savedPNG(saved []savedIcon) []mcp.Content {
	for _, icon := range saved {
		if icon.asset.Format == uxwing.FormatPNG {
			return []mcp.Content{&mcp.ImageContent{Data: icon.data, MIMEType: "image/png"}}
		}
	}
	return nil
}

func parseFormats(requested []string) ([]uxwing.Format, error) {
	if len(requested) == 0 {
		return []uxwing.Format{uxwing.FormatSVG}, nil
	}

	formats := make([]uxwing.Format, 0, len(requested))
	for _, raw := range requested {
		format, err := uxwing.ParseFormat(raw)
		if err != nil {
			return nil, err
		}
		formats = append(formats, format)
	}
	return formats, nil
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	return min(limit, maxLimit)
}

func previewCount(requested *int) int {
	if requested == nil {
		return defaultPreviews
	}
	return max(0, min(*requested, maxPreviews))
}

func dirOrDefault(dir, fallback string) string {
	if trimmed := strings.TrimSpace(dir); trimmed != "" {
		return trimmed
	}
	return fallback
}

func describeIcons(icons []uxwing.Icon) string {
	if len(icons) == 0 {
		return "No icons matched."
	}

	lines := make([]string, len(icons))
	for i, icon := range icons {
		lines[i] = fmt.Sprintf("%d. %s — %s", i+1, icon.Title, icon.PageURL)
	}
	return strings.Join(lines, "\n")
}

func describeIcon(output GetIconOutput) string {
	formats := "none"
	if len(output.Formats) > 0 {
		formats = strings.Join(output.Formats, ", ")
	}

	return fmt.Sprintf("%s (%s)\nPage: %s\nFormats: %s\nVariations:\n%s",
		output.Title, output.Slug, output.URL, formats, describeIcons(output.Related))
}

func describeSaved(output DownloadOutput) string {
	lines := make([]string, len(output.Files))
	for i, file := range output.Files {
		lines[i] = fmt.Sprintf("%s — %s, %d bytes", file.Path, file.Format, file.Bytes)
	}
	return strings.Join(lines, "\n")
}

// licenceNote ends every answer: the icons are not this tool's to give away,
// and their licence is what the user has to follow.
const licenceNote = "Icons belong to UXWing — licence: https://uxwing.com/license/"

// answer is the content of a result: the answer itself, then the licence note.
func answer(text string) []mcp.Content {
	return []mcp.Content{&mcp.TextContent{Text: text + "\n\n" + licenceNote}}
}

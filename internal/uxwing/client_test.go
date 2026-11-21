package uxwing

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeSite serves captured UXWing pages and payloads, rewriting the {{BASE}}
// placeholder so the pages point back at the test server.
type fakeSite struct {
	*httptest.Server
	content map[string][]byte
}

func newFakeSite(t *testing.T) *fakeSite {
	t.Helper()

	site := &fakeSite{content: make(map[string][]byte)}
	site.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := site.content[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, strings.ReplaceAll(string(body), "{{BASE}}", "http://"+r.Host))
	}))
	t.Cleanup(site.Close)
	return site
}

func (s *fakeSite) page(t *testing.T, path, fixture string) {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("testdata", fixture))
	if err != nil {
		t.Fatalf("read fixture %q: %v", fixture, err)
	}
	s.content[path] = raw
}

func (s *fakeSite) html(path, markup string) { s.content[path] = []byte(markup) }

func (s *fakeSite) file(path string, body []byte) { s.content[path] = body }

func (s *fakeSite) client(t *testing.T) *Client {
	t.Helper()

	client, err := NewClient(NewHTTPGetter(), s.URL)
	if err != nil {
		t.Fatalf("NewClient(%q): %v", s.URL, err)
	}
	return client
}

const (
	pngMagic = "\x89PNG\r\n\x1a\n"
	svgBody  = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><path d="M0 0h10v10H0z"/></svg>`
)

func TestSearchReturnsIconsFromTheResultsPage(t *testing.T) {
	site := newFakeSite(t)
	site.page(t, "/", "search.html")

	icons, err := site.client(t).Search(context.Background(), "robot", 10)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	if len(icons) != 3 {
		t.Fatalf("Search found %d icons, want 3: %v", len(icons), slugsOf(icons))
	}
	if icons[0].Title != "Robot Icon" {
		t.Errorf("first icon title = %q, want %q", icons[0].Title, "Robot Icon")
	}
	if want := site.URL + "/robot-icon/"; icons[0].PageURL != want {
		t.Errorf("first icon page URL = %q, want %q", icons[0].PageURL, want)
	}
}

func TestSearchSendsTheQueryToUXWing(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("s")
		_, _ = io.WriteString(w, `<html><body></body></html>`)
	}))
	t.Cleanup(srv.Close)
	client, err := NewClient(NewHTTPGetter(), srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := client.Search(context.Background(), "robot arm", 5); err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if gotQuery != "robot arm" {
		t.Errorf("UXWing received query %q, want %q", gotQuery, "robot arm")
	}
}

func TestSearchRejectsAnEmptyQueryOrZeroLimit(t *testing.T) {
	client := newFakeSite(t).client(t)

	if _, err := client.Search(context.Background(), "   ", 10); err == nil {
		t.Error("Search accepted an empty query")
	}
	if _, err := client.Search(context.Background(), "robot", 0); err == nil {
		t.Error("Search accepted a limit of zero")
	}
}

func TestSearchReportsAPageThatDoesNotLoad(t *testing.T) {
	client := newFakeSite(t).client(t)

	if _, err := client.Search(context.Background(), "robot", 10); err == nil {
		t.Error("Search found icons on a page that returned 404")
	}
}

func TestPageListsDownloadsAndVariations(t *testing.T) {
	site := newFakeSite(t)
	site.page(t, "/robot-icon/", "robot-icon.html")

	page, err := site.client(t).Page(context.Background(), "robot-icon")
	if err != nil {
		t.Fatalf("Page returned error: %v", err)
	}

	if want := site.URL + "/robot-icon/"; page.URL != want {
		t.Errorf("page URL = %q, want %q", page.URL, want)
	}
	if len(page.Assets) != 2 || page.Assets[0].Format != FormatPNG || page.Assets[1].Format != FormatSVG {
		t.Errorf("page assets = %v, want the PNG and SVG downloads", page.Assets)
	}
	if got := slugsOf(page.Related); len(got) != 2 || got[0] != "robot-bot-icon" {
		t.Errorf("page variations = %v, want the related icon cards", got)
	}
}

func TestPreviewReturnsThePNGImage(t *testing.T) {
	site := newFakeSite(t)
	site.file("/previews/robot-icon.png", []byte(pngMagic+"payload"))

	data, err := site.client(t).Preview(context.Background(), site.URL+"/previews/robot-icon.png")
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if !strings.HasPrefix(string(data), pngMagic) {
		t.Errorf("Preview returned %q, want the PNG bytes", data)
	}
}

func TestPreviewRejectsAPayloadThatIsNotAPNG(t *testing.T) {
	site := newFakeSite(t)
	site.file("/previews/robot-icon.png", []byte("<html>blocked</html>"))

	if _, err := site.client(t).Preview(context.Background(), site.URL+"/previews/robot-icon.png"); err == nil {
		t.Error("Preview accepted an HTML page as a PNG image")
	}
}

func TestDownloadReturnsTheSVGFile(t *testing.T) {
	site := newFakeSite(t)
	site.page(t, "/robot-icon/", "robot-icon.html")
	site.file("/wp-content/themes/uxwing/download/internet-network-technology/robot-icon.svg", []byte(svgBody))

	data, asset, err := site.client(t).Download(context.Background(), "robot-icon", FormatSVG)
	if err != nil {
		t.Fatalf("Download returned error: %v", err)
	}
	if string(data) != svgBody {
		t.Errorf("Download returned %q, want the SVG file", data)
	}
	if asset.Filename != "robot-icon.svg" {
		t.Errorf("Download reported file %q, want robot-icon.svg", asset.Filename)
	}
}

func TestDownloadFindsThePNGNextToTheSVG(t *testing.T) {
	site := newFakeSite(t)
	site.page(t, "/robot-icon/", "robot-icon-svg-only.html")
	site.file("/wp-content/themes/uxwing/download/internet-network-technology/robot-icon.png", []byte(pngMagic+"payload"))

	data, asset, err := site.client(t).Download(context.Background(), "robot-icon", FormatPNG)
	if err != nil {
		t.Fatalf("Download returned error: %v", err)
	}
	if !strings.HasPrefix(string(data), pngMagic) {
		t.Errorf("Download returned %q, want PNG bytes", data)
	}
	if asset.Filename != "robot-icon.png" {
		t.Errorf("Download reported file %q, want robot-icon.png", asset.Filename)
	}
}

func TestDownloadReturnsThePNGFile(t *testing.T) {
	site := newFakeSite(t)
	site.page(t, "/robot-icon/", "robot-icon.html")
	site.file("/wp-content/themes/uxwing/download/internet-network-technology/robot-icon.png", []byte(pngMagic+"payload"))

	data, asset, err := site.client(t).Download(context.Background(), "robot-icon", FormatPNG)
	if err != nil {
		t.Fatalf("Download returned error: %v", err)
	}
	if !strings.HasPrefix(string(data), pngMagic) {
		t.Errorf("Download returned %q, want PNG bytes", data)
	}
	if asset.Format != FormatPNG {
		t.Errorf("Download reported format %q, want png", asset.Format)
	}
}

func TestDownloadReportsAPageWithoutDownloads(t *testing.T) {
	site := newFakeSite(t)
	site.html("/robot-icon/", `<html><body><h1>Nothing here</h1></body></html>`)

	_, _, err := site.client(t).Download(context.Background(), "robot-icon", FormatSVG)
	if err == nil {
		t.Fatal("Download succeeded on a page without download buttons")
	}
	if !strings.Contains(err.Error(), "no downloads") {
		t.Errorf("error %q does not say the page offers no downloads", err)
	}
}

func TestDownloadRejectsAPayloadThatIsNotTheRequestedFormat(t *testing.T) {
	site := newFakeSite(t)
	site.page(t, "/robot-icon/", "robot-icon.html")
	site.file("/wp-content/themes/uxwing/download/internet-network-technology/robot-icon.svg", []byte("<html>blocked</html>"))

	if _, _, err := site.client(t).Download(context.Background(), "robot-icon", FormatSVG); err == nil {
		t.Error("Download accepted an HTML page as an SVG file")
	}
}

func TestNewClientRejectsABadBaseURL(t *testing.T) {
	if _, err := NewClient(NewHTTPGetter(), "://nope"); err == nil {
		t.Error("NewClient accepted an unparsable base URL")
	}
}

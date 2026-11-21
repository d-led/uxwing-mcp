package uxwing

import (
	"bytes"
	"os"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

// liveSite is what the {{BASE}} placeholder stands for; the fake site in the
// client tests rewrites it to the test server instead.
const liveSite = "https://uxwing.com"

func mustDocument(t *testing.T, fixture string) *goquery.Document {
	t.Helper()

	raw, err := os.ReadFile("testdata/" + fixture)
	if err != nil {
		t.Fatalf("read fixture %q: %v", fixture, err)
	}
	raw = bytes.ReplaceAll(raw, []byte("{{BASE}}"), []byte(liveSite))

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("parse fixture %q: %v", fixture, err)
	}
	return doc
}

func slugsOf(icons []Icon) []string {
	out := make([]string, len(icons))
	for i, icon := range icons {
		out[i] = icon.Slug.String()
	}
	return out
}

func TestSearchPageYieldsOnlyIconLinks(t *testing.T) {
	icons := parseIconCards(mustDocument(t, "search.html"), 10)

	want := []string{"robot-icon", "robot-line-icon", "chatbot-icon"}
	got := slugsOf(icons)
	if len(got) != len(want) {
		t.Fatalf("found icons %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("found icons %v, want %v", got, want)
		}
	}
}

func TestSearchPageNamesEachIcon(t *testing.T) {
	icons := parseIconCards(mustDocument(t, "search.html"), 10)

	if icons[0].Title != "Robot Icon" {
		t.Errorf("first icon title = %q, want the link title %q", icons[0].Title, "Robot Icon")
	}
	if icons[2].Title != "Chatbot Icon" {
		t.Errorf("untitled icon got title %q, want the image alt text %q", icons[2].Title, "Chatbot Icon")
	}
}

func TestSearchPageGivesEachIconItsPageAndPreview(t *testing.T) {
	icons := parseIconCards(mustDocument(t, "search.html"), 10)

	if want := "https://uxwing.com/robot-line-icon/"; icons[1].PageURL != want {
		t.Errorf("icon page URL = %q, want %q", icons[1].PageURL, want)
	}
	wantPreview := "https://uxwing.com/wp-content/themes/uxwing/download/internet-network-technology/robot-line-icon.png"
	if icons[1].PreviewURL != wantPreview {
		t.Errorf("icon preview URL = %q, want %q", icons[1].PreviewURL, wantPreview)
	}
}

func TestSearchPageStopsAtLimit(t *testing.T) {
	icons := parseIconCards(mustDocument(t, "search.html"), 2)

	if got := slugsOf(icons); len(got) != 2 || got[1] != "robot-line-icon" {
		t.Errorf("limit 2 returned %v, want the first two matches", got)
	}
}

func TestSearchPageCollapsesRepeatedLinks(t *testing.T) {
	icons := parseIconCards(mustDocument(t, "search.html"), 10)

	seen := make(map[string]bool)
	for _, icon := range icons {
		if seen[icon.Slug.String()] {
			t.Fatalf("icon %q appears twice in %v", icon.Slug, slugsOf(icons))
		}
		seen[icon.Slug.String()] = true
	}
}

func TestIconPageListsEveryDownload(t *testing.T) {
	assets := parseAssets(mustDocument(t, "robot-icon.html"))

	if len(assets) != 2 {
		t.Fatalf("found %d downloads, want the PNG and SVG buttons only: %v", len(assets), assets)
	}
	if assets[0].Format != FormatPNG || assets[0].Filename != "robot-icon.png" {
		t.Errorf("first download = %+v, want the PNG button", assets[0])
	}
	if want := liveSite + "/wp-content/themes/uxwing/download/internet-network-technology/robot-icon.svg"; assets[1].URL != want {
		t.Errorf("second download = %+v, want the absolute SVG file URL %q", assets[1], want)
	}
}

func TestIconPageOffersVariationsOfTheIcon(t *testing.T) {
	page := parsePage(mustDocument(t, "robot-icon.html"))

	if page.Title != "Robot Icon" {
		t.Errorf("page title = %q, want %q", page.Title, "Robot Icon")
	}
	want := []string{"robot-bot-icon", "robot-bot-black-icon"}
	if got := slugsOf(page.Related); len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("related icons = %v, want the variation cards %v", got, want)
	}
}

func TestIconPageOffersThePNGFileEvenWhenItsButtonIsHidden(t *testing.T) {
	page := parsePage(mustDocument(t, "robot-icon-svg-only.html"))

	png, ok := FindAsset(page.Assets, FormatPNG)
	if !ok {
		t.Fatalf("page offers %v, want the PNG file next to the SVG", page.Assets)
	}
	want := liveSite + "/wp-content/themes/uxwing/download/internet-network-technology/robot-icon.png"
	if png.URL != want || png.Filename != "robot-icon.png" {
		t.Errorf("PNG asset = %+v, want %+v", png, Asset{Format: FormatPNG, URL: want, Filename: "robot-icon.png"})
	}
}

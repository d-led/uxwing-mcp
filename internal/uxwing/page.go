package uxwing

import (
	"net/url"
	"path"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// maxRelatedIcons bounds the variations read from one icon page.
const maxRelatedIcons = 20

// iconCard matches icon cards on both page kinds: the search grid puts the
// class on the link, the related grid puts it on the wrapper.
const iconCard = "a.post-thumbnail[href], div.post-thumbnail a[href]"

// downloadButton is the anchor behind UXWing's Download SVG / Download PNG
// buttons; its download attribute carries the file name.
const downloadButton = "a.downloadicon[href][download]"

func parsePage(doc *goquery.Document) Page {
	return Page{
		Title:   pageTitle(doc),
		Assets:  withOtherFormats(parseAssets(doc)),
		Related: parseIconCards(doc, maxRelatedIcons),
	}
}

// pageTitle is the icon name without the marketing suffix UXWing appends to
// its headings, such as "Robot Icon SVG Vector".
func pageTitle(doc *goquery.Document) string {
	return strings.TrimSuffix(strings.TrimSpace(doc.Find("h1").First().Text()), " SVG Vector")
}

// withOtherFormats adds the missing format next to the one the page lists.
// UXWing keeps both files under the same name but currently comments the PNG
// button out, so the file is only reachable by name.
func withOtherFormats(assets []Asset) []Asset {
	complete := append(make([]Asset, 0, 2), assets...)

	for _, format := range []Format{FormatSVG, FormatPNG} {
		if _, ok := FindAsset(complete, format); ok {
			continue
		}
		if sibling, ok := FindAsset(complete, siblingFormat(format)); ok {
			complete = append(complete, sibling.asFormat(format))
		}
	}

	return complete
}

func siblingFormat(format Format) Format {
	if format == FormatSVG {
		return FormatPNG
	}
	return FormatSVG
}

func parseIconCards(doc *goquery.Document, limit int) []Icon {
	icons := make([]Icon, 0, limit)
	seen := make(map[Slug]bool)

	doc.Find(iconCard).EachWithBreak(func(_ int, card *goquery.Selection) bool {
		icon, ok := iconFromCard(card)
		if !ok || seen[icon.Slug] {
			return true
		}
		seen[icon.Slug] = true
		icons = append(icons, icon)
		return len(icons) < limit
	})

	return icons
}

func iconFromCard(card *goquery.Selection) (Icon, bool) {
	href := strings.TrimSpace(card.AttrOr("href", ""))
	slug, ok := slugFromHref(href)
	if !ok || !isAbsoluteURL(href) {
		return Icon{}, false
	}
	return Icon{
		Slug:       slug,
		Title:      titleOf(card, slug),
		PageURL:    href,
		PreviewURL: previewURL(card.Find("img").AttrOr("src", "")),
	}, true
}

func slugFromHref(href string) (Slug, bool) {
	parsed, err := url.Parse(href)
	if err != nil {
		return "", false
	}
	slug := Slug(lastPathSegment(parsed.Path))
	if !slugPattern.MatchString(slug.String()) {
		return "", false
	}
	return slug, true
}

func titleOf(card *goquery.Selection, slug Slug) string {
	candidates := []string{
		card.AttrOr("title", ""),
		card.Find("img").AttrOr("alt", ""),
		strings.ReplaceAll(slug.String(), "-", " "),
	}
	for _, candidate := range candidates {
		if title := strings.TrimSpace(candidate); title != "" {
			return title
		}
	}
	return slug.String()
}

// previewURL prefers the PNG version of a card image, because chat clients
// render PNG and not SVG.
func previewURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !isAbsoluteURL(parsed.String()) {
		return ""
	}
	if !strings.HasSuffix(parsed.Path, ".png") {
		parsed.Path = strings.TrimSuffix(parsed.Path, ".svg") + ".png"
	}
	return parsed.String()
}

func parseAssets(doc *goquery.Document) []Asset {
	assets := make([]Asset, 0, 2)

	doc.Find(downloadButton).Each(func(_ int, button *goquery.Selection) {
		if asset, ok := assetFromButton(button); ok {
			assets = append(assets, asset)
		}
	})

	return assets
}

func assetFromButton(button *goquery.Selection) (Asset, bool) {
	filename := strings.TrimSpace(button.AttrOr("download", ""))
	href := strings.TrimSpace(button.AttrOr("href", ""))
	format, err := ParseFormat(strings.TrimPrefix(path.Ext(filename), "."))
	if err != nil || !isAbsoluteURL(href) {
		return Asset{}, false
	}
	return Asset{Format: format, URL: href, Filename: filename}, true
}

// isAbsoluteURL reports whether raw is a complete http(s) address, so a
// relative link never becomes a download target.
func isAbsoluteURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

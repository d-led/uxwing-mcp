// Package uxwing searches uxwing.com for icons, and downloads their SVG and PNG
// files. The icons themselves are free to use under the UXWing licence:
// https://uxwing.com/license/.
package uxwing

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
)

// Format is a file format UXWing serves for an icon. UXWing offers SVG and PNG
// only: probing .jpg, .webp, .ico, .eps and .pdf returns 404.
type Format string

const (
	FormatSVG Format = "svg"
	FormatPNG Format = "png"
)

func (f Format) String() string { return string(f) }

// ParseFormat maps caller input such as "SVG" onto a served format.
func ParseFormat(raw string) (Format, error) {
	format := Format(strings.ToLower(strings.TrimSpace(raw)))
	if format != FormatSVG && format != FormatPNG {
		return "", fmt.Errorf("unsupported icon format %q: UXWing serves svg and png", raw)
	}
	return format, nil
}

// Slug identifies one icon page: https://uxwing.com/<slug>/ such as
// "robot-icon".
type Slug string

func (s Slug) String() string { return string(s) }

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// ParseSlug accepts a bare slug or an uxwing.com page address.
func ParseSlug(raw string) (Slug, error) {
	slug := strings.Trim(strings.TrimSpace(raw), "/")
	if parsed, err := url.Parse(slug); err == nil && parsed.Host != "" {
		if parsed.Host != "uxwing.com" && parsed.Host != "www.uxwing.com" {
			return "", fmt.Errorf("%q is not an uxwing.com address", raw)
		}
		slug = lastPathSegment(parsed.Path)
	}
	if !slugPattern.MatchString(slug) {
		return "", fmt.Errorf("%q is not an icon slug such as robot-icon", raw)
	}
	return Slug(slug), nil
}

// Icon is one icon on UXWing, as listed on a search or icon page.
type Icon struct {
	Slug       Slug   `json:"slug" jsonschema:"icon slug such as robot-icon"`
	Title      string `json:"title" jsonschema:"human readable icon name"`
	PageURL    string `json:"url" jsonschema:"uxwing.com page showing the icon"`
	PreviewURL string `json:"preview_url,omitempty" jsonschema:"512x512 PNG preview of the icon"`
}

// Asset is one downloadable file of an icon.
type Asset struct {
	Format   Format `json:"format"`
	URL      string `json:"url"`
	Filename string `json:"filename"`
}

// asFormat points the same file name at another format.
func (a Asset) asFormat(format Format) Asset {
	name := strings.TrimSuffix(a.Filename, path.Ext(a.Filename)) + "." + format.String()

	parsed, err := url.Parse(a.URL)
	if err != nil {
		return Asset{Format: format, Filename: name}
	}

	parsed.Path = strings.TrimSuffix(parsed.Path, path.Ext(parsed.Path)) + "." + format.String()
	return Asset{Format: format, URL: parsed.String(), Filename: name}
}

// Page is everything the tools need from a single icon page. Related holds the
// variations UXWing offers for the same idea, for example a black or an outline
// drawing of the icon.
type Page struct {
	Slug    Slug    `json:"slug" jsonschema:"icon slug such as robot-icon"`
	Title   string  `json:"title" jsonschema:"human readable icon name"`
	URL     string  `json:"url" jsonschema:"uxwing.com page showing the icon"`
	Assets  []Asset `json:"assets" jsonschema:"files UXWing serves for this icon"`
	Related []Icon  `json:"related" jsonschema:"variations of this icon offered by UXWing"`
}

func lastPathSegment(path string) string {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	return segments[len(segments)-1]
}

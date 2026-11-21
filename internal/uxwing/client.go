package uxwing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const pngSignature = "\x89PNG\r\n\x1a\n"

// Getter fetches one address. It is the client's only network dependency.
type Getter interface {
	Get(ctx context.Context, targetURL, referer string) ([]byte, error)
}

// Client reads uxwing.com pages and files.
type Client struct {
	getter Getter
	base   *url.URL
}

// NewClient takes the site the client reads from, so a test can point it at a
// local server.
func NewClient(getter Getter, baseURL string) (*Client, error) {
	base, err := url.Parse(strings.TrimSuffix(strings.TrimSpace(baseURL), "/"))
	if err != nil {
		return nil, fmt.Errorf("parse UXWing address %q: %w", baseURL, err)
	}
	if base.Scheme == "" || base.Host == "" {
		return nil, fmt.Errorf("UXWing address %q needs a scheme and a host", baseURL)
	}
	return &Client{getter: getter, base: base}, nil
}

// Search returns the icons a UXWing search lists, in page order.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]Icon, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("search query must not be empty")
	}
	if limit < 1 {
		return nil, fmt.Errorf("search limit must be at least 1, got %d", limit)
	}

	doc, err := c.document(ctx, c.base.String()+"/?s="+url.QueryEscape(query))
	if err != nil {
		return nil, fmt.Errorf("search %q: %w", query, err)
	}
	return parseIconCards(doc, limit), nil
}

// Page reads one icon page: its downloads and the variations of the icon.
func (c *Client) Page(ctx context.Context, slug Slug) (Page, error) {
	target := c.pageURL(slug)

	doc, err := c.document(ctx, target)
	if err != nil {
		return Page{}, fmt.Errorf("open icon page %q: %w", slug, err)
	}

	page := parsePage(doc)
	page.Slug = slug
	page.URL = target
	return page, nil
}

// Download fetches one file of an icon.
func (c *Client) Download(ctx context.Context, slug Slug, format Format) ([]byte, Asset, error) {
	asset, err := c.asset(ctx, slug, format)
	if err != nil {
		return nil, Asset{}, err
	}

	data, err := c.fetchImage(ctx, asset.URL, format)
	if err != nil {
		return nil, Asset{}, fmt.Errorf("download %s of %q: %w", format, slug, err)
	}
	return data, asset, nil
}

// Preview fetches an icon image that UXWing serves as PNG.
func (c *Client) Preview(ctx context.Context, previewURL string) ([]byte, error) {
	if !isAbsoluteURL(previewURL) {
		return nil, fmt.Errorf("preview address %q is not a complete URL", previewURL)
	}
	return c.fetchImage(ctx, previewURL, FormatPNG)
}

func (c *Client) asset(ctx context.Context, slug Slug, format Format) (Asset, error) {
	page, err := c.Page(ctx, slug)
	if err != nil {
		return Asset{}, err
	}

	asset, ok := FindAsset(page.Assets, format)
	if !ok {
		return Asset{}, fmt.Errorf("icon %q has no downloads (check %s)", slug, page.URL)
	}
	return asset, nil
}

func (c *Client) fetchImage(ctx context.Context, targetURL string, format Format) ([]byte, error) {
	data, err := c.getter.Get(ctx, targetURL, c.base.String()+"/")
	if err != nil {
		return nil, err
	}
	if err := checkPayload(data, format); err != nil {
		return nil, fmt.Errorf("%w, fetched from %s", err, targetURL)
	}
	return data, nil
}

func (c *Client) document(ctx context.Context, targetURL string) (*goquery.Document, error) {
	raw, err := c.getter.Get(ctx, targetURL, "")
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", targetURL, err)
	}
	return doc, nil
}

func (c *Client) pageURL(slug Slug) string { return c.base.String() + "/" + slug.String() + "/" }

// FindAsset picks the download of the wanted format, if UXWing serves one.
func FindAsset(assets []Asset, format Format) (Asset, bool) {
	for _, asset := range assets {
		if asset.Format == format {
			return asset, true
		}
	}
	return Asset{}, false
}

// FormatNames lists the formats of the given downloads, for messages.
func FormatNames(assets []Asset) []string {
	names := make([]string, len(assets))
	for i, asset := range assets {
		names[i] = asset.Format.String()
	}
	return names
}

// checkPayload makes sure a download really is the requested image and not an
// error page served with status 200.
func checkPayload(data []byte, format Format) error {
	switch format {
	case FormatPNG:
		if !bytes.HasPrefix(data, []byte(pngSignature)) {
			return errors.New("payload is not a PNG image")
		}
		return nil
	case FormatSVG:
		head := bytes.ToLower(data[:min(len(data), 512)])
		if !bytes.Contains(head, []byte("<svg")) {
			return errors.New("payload is not an SVG image")
		}
		return nil
	default:
		return fmt.Errorf("cannot verify format %q", format)
	}
}

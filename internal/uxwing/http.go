package uxwing

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	userAgent       = "Mozilla/5.0 (compatible; uxwing-mcp/1.0)"
	maxPayloadBytes = 8 << 20
)

// HTTPGetter fetches pages and files over HTTP.
type HTTPGetter struct {
	client *http.Client
}

func NewHTTPGetter() *HTTPGetter {
	return &HTTPGetter{client: &http.Client{Timeout: 20 * time.Second}}
}

func (g *HTTPGetter) Get(ctx context.Context, targetURL, referer string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request for %s: %w", targetURL, err)
	}
	request.Header.Set("User-Agent", userAgent)
	if referer != "" {
		request.Header.Set("Referer", referer)
	}

	response, err := g.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", targetURL, err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: HTTP %d", targetURL, response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxPayloadBytes))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", targetURL, err)
	}
	return body, nil
}

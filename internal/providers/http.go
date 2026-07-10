package providers

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// userAgent identifies the app to upstream APIs. Wikimedia in particular rejects
// requests without a descriptive User-Agent, so this is not optional.
const userAgent = "SummitApp/1.0 (https://github.com/kumargaurav/summit-backend; contact@summit.app)"

// getBytes performs a GET and returns the response body on a 2xx. It fails soft:
// on transport error, non-2xx, or read error it logs a warning and returns
// (nil, false) so callers can fall back to hardcoded content.
func getBytes(ctx context.Context, c *http.Client, rawURL string) ([]byte, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		slog.Warn("provider: build request", "url", rawURL, "err", err)
		return nil, false
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		slog.Warn("provider: request failed", "url", rawURL, "err", err)
		return nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.Warn("provider: non-2xx", "url", rawURL, "status", resp.StatusCode)
		return nil, false
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Warn("provider: read body", "url", rawURL, "err", err)
		return nil, false
	}
	return body, true
}

// postBytes performs a form POST (used by Overpass) and returns the body on 2xx,
// failing soft the same way as getBytes.
func postBytes(ctx context.Context, c *http.Client, rawURL, body string) ([]byte, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, strings.NewReader(body))
	if err != nil {
		slog.Warn("provider: build request", "url", rawURL, "err", err)
		return nil, false
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.Do(req)
	if err != nil {
		slog.Warn("provider: request failed", "url", rawURL, "err", err)
		return nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.Warn("provider: non-2xx", "url", rawURL, "status", resp.StatusCode)
		return nil, false
	}
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Warn("provider: read body", "url", rawURL, "err", err)
		return nil, false
	}
	return out, true
}

package providers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// WikiSummary is the trimmed result of a Wikipedia / Wikivoyage page summary.
type WikiSummary struct {
	Extract      string `json:"extract"`
	ThumbnailURL string `json:"thumbnailUrl"`
	SourceURL    string `json:"sourceUrl"`
}

// WikiProvider fetches a place/trek description + thumbnail from the keyless
// Wikimedia REST summary API. It tries the encyclopedia first, then Wikivoyage
// (travel content), and fails soft: any error yields (zero, false).
type WikiProvider interface {
	Summary(ctx context.Context, title string) (WikiSummary, bool)
}

type wikiProvider struct {
	http  *http.Client
	cache *ttlCache
	// hosts is the ordered list of REST bases to try (Wikipedia, then Wikivoyage).
	hosts []string
}

// NewWikiProvider builds a WikiProvider. A nil client gets a 10s-timeout default.
func NewWikiProvider(c *http.Client) WikiProvider {
	if c == nil {
		c = &http.Client{Timeout: 10 * time.Second}
	}
	return &wikiProvider{
		http:  c,
		cache: newTTLCache(24*time.Hour, 4000),
		hosts: []string{
			"https://en.wikipedia.org/api/rest_v1/page/summary/",
			"https://en.wikivoyage.org/api/rest_v1/page/summary/",
		},
	}
}

// rawWikiSummary mirrors the fields we use from the REST summary response.
type rawWikiSummary struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Extract   string `json:"extract"`
	Thumbnail struct {
		Source string `json:"source"`
	} `json:"thumbnail"`
	ContentURLs struct {
		Desktop struct {
			Page string `json:"page"`
		} `json:"desktop"`
	} `json:"content_urls"`
}

// parseWikiSummary decodes a REST summary body into a WikiSummary. It returns
// ok=false for empty extracts or disambiguation/not-found stubs so callers keep
// their hardcoded fallback.
func parseWikiSummary(body []byte) (WikiSummary, bool) {
	var r rawWikiSummary
	if err := json.Unmarshal(body, &r); err != nil {
		return WikiSummary{}, false
	}
	extract := strings.TrimSpace(r.Extract)
	if extract == "" || r.Type == "disambiguation" {
		return WikiSummary{}, false
	}
	return WikiSummary{
		Extract:      extract,
		ThumbnailURL: r.Thumbnail.Source,
		SourceURL:    r.ContentURLs.Desktop.Page,
	}, true
}

func (p *wikiProvider) Summary(ctx context.Context, title string) (WikiSummary, bool) {
	title = strings.TrimSpace(title)
	if title == "" {
		return WikiSummary{}, false
	}
	if v, ok := p.cache.get("w|" + title); ok {
		s := v.(WikiSummary)
		return s, s.Extract != ""
	}
	// The REST path segment uses underscores for spaces and is percent-encoded.
	slug := url.PathEscape(strings.ReplaceAll(title, " ", "_"))
	for _, host := range p.hosts {
		body, ok := getBytes(ctx, p.http, host+slug)
		if !ok {
			continue
		}
		if s, ok := parseWikiSummary(body); ok {
			p.cache.put("w|"+title, s)
			return s, true
		}
	}
	// Cache the negative result too, so repeated misses don't re-hit upstream.
	p.cache.put("w|"+title, WikiSummary{})
	slog.Warn("wiki provider: no summary", "title", title)
	return WikiSummary{}, false
}

package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// CommonsProvider fetches a handful of freely-licensed image URLs for a query
// from the keyless Wikimedia Commons API. Fails soft: any error yields nil.
type CommonsProvider interface {
	Images(ctx context.Context, query string, limit int) []string
}

type commonsProvider struct {
	http  *http.Client
	cache *ttlCache
}

// NewCommonsProvider builds a CommonsProvider. A nil client gets a 10s default.
func NewCommonsProvider(c *http.Client) CommonsProvider {
	if c == nil {
		c = &http.Client{Timeout: 10 * time.Second}
	}
	return &commonsProvider{http: c, cache: newTTLCache(24*time.Hour, 4000)}
}

const commonsAPI = "https://commons.wikimedia.org/w/api.php"

// rawCommons mirrors an action=query&generator=search&prop=imageinfo response.
type rawCommons struct {
	Query struct {
		Pages map[string]struct {
			Title     string `json:"title"`
			ImageInfo []struct {
				URL string `json:"url"`
			} `json:"imageinfo"`
		} `json:"pages"`
	} `json:"query"`
}

// parseCommonsImages extracts image URLs from a Commons query response, capped
// at limit. Order across pages is not guaranteed by the API (map), so callers
// should treat the set as unordered.
func parseCommonsImages(body []byte, limit int) []string {
	var r rawCommons
	if err := json.Unmarshal(body, &r); err != nil {
		return nil
	}
	out := []string{}
	for _, p := range r.Query.Pages {
		for _, ii := range p.ImageInfo {
			if ii.URL != "" {
				out = append(out, ii.URL)
			}
		}
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func (p *commonsProvider) Images(ctx context.Context, query string, limit int) []string {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	if limit <= 0 {
		limit = 3
	}
	key := fmt.Sprintf("c|%d|%s", limit, query)
	if v, ok := p.cache.get(key); ok {
		return v.([]string)
	}
	q := url.Values{}
	q.Set("action", "query")
	q.Set("format", "json")
	q.Set("generator", "search")
	q.Set("gsrsearch", query)
	q.Set("gsrnamespace", "6") // File namespace
	q.Set("gsrlimit", fmt.Sprintf("%d", limit))
	q.Set("prop", "imageinfo")
	q.Set("iiprop", "url")
	q.Set("iiurlwidth", "1024")
	body, ok := getBytes(ctx, p.http, commonsAPI+"?"+q.Encode())
	if !ok {
		return nil
	}
	imgs := parseCommonsImages(body, limit)
	p.cache.put(key, imgs)
	return imgs
}

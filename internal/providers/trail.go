package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Trail is a hiking route or peak surfaced from OpenStreetMap via Overpass.
type Trail struct {
	ID     string  `json:"id"` // OSM element id, e.g. "way/123" or "node/456"
	Name   string  `json:"name"`
	Lat    float64 `json:"lat"`
	Lng    float64 `json:"lng"`
	Kind   string  `json:"kind"`   // TREK (hiking route) | PEAK
	Source string  `json:"source"` // always "openstreetmap"
}

// TrailProvider fetches hiking routes and peaks near a coordinate from the
// keyless Overpass API. Overpass can be slow/rate-limited, so this is defensive:
// a 15s timeout and empty result on any error (caller falls back to treks.go).
type TrailProvider interface {
	Nearby(ctx context.Context, lat, lng float64, radiusMeters int) []Trail
}

type trailProvider struct {
	http  *http.Client
	cache *ttlCache
}

// NewTrailProvider builds a TrailProvider. A nil client gets a 15s-timeout
// default, matching Overpass's tendency to be slow.
func NewTrailProvider(c *http.Client) TrailProvider {
	if c == nil {
		c = &http.Client{Timeout: 15 * time.Second}
	}
	return &trailProvider{http: c, cache: newTTLCache(12*time.Hour, 2000)}
}

const overpassAPI = "https://overpass-api.de/api/interpreter"

// rawOverpass mirrors the Overpass JSON response. Ways queried with `out center`
// carry a synthetic center; nodes carry lat/lon directly.
type rawOverpass struct {
	Elements []struct {
		Type   string  `json:"type"`
		ID     int64   `json:"id"`
		Lat    float64 `json:"lat"`
		Lon    float64 `json:"lon"`
		Center struct {
			Lat float64 `json:"lat"`
			Lon float64 `json:"lon"`
		} `json:"center"`
		Tags map[string]string `json:"tags"`
	} `json:"elements"`
}

// parseTrails maps an Overpass response into named trails/peaks, skipping
// unnamed elements (a nameless route is useless in a list).
func parseTrails(body []byte) []Trail {
	var r rawOverpass
	if err := json.Unmarshal(body, &r); err != nil {
		return nil
	}
	out := []Trail{}
	for _, e := range r.Elements {
		name := strings.TrimSpace(e.Tags["name"])
		if name == "" {
			continue
		}
		lat, lng := e.Lat, e.Lon
		if lat == 0 && lng == 0 {
			lat, lng = e.Center.Lat, e.Center.Lon
		}
		kind := "TREK"
		if e.Tags["natural"] == "peak" {
			kind = "PEAK"
		}
		out = append(out, Trail{
			ID:   e.Type + "/" + strconv.FormatInt(e.ID, 10),
			Name: name, Lat: lat, Lng: lng, Kind: kind, Source: "openstreetmap",
		})
	}
	return out
}

func (p *trailProvider) Nearby(ctx context.Context, lat, lng float64, radiusMeters int) []Trail {
	if radiusMeters <= 0 {
		radiusMeters = 25000
	}
	key := fmt.Sprintf("t|%d|%s", radiusMeters, geoCell(lat, lng))
	if v, ok := p.cache.get(key); ok {
		return v.([]Trail)
	}
	query := fmt.Sprintf(
		`[out:json][timeout:15];(way["route"="hiking"](around:%d,%f,%f);node["natural"="peak"](around:%d,%f,%f););out center 30;`,
		radiusMeters, lat, lng, radiusMeters, lat, lng,
	)
	body, ok := postBytes(ctx, p.http, overpassAPI, "data="+query)
	if !ok {
		return nil // caller falls back to hardcoded treks.go
	}
	trails := parseTrails(body)
	p.cache.put(key, trails)
	return trails
}

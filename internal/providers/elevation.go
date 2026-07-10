package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Point is a lat/lng pair to look up an elevation for.
type Point struct {
	Lat float64
	Lng float64
}

// ElevationProvider batch-resolves elevations (metres) for a list of points via
// the keyless open-elevation API. Used later for trek elevation profiles. Fails
// soft: any error yields nil.
type ElevationProvider interface {
	Lookup(ctx context.Context, points []Point) []float64
}

type elevationProvider struct {
	http  *http.Client
	cache *ttlCache
}

// NewElevationProvider builds an ElevationProvider. A nil client gets a 15s
// default (open-elevation can be slow for large batches).
func NewElevationProvider(c *http.Client) ElevationProvider {
	if c == nil {
		c = &http.Client{Timeout: 15 * time.Second}
	}
	return &elevationProvider{http: c, cache: newTTLCache(24*time.Hour, 4000)}
}

const elevationAPI = "https://api.open-elevation.com/api/v1/lookup"

// rawElevation mirrors the open-elevation lookup response.
type rawElevation struct {
	Results []struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Elevation float64 `json:"elevation"`
	} `json:"results"`
}

// parseElevations extracts the ordered elevation values from a lookup response.
func parseElevations(body []byte) []float64 {
	var r rawElevation
	if err := json.Unmarshal(body, &r); err != nil {
		return nil
	}
	out := make([]float64, 0, len(r.Results))
	for _, res := range r.Results {
		out = append(out, res.Elevation)
	}
	return out
}

// coordKey rounds a point to ~1km so nearby lookups reuse cached elevations.
func coordKey(p Point) string {
	return fmt.Sprintf("%.2f,%.2f", p.Lat, p.Lng)
}

func (p *elevationProvider) Lookup(ctx context.Context, points []Point) []float64 {
	if len(points) == 0 {
		return nil
	}
	// Cache key is the ordered, rounded coordinate set.
	parts := make([]string, len(points))
	for i, pt := range points {
		parts[i] = coordKey(pt)
	}
	key := "e|" + strings.Join(parts, "|")
	if v, ok := p.cache.get(key); ok {
		return v.([]float64)
	}
	locs := make([]string, len(points))
	for i, pt := range points {
		locs[i] = fmt.Sprintf("%f,%f", pt.Lat, pt.Lng)
	}
	url := elevationAPI + "?locations=" + strings.Join(locs, "|")
	body, ok := getBytes(ctx, p.http, url)
	if !ok {
		return nil
	}
	elevs := parseElevations(body)
	if len(elevs) > 0 {
		p.cache.put(key, elevs)
	}
	return elevs
}

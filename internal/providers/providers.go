// Package providers wraps free, keyless external content APIs (Wikipedia,
// Wikimedia Commons, open-elevation, Overpass/OpenStreetMap) behind small
// interfaces, each with its own TTL cache so we never hammer the upstreams.
//
// Every provider fails soft: on any network error, timeout, or non-2xx it logs
// via slog.Warn and returns an empty result, never panicking. The hardcoded
// catalog in internal/explore (treks.go / trek_meta.go / fallback.go) remains
// the seed and fallback whenever a provider returns nothing.
package providers

import (
	"net/http"
	"time"
)

// Providers bundles every external content provider so callers can construct
// and pass them as a unit.
type Providers struct {
	Wiki      WikiProvider
	Commons   CommonsProvider
	Elevation ElevationProvider
	Trail     TrailProvider
}

// New constructs the full provider set with sensible shared HTTP clients. Wiki
// and Commons share a 10s client; elevation and trails get a 15s client because
// open-elevation and Overpass are slower.
func New() *Providers {
	fast := &http.Client{Timeout: 10 * time.Second}
	slow := &http.Client{Timeout: 15 * time.Second}
	return &Providers{
		Wiki:      NewWikiProvider(fast),
		Commons:   NewCommonsProvider(fast),
		Elevation: NewElevationProvider(slow),
		Trail:     NewTrailProvider(slow),
	}
}

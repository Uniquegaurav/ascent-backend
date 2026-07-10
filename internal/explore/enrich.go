package explore

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/kumargaurav/summit-backend/internal/domain"
	"github.com/kumargaurav/summit-backend/internal/providers"
)

// enrich supplements an ExploreDetail with real content from the external
// providers: a Wikipedia/Wikivoyage extract + thumbnail and a few Wikimedia
// Commons images, keyed on the item title. The provider content is preferred
// when present; otherwise the caller's hardcoded blurb/image (trek_meta.go) is
// left untouched. Attribution source URLs are recorded in detail.Sources.
func (h *Handler) enrich(ctx context.Context, detail domain.ExploreDetail) domain.ExploreDetail {
	if h.prov == nil {
		return detail
	}
	title := detail.Item.Title
	if title == "" {
		return detail
	}

	sources := detail.Sources
	photos := detail.Photos

	// Try the exact title first, then cleaned variants — most catalog titles
	// carry a " Trek"/" Trail" suffix that isn't the actual Wikipedia page name
	// ("Triund", not "Triund Trek"). First candidate that resolves wins.
	candidates := titleCandidates(title)

	// Wikipedia / Wikivoyage: prefer its extract over the hardcoded blurb.
	if h.prov.Wiki != nil {
		for _, cand := range candidates {
			s, ok := h.prov.Wiki.Summary(ctx, cand)
			if !ok || s.Extract == "" {
				continue
			}
			detail.Item.Description = s.Extract
			if s.ThumbnailURL != "" {
				// Real photo becomes the hero; keep the hardcoded image behind it.
				photos = dedupePrepend(photos, s.ThumbnailURL)
				detail.Item.ImageURL = s.ThumbnailURL
			}
			if s.SourceURL != "" {
				sources = appendUnique(sources, s.SourceURL)
			}
			break
		}
	}

	// Wikimedia Commons: a few extra freely-licensed images.
	if h.prov.Commons != nil {
		if imgs := h.prov.Commons.Images(ctx, candidates[len(candidates)-1], 3); len(imgs) > 0 {
			for _, u := range imgs {
				photos = appendUnique(photos, u)
			}
			sources = appendUnique(sources, "https://commons.wikimedia.org/")
		}
	}

	detail.Photos = photos
	detail.Sources = sources
	return detail
}

// titleCandidates returns the title plus cleaned variants (suffixes like
// "Trek"/"Trail"/"Trekking" stripped), most-specific first, de-duplicated.
func titleCandidates(title string) []string {
	out := []string{title}
	trimmed := title
	for _, suffix := range []string{" Trek", " Trekking", " Trail", " Hike", " National Park Trek"} {
		if strings.HasSuffix(trimmed, suffix) {
			trimmed = strings.TrimSpace(strings.TrimSuffix(trimmed, suffix))
		}
	}
	if trimmed != "" && trimmed != title {
		out = append(out, trimmed)
	}
	return out
}

func appendUnique(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}

// dedupePrepend puts v at the front of list (as the hero) without duplicating it.
func dedupePrepend(list []string, v string) []string {
	out := []string{v}
	for _, x := range list {
		if x != v {
			out = append(out, x)
		}
	}
	return out
}

func take2(trails []providers.Trail, n int) []providers.Trail {
	if len(trails) > n {
		return trails[:n]
	}
	return trails
}

// trailToItem converts an OSM trail/peak into an ExploreItem card. Both hiking
// routes and peaks render as TREK cards in the treks rails.
func trailToItem(t providers.Trail) domain.ExploreItem {
	subtitle := "Nearby route"
	if t.Kind == "PEAK" {
		subtitle = "Nearby peak"
	}
	return domain.ExploreItem{
		ID: "osm_" + t.ID, Title: t.Name, Subtitle: subtitle,
		Category: "trekking", Kind: "TREK", LocationName: "Near you", Theme: "ALPINE",
		ImageURL: img(trekImgs[0]),
	}.WithSummitCategory()
}

// trailItemByID resolves an "osm_<type>/<id>" card back to an ExploreItem from
// the trails catalog, so a tapped nearby-trail card can open a detail page.
func (h *Handler) trailItemByID(ctx context.Context, id string) (domain.ExploreItem, bool) {
	if h.pool == nil {
		return domain.ExploreItem{}, false
	}
	osmID := strings.TrimPrefix(id, "osm_")
	var name, source string
	var lat, lng float64
	err := h.pool.QueryRow(ctx,
		`SELECT name, lat, lng, source FROM trails WHERE id = $1`, osmID).
		Scan(&name, &lat, &lng, &source)
	if err != nil || name == "" {
		return domain.ExploreItem{}, false
	}
	return trailToItem(providers.Trail{ID: osmID, Name: name, Lat: lat, Lng: lng, Source: source}), true
}

// persistTrails writes fetched OSM trails into the trails catalog
// (fire-and-forget), mirroring persistPlaces so the catalog becomes a
// server-owned asset instead of a per-request API cost.
func (h *Handler) persistTrails(trails []providers.Trail) {
	if h.pool == nil || len(trails) == 0 {
		return
	}
	go func(trails []providers.Trail) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, t := range trails {
			_, err := h.pool.Exec(ctx, `
				INSERT INTO trails (id, name, lat, lng, source)
				VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (id) DO UPDATE SET
					name = EXCLUDED.name, lat = EXCLUDED.lat, lng = EXCLUDED.lng,
					source = EXCLUDED.source, last_seen = now()`,
				t.ID, t.Name, t.Lat, t.Lng, t.Source)
			if err != nil {
				slog.Warn("trails upsert failed", "err", err)
				return
			}
		}
	}(trails)
}

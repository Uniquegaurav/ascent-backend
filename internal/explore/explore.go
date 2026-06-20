package explore

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/kumargaurav/summit-backend/internal/domain"
	"github.com/kumargaurav/summit-backend/internal/httpx"
	"github.com/kumargaurav/summit-backend/internal/places"
)

type Handler struct{ pc *places.Client }

func NewHandler(pc *places.Client) *Handler { return &Handler{pc: pc} }

// Relative photo-proxy URL; the app prefixes the API base. Falls back to a stock
// image when a place has no photo.
func photoURL(ref string) string {
	if ref == "" {
		return "https://images.unsplash.com/photo-1500530855697-b586d89ba3ee?auto=format&fit=crop&w=1200&q=80"
	}
	return "/place-photo?w=800&ref=" + url.QueryEscape(ref)
}

func themeForTypes(types []string) string {
	has := func(s string) bool {
		for _, t := range types {
			if strings.Contains(t, s) {
				return true
			}
		}
		return false
	}
	switch {
	case has("natural_feature"), has("park"), has("campground"), has("hiking"):
		return "ALPINE"
	case has("gym"), has("stadium"), has("sports"):
		return "FOREST"
	case has("book"), has("library"), has("school"), has("university"):
		return "COSMIC"
	case has("bar"), has("night_club"), has("movie"), has("art_gallery"):
		return "EMBER"
	case has("beach"), has("aquarium"), has("lodging"):
		return "OCEAN"
	default:
		return "DESERT"
	}
}

func placeToItem(p places.Place, category, kind string) domain.ExploreItem {
	loc := p.Address
	if i := strings.Index(loc, ","); i > 0 {
		loc = strings.TrimSpace(loc[:i])
	}
	return domain.ExploreItem{
		ID: p.PlaceID, PlaceID: p.PlaceID, Title: p.Name, Subtitle: p.Address,
		Category: category, Kind: kind, LocationName: loc, ImageURL: photoURL(p.PhotoRef),
		Theme: themeForTypes(p.Types), Rating: p.Rating, RatingsTotal: p.RatingsTotal,
	}
}

func take(items []domain.ExploreItem, n int) []domain.ExploreItem {
	if len(items) > n {
		return items[:n]
	}
	return items
}

func (h *Handler) searchItems(ctx context.Context, query, category, kind string, lat, lng float64) []domain.ExploreItem {
	res, err := h.pc.TextSearch(ctx, query, lat, lng)
	if err != nil {
		return nil
	}
	out := make([]domain.ExploreItem, 0, len(res))
	for _, p := range res {
		out = append(out, placeToItem(p, category, kind))
	}
	return out
}

// Hobby launchers are curated; tapping one runs a Places search for nearby teachers.
var hobbyLaunchers = []domain.ExploreItem{
	{ID: "hob_guitar", Title: "Learn Guitar", Subtitle: "Find classes near you", Category: "music", Kind: "HOBBY", Theme: "EMBER", SearchQuery: "guitar classes", ImageURL: img("1510915361894-db8b60106cb1")},
	{ID: "hob_pottery", Title: "Pottery", Subtitle: "Workshops near you", Category: "art", Kind: "HOBBY", Theme: "AURORA", SearchQuery: "pottery workshop", ImageURL: img("1513364776144-60967b0f800f")},
	{ID: "hob_dance", Title: "Dance", Subtitle: "Studios near you", Category: "dance", Kind: "HOBBY", Theme: "EMBER", SearchQuery: "dance classes", ImageURL: img("1504609773096-104ff2c73ba4")},
	{ID: "hob_climb", Title: "Climbing", Subtitle: "Gyms near you", Category: "fitness", Kind: "HOBBY", Theme: "ALPINE", SearchQuery: "climbing gym", ImageURL: img("1522163182402-834f871fd851")},
	{ID: "hob_yoga", Title: "Yoga", Subtitle: "Studios near you", Category: "fitness", Kind: "HOBBY", Theme: "FOREST", SearchQuery: "yoga studio", ImageURL: img("1545205597-3d9d02c29597")},
	{ID: "hob_swim", Title: "Swimming", Subtitle: "Pools near you", Category: "fitness", Kind: "HOBBY", Theme: "OCEAN", SearchQuery: "swimming pool", ImageURL: img("1530549387789-4c1017266635")},
}

func img(id string) string {
	return "https://images.unsplash.com/photo-" + id + "?auto=format&fit=crop&w=1200&q=80"
}

func (h *Handler) Feed(w http.ResponseWriter, r *http.Request) {
	lat, lng := parseLatLng(r)
	if !h.pc.Enabled() || (lat == 0 && lng == 0) {
		httpx.JSON(w, http.StatusOK, fallbackFeed())
		return
	}
	ctx := r.Context()
	sections := []domain.ExploreSection{
		{ID: "popular", Title: "Popular near you", Layout: "CARDS", Items: take(h.searchItems(ctx, "top tourist attractions", "explore", "PLACE", lat, lng), 6)},
		{ID: "treks", Title: "Trending treks", Layout: "CAROUSEL", Items: take(h.searchItems(ctx, "trekking trail hiking", "trekking", "TREK", lat, lng), 8)},
		{ID: "hobbies", Title: "Start a new hobby", Layout: "GRID", Items: hobbyLaunchers},
		{ID: "workshops", Title: "Workshops near you", Layout: "CARDS", Items: take(h.searchItems(ctx, "workshop class", "learning", "EVENT", lat, lng), 6)},
	}
	httpx.JSON(w, http.StatusOK, domain.ExploreFeed{Sections: sections})
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		httpx.JSON(w, http.StatusOK, map[string]any{"items": []domain.ExploreItem{}})
		return
	}
	lat, lng := parseLatLng(r)
	if !h.pc.Enabled() {
		httpx.JSON(w, http.StatusOK, map[string]any{"items": fallbackSearch(q)})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": h.searchItems(r.Context(), q, "explore", "PLACE", lat, lng)})
}

func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// Static hobby launcher → return as-is (no reviews).
	for _, hob := range hobbyLaunchers {
		if hob.ID == id {
			httpx.JSON(w, http.StatusOK, domain.ExploreDetail{Item: hob})
			return
		}
	}
	if !h.pc.Enabled() {
		if it, ok := fallbackItem(id); ok {
			httpx.JSON(w, http.StatusOK, domain.ExploreDetail{Item: it})
			return
		}
		httpx.Error(w, http.StatusNotFound, "not found")
		return
	}
	d, err := h.pc.Details(r.Context(), id)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "details lookup failed")
		return
	}
	photos := make([]string, 0, len(d.PhotoRefs))
	for _, ref := range d.PhotoRefs {
		photos = append(photos, photoURL(ref))
	}
	reviews := make([]domain.ExploreReview, 0, len(d.Reviews))
	for _, rv := range d.Reviews {
		reviews = append(reviews, domain.ExploreReview{Author: rv.Author, Rating: rv.Rating, Text: rv.Text, When: rv.When})
	}
	hero := ""
	if len(photos) > 0 {
		hero = photos[0]
	} else {
		hero = photoURL("")
	}
	loc := d.Address
	if i := strings.Index(loc, ","); i > 0 {
		loc = strings.TrimSpace(loc[:i])
	}
	item := domain.ExploreItem{
		ID: d.PlaceID, PlaceID: d.PlaceID, Title: d.Name, Subtitle: d.Address,
		Category: "explore", Kind: "PLACE", LocationName: loc, ImageURL: hero,
		Theme: themeForTypes(d.Types), Rating: d.Rating, RatingsTotal: d.RatingsTotal,
	}
	httpx.JSON(w, http.StatusOK, domain.ExploreDetail{
		Item: item, Photos: photos, Reviews: reviews,
		Address: d.Address, Phone: d.Phone, Website: d.Website, Hours: d.Hours,
	})
}

// Photo proxies a Place photo (public — Coil can't send the JWT).
func (h *Handler) Photo(w http.ResponseWriter, r *http.Request) {
	ref := r.URL.Query().Get("ref")
	width := 800
	if v, err := strconv.Atoi(r.URL.Query().Get("w")); err == nil && v > 0 {
		width = v
	}
	if ref == "" || !h.pc.Enabled() {
		http.Redirect(w, r, photoURL(""), http.StatusFound)
		return
	}
	ct, err := h.pc.Photo(r.Context(), ref, width, w)
	if err != nil {
		return
	}
	if ct != "" {
		w.Header().Set("Content-Type", ct)
	}
}

// Debug surfaces Google's raw status/error for the configured key (temporary).
func (h *Handler) Debug(w http.ResponseWriter, r *http.Request) {
	lat, lng := parseLatLng(r)
	if lat == 0 && lng == 0 {
		lat, lng = 12.9716, 77.5946
	}
	httpx.JSON(w, http.StatusOK, h.pc.Diagnose(r.Context(), lat, lng))
}

func (h *Handler) ReverseGeocode(w http.ResponseWriter, r *http.Request) {
	lat, lng := parseLatLng(r)
	if !h.pc.Enabled() || (lat == 0 && lng == 0) {
		httpx.JSON(w, http.StatusOK, map[string]string{"label": ""})
		return
	}
	label, err := h.pc.ReverseGeocode(r.Context(), lat, lng)
	if err != nil {
		httpx.JSON(w, http.StatusOK, map[string]string{"label": ""})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"label": label})
}

func parseLatLng(r *http.Request) (float64, float64) {
	lat, _ := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lng, _ := strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
	return lat, lng
}

// Lookup lets the ascent package build an ascent from an explore item (static or place).
func Lookup(id string) (domain.ExploreItem, bool) {
	for _, hob := range hobbyLaunchers {
		if hob.ID == id {
			return hob, true
		}
	}
	return fallbackItem(id)
}

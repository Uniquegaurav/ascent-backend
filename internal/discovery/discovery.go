package discovery

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/kumargaurav/summit-backend/internal/httpx"
	"github.com/kumargaurav/summit-backend/internal/places"
)

// Response shapes mirror the app's PlacesDto.
type PlacesResponse struct {
	Results []PlaceResult `json:"results"`
}

type PlaceResult struct {
	PlaceID          string   `json:"place_id"`
	Name             string   `json:"name"`
	Vicinity         string   `json:"vicinity,omitempty"`
	Rating           *float64 `json:"rating,omitempty"`
	UserRatingsTotal *int     `json:"user_ratings_total,omitempty"`
	Types            []string `json:"types,omitempty"`
	DistanceKm       *float64 `json:"distance_km,omitempty"`
}

type EventsResponse struct {
	Events []Event `json:"events"`
}

type Event struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Category   string   `json:"category"`
	WhenLabel  string   `json:"whenLabel"`
	Venue      string   `json:"venue,omitempty"`
	DistanceKm *float64 `json:"distance_km,omitempty"`
}

type Service struct{ pc *places.Client }

func NewService(pc *places.Client) *Service { return &Service{pc: pc} }

func keywordFor(category string) string {
	switch category {
	case "trekking":
		return "hiking trail"
	case "running":
		return "running track park"
	case "music":
		return "live music venue"
	case "gaming":
		return "gaming cafe"
	case "reading":
		return "bookstore"
	case "fitness":
		return "gym"
	case "dance":
		return "dance studio"
	case "art":
		return "art studio"
	default:
		return category
	}
}

func (s *Service) places(ctx context.Context, category string, lat, lng float64) []PlaceResult {
	if !s.pc.Enabled() || (lat == 0 && lng == 0) {
		return samplePlaces(category)
	}
	res, err := s.pc.TextSearch(ctx, keywordFor(category), lat, lng)
	if err != nil {
		return samplePlaces(category)
	}
	out := make([]PlaceResult, 0, len(res))
	for _, p := range res {
		var rating *float64
		if p.Rating > 0 {
			r := p.Rating
			rating = &r
		}
		var total *int
		if p.RatingsTotal > 0 {
			t := p.RatingsTotal
			total = &t
		}
		out = append(out, PlaceResult{PlaceID: p.PlaceID, Name: p.Name, Vicinity: p.Address, Rating: rating, UserRatingsTotal: total, Types: p.Types})
		if len(out) >= 8 {
			break
		}
	}
	return out
}

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Places(w http.ResponseWriter, r *http.Request) {
	cat := r.URL.Query().Get("type")
	lat, lng := parseLocation(r.URL.Query().Get("location"))
	httpx.JSON(w, http.StatusOK, PlacesResponse{Results: h.svc.places(r.Context(), cat, lat, lng)})
}

func (h *Handler) Events(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, EventsResponse{Events: sampleEvents(r.URL.Query().Get("type"))})
}

func parseLocation(s string) (float64, float64) {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return 0, 0
	}
	lat, _ := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	lng, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	return lat, lng
}

// ---- sample fallback ------------------------------------------------------

func f(v float64) *float64 { return &v }

// Curated per-category sample places used when Google Places is disabled, so every
// hobby/category screen shows distinct, believable content.
var samplePlacesByCategory = map[string][]PlaceResult{
	"trekking": {
		{PlaceID: "p_trek1", Name: "Skandagiri Trail Head", Vicinity: "Nandi Hills Rd", Rating: f(4.7), UserRatingsTotal: i(2100), Types: []string{"trail"}, DistanceKm: f(38)},
		{PlaceID: "p_trek2", Name: "Savandurga Base Camp", Vicinity: "Magadi", Rating: f(4.5), UserRatingsTotal: i(860), Types: []string{"campground"}, DistanceKm: f(52)},
		{PlaceID: "p_trek3", Name: "Anthargange Caves Trek", Vicinity: "Kolar", Rating: f(4.4), UserRatingsTotal: i(540), Types: []string{"trail"}, DistanceKm: f(70)},
	},
	"running": {
		{PlaceID: "p_run1", Name: "Cubbon Park Loop", Vicinity: "Central", Rating: f(4.8), UserRatingsTotal: i(9200), Types: []string{"park"}, DistanceKm: f(3.1)},
		{PlaceID: "p_run2", Name: "Kanteerava Stadium Track", Vicinity: "Sampangi Rama Nagar", Rating: f(4.6), UserRatingsTotal: i(3100), Types: []string{"stadium"}, DistanceKm: f(4.2)},
		{PlaceID: "p_run3", Name: "Agara Lake Trail", Vicinity: "HSR Layout", Rating: f(4.5), UserRatingsTotal: i(2400), Types: []string{"trail"}, DistanceKm: f(6.0)},
	},
	"fitness": {
		{PlaceID: "p_fit1", Name: "Cult.fit HSR", Vicinity: "HSR Layout", Rating: f(4.6), UserRatingsTotal: i(1800), Types: []string{"gym"}, DistanceKm: f(2.5)},
		{PlaceID: "p_fit2", Name: "Gold's Gym Indiranagar", Vicinity: "Indiranagar", Rating: f(4.4), UserRatingsTotal: i(2600), Types: []string{"gym"}, DistanceKm: f(5.1)},
		{PlaceID: "p_fit3", Name: "The Yard CrossFit", Vicinity: "Koramangala", Rating: f(4.8), UserRatingsTotal: i(720), Types: []string{"gym"}, DistanceKm: f(4.0)},
	},
	"music": {
		{PlaceID: "p_mus1", Name: "The Humming Tree", Vicinity: "Indiranagar", Rating: f(4.6), UserRatingsTotal: i(4100), Types: []string{"live_music_venue"}, DistanceKm: f(6.8)},
		{PlaceID: "p_mus2", Name: "Bangalore School of Music", Vicinity: "RT Nagar", Rating: f(4.7), UserRatingsTotal: i(900), Types: []string{"school"}, DistanceKm: f(9.2)},
		{PlaceID: "p_mus3", Name: "Furtados Music", Vicinity: "Brigade Rd", Rating: f(4.5), UserRatingsTotal: i(1300), Types: []string{"store"}, DistanceKm: f(7.0)},
	},
	"dance": {
		{PlaceID: "p_dan1", Name: "Lourd Vijay's Dance Studio", Vicinity: "Residency Rd", Rating: f(4.7), UserRatingsTotal: i(1600), Types: []string{"dance_studio"}, DistanceKm: f(6.2)},
		{PlaceID: "p_dan2", Name: "The Dance Centre", Vicinity: "Koramangala", Rating: f(4.6), UserRatingsTotal: i(840), Types: []string{"dance_studio"}, DistanceKm: f(4.3)},
		{PlaceID: "p_dan3", Name: "Shiamak Davar Studio", Vicinity: "Whitefield", Rating: f(4.5), UserRatingsTotal: i(620), Types: []string{"dance_studio"}, DistanceKm: f(14)},
	},
	"art": {
		{PlaceID: "p_art1", Name: "The Pottery Studio", Vicinity: "Jayanagar", Rating: f(4.9), UserRatingsTotal: i(430), Types: []string{"art_studio"}, DistanceKm: f(8.1)},
		{PlaceID: "p_art2", Name: "Chitrakala Parishath", Vicinity: "Kumara Krupa", Rating: f(4.7), UserRatingsTotal: i(2200), Types: []string{"art_gallery"}, DistanceKm: f(7.5)},
		{PlaceID: "p_art3", Name: "Ochre Art Studio", Vicinity: "Indiranagar", Rating: f(4.6), UserRatingsTotal: i(310), Types: []string{"art_studio"}, DistanceKm: f(6.0)},
	},
	"photography": {
		{PlaceID: "p_pho1", Name: "Lalbagh Botanical Garden", Vicinity: "Lalbagh", Rating: f(4.6), UserRatingsTotal: i(28000), Types: []string{"park"}, DistanceKm: f(5.4)},
		{PlaceID: "p_pho2", Name: "Bangalore Palace", Vicinity: "Vasanth Nagar", Rating: f(4.4), UserRatingsTotal: i(15000), Types: []string{"landmark"}, DistanceKm: f(6.7)},
		{PlaceID: "p_pho3", Name: "Nandi Hills Viewpoint", Vicinity: "Chikkaballapur", Rating: f(4.5), UserRatingsTotal: i(9000), Types: []string{"viewpoint"}, DistanceKm: f(60)},
	},
	"reading": {
		{PlaceID: "p_rea1", Name: "Blossom Book House", Vicinity: "Church St", Rating: f(4.8), UserRatingsTotal: i(7400), Types: []string{"bookstore"}, DistanceKm: f(7.1)},
		{PlaceID: "p_rea2", Name: "Atta Galatta", Vicinity: "Koramangala", Rating: f(4.6), UserRatingsTotal: i(2100), Types: []string{"bookstore"}, DistanceKm: f(4.5)},
		{PlaceID: "p_rea3", Name: "Champaca Bookstore", Vicinity: "Vasanth Nagar", Rating: f(4.7), UserRatingsTotal: i(1200), Types: []string{"bookstore"}, DistanceKm: f(6.9)},
	},
	"gaming": {
		{PlaceID: "p_gam1", Name: "Smaaash", Vicinity: "Mantri Mall", Rating: f(4.3), UserRatingsTotal: i(5600), Types: []string{"arcade"}, DistanceKm: f(8.0)},
		{PlaceID: "p_gam2", Name: "Player's Lounge eSports", Vicinity: "HSR Layout", Rating: f(4.5), UserRatingsTotal: i(420), Types: []string{"gaming_cafe"}, DistanceKm: f(3.0)},
	},
	"football": {
		{PlaceID: "p_fb1", Name: "PlayArena Turf", Vicinity: "Sarjapur Rd", Rating: f(4.6), UserRatingsTotal: i(3400), Types: []string{"sports"}, DistanceKm: f(11)},
		{PlaceID: "p_fb2", Name: "Just Play Turf", Vicinity: "HSR Layout", Rating: f(4.5), UserRatingsTotal: i(900), Types: []string{"sports"}, DistanceKm: f(3.4)},
	},
	"travel": {
		{PlaceID: "p_trv1", Name: "Coorg Hill Retreat", Vicinity: "Madikeri", Rating: f(4.7), UserRatingsTotal: i(5200), Types: []string{"destination"}, DistanceKm: f(250)},
		{PlaceID: "p_trv2", Name: "Hampi Heritage Walk", Vicinity: "Hampi", Rating: f(4.8), UserRatingsTotal: i(8100), Types: []string{"destination"}, DistanceKm: f(340)},
	},
	"learning": {
		{PlaceID: "p_lrn1", Name: "Cooking Masterclass Studio", Vicinity: "Koramangala", Rating: f(4.7), UserRatingsTotal: i(310), Types: []string{"class"}, DistanceKm: f(4.4)},
		{PlaceID: "p_lrn2", Name: "Maker's Asylum Workshop", Vicinity: "HSR Layout", Rating: f(4.6), UserRatingsTotal: i(180), Types: []string{"workshop"}, DistanceKm: f(3.2)},
	},
	"cafe": {
		{PlaceID: "p_caf1", Name: "Third Wave Coffee", Vicinity: "HSR Layout", Rating: f(4.4), UserRatingsTotal: i(4200), Types: []string{"cafe"}, DistanceKm: f(2.1)},
		{PlaceID: "p_caf2", Name: "Dyu Art Cafe", Vicinity: "Koramangala", Rating: f(4.6), UserRatingsTotal: i(3100), Types: []string{"cafe"}, DistanceKm: f(4.0)},
		{PlaceID: "p_caf3", Name: "Matteo Coffea", Vicinity: "Church St", Rating: f(4.5), UserRatingsTotal: i(2600), Types: []string{"cafe"}, DistanceKm: f(7.0)},
	},
	"park": {
		{PlaceID: "p_prk1", Name: "Cubbon Park", Vicinity: "Central", Rating: f(4.6), UserRatingsTotal: i(31000), Types: []string{"park"}, DistanceKm: f(3.1)},
		{PlaceID: "p_prk2", Name: "Lalbagh Garden", Vicinity: "Lalbagh", Rating: f(4.6), UserRatingsTotal: i(28000), Types: []string{"park"}, DistanceKm: f(5.4)},
	},
}

func samplePlaces(category string) []PlaceResult {
	if list, ok := samplePlacesByCategory[strings.ToLower(category)]; ok {
		return list
	}
	// Unknown category → distinct, on-topic-looking entries so no two categories
	// ever show identical content.
	label := titleCase(category)
	return []PlaceResult{
		{PlaceID: "p_" + category + "_1", Name: label + " Studio", Vicinity: "Indiranagar", Rating: f(4.6), UserRatingsTotal: i(420), Types: []string{"point_of_interest"}, DistanceKm: f(4.2)},
		{PlaceID: "p_" + category + "_2", Name: label + " Collective", Vicinity: "Koramangala", Rating: f(4.5), UserRatingsTotal: i(260), Types: []string{"point_of_interest"}, DistanceKm: f(5.6)},
		{PlaceID: "p_" + category + "_3", Name: label + " Hub", Vicinity: "HSR Layout", Rating: f(4.4), UserRatingsTotal: i(180), Types: []string{"point_of_interest"}, DistanceKm: f(3.0)},
	}
}

func i(v int) *int { return &v }

func titleCase(s string) string {
	if s == "" {
		return "Nearby"
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func sampleEvents(category string) []Event {
	switch category {
	case "trekking":
		return []Event{{ID: "e_t1", Name: "Sunrise Hike: Skandagiri", Category: "Trek", WhenLabel: "Sat · 4:30 AM", Venue: "Nandi Hills", DistanceKm: f(38)}}
	case "music":
		return []Event{{ID: "e_m1", Name: "Open Mic Night", Category: "Live", WhenLabel: "Fri · 8:00 PM", Venue: "The Humming Tree", DistanceKm: f(6.8)}}
	default:
		return []Event{}
	}
}

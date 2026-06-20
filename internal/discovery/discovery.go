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

func samplePlaces(category string) []PlaceResult {
	switch category {
	case "trekking":
		return []PlaceResult{
			{PlaceID: "p1", Name: "Skandagiri Trail Head", Vicinity: "Nandi Hills Rd", Rating: f(4.7), Types: []string{"trail"}, DistanceKm: f(38)},
			{PlaceID: "p2", Name: "Savandurga Base Camp", Vicinity: "Magadi", Rating: f(4.5), Types: []string{"campground"}, DistanceKm: f(52)},
		}
	case "running":
		return []PlaceResult{{PlaceID: "p4", Name: "Cubbon Park Loop", Vicinity: "Central", Rating: f(4.8), Types: []string{"park"}, DistanceKm: f(3.1)}}
	case "music":
		return []PlaceResult{{PlaceID: "p7", Name: "The Humming Tree", Vicinity: "Indiranagar", Rating: f(4.6), Types: []string{"live_music_venue"}, DistanceKm: f(6.8)}}
	default:
		return []PlaceResult{{PlaceID: "p_def", Name: "Explorer's Club", Vicinity: "City Centre", Rating: f(4.5), Types: []string{"point_of_interest"}, DistanceKm: f(4)}}
	}
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

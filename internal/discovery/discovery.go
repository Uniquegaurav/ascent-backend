package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kumargaurav/summit-backend/internal/httpx"
)

// Response shapes mirror the app's PlacesDto so the client parses them as-is.

type PlacesResponse struct {
	Results []PlaceResult `json:"results"`
}

type PlaceResult struct {
	PlaceID         string   `json:"place_id"`
	Name            string   `json:"name"`
	Vicinity        string   `json:"vicinity,omitempty"`
	Rating          *float64 `json:"rating,omitempty"`
	UserRatingsTotal *int    `json:"user_ratings_total,omitempty"`
	Types           []string `json:"types,omitempty"`
	DistanceKm      *float64 `json:"distance_km,omitempty"`
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

type Service struct {
	client    *http.Client
	googleKey string
}

func NewService(googleKey string) *Service {
	return &Service{client: &http.Client{Timeout: 8 * time.Second}, googleKey: googleKey}
}

func (s *Service) Places(ctx context.Context, interestID string, lat, lng float64) ([]PlaceResult, error) {
	if s.googleKey == "" {
		return samplePlaces(interestID), nil
	}
	return s.googlePlaces(ctx, interestID, lat, lng)
}

func (s *Service) Events(_ context.Context, interestID string) []Event {
	// No public Google events API; curated for now (real provider plugs in here).
	return sampleEvents(interestID)
}

// ---- Google Places Nearby Search ------------------------------------------

func (s *Service) googlePlaces(ctx context.Context, interestID string, lat, lng float64) ([]PlaceResult, error) {
	q := url.Values{}
	q.Set("location", fmt.Sprintf("%f,%f", lat, lng))
	q.Set("radius", "8000")
	q.Set("keyword", keywordFor(interestID))
	q.Set("key", s.googleKey)
	endpoint := "https://maps.googleapis.com/maps/api/place/nearbysearch/json?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw struct {
		Results []struct {
			PlaceID          string   `json:"place_id"`
			Name             string   `json:"name"`
			Vicinity         string   `json:"vicinity"`
			Rating           *float64 `json:"rating"`
			UserRatingsTotal *int     `json:"user_ratings_total"`
			Types            []string `json:"types"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	out := make([]PlaceResult, 0, len(raw.Results))
	for _, r := range raw.Results {
		out = append(out, PlaceResult{
			PlaceID: r.PlaceID, Name: r.Name, Vicinity: r.Vicinity,
			Rating: r.Rating, UserRatingsTotal: r.UserRatingsTotal, Types: r.Types,
		})
	}
	return out, nil
}

func keywordFor(interestID string) string {
	switch interestID {
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
	default:
		return interestID
	}
}

// ---- HTTP -----------------------------------------------------------------

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Places(w http.ResponseWriter, r *http.Request) {
	interest := r.URL.Query().Get("type")
	lat, lng := parseLocation(r.URL.Query().Get("location"))
	results, err := h.svc.Places(r.Context(), interest, lat, lng)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "places lookup failed")
		return
	}
	httpx.JSON(w, http.StatusOK, PlacesResponse{Results: results})
}

func (h *Handler) Events(w http.ResponseWriter, r *http.Request) {
	interest := r.URL.Query().Get("type")
	httpx.JSON(w, http.StatusOK, EventsResponse{Events: h.svc.Events(r.Context(), interest)})
}

func parseLocation(s string) (float64, float64) {
	// default: Bangalore
	lat, lng := 12.9716, 77.5946
	parts := strings.Split(s, ",")
	if len(parts) == 2 {
		if v, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64); err == nil {
			lat = v
		}
		if v, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
			lng = v
		}
	}
	return lat, lng
}

// ---- Curated sample data --------------------------------------------------

func f(v float64) *float64 { return &v }

func samplePlaces(interestID string) []PlaceResult {
	switch interestID {
	case "trekking":
		return []PlaceResult{
			{PlaceID: "p1", Name: "Skandagiri Trail Head", Vicinity: "Nandi Hills Rd", Rating: f(4.7), Types: []string{"trail", "park"}, DistanceKm: f(38)},
			{PlaceID: "p2", Name: "Savandurga Base Camp", Vicinity: "Magadi", Rating: f(4.5), Types: []string{"campground"}, DistanceKm: f(52)},
			{PlaceID: "p3", Name: "Decathlon Trek Gear", Vicinity: "Whitefield", Rating: f(4.4), Types: []string{"outdoor_store"}, DistanceKm: f(9.2)},
		}
	case "running":
		return []PlaceResult{
			{PlaceID: "p4", Name: "Cubbon Park Loop", Vicinity: "Central", Rating: f(4.8), Types: []string{"park"}, DistanceKm: f(3.1)},
			{PlaceID: "p5", Name: "Kanteerava Stadium Track", Vicinity: "Sampangi", Rating: f(4.6), Types: []string{"stadium"}, DistanceKm: f(4)},
		}
	case "music":
		return []PlaceResult{
			{PlaceID: "p6", Name: "Furtados Music Store", Vicinity: "Brigade Rd", Rating: f(4.5), Types: []string{"music_store"}, DistanceKm: f(5.5)},
			{PlaceID: "p7", Name: "The Humming Tree", Vicinity: "Indiranagar", Rating: f(4.6), Types: []string{"live_music_venue"}, DistanceKm: f(6.8)},
		}
	case "gaming":
		return []PlaceResult{
			{PlaceID: "p8", Name: "Pixel Arena LAN Cafe", Vicinity: "Koramangala", Rating: f(4.4), Types: []string{"gaming_cafe"}, DistanceKm: f(7.1)},
		}
	case "reading":
		return []PlaceResult{
			{PlaceID: "p11", Name: "Blossom Book House", Vicinity: "Church St", Rating: f(4.8), Types: []string{"book_store"}, DistanceKm: f(5.2)},
			{PlaceID: "p12", Name: "Atta Galatta", Vicinity: "Koramangala", Rating: f(4.6), Types: []string{"book_store", "cafe"}, DistanceKm: f(7)},
		}
	default:
		return []PlaceResult{
			{PlaceID: "p_def", Name: "Explorer's Club", Vicinity: "City Centre", Rating: f(4.5), Types: []string{"point_of_interest"}, DistanceKm: f(4)},
		}
	}
}

func sampleEvents(interestID string) []Event {
	switch interestID {
	case "trekking":
		return []Event{
			{ID: "e_t1", Name: "Sunrise Hike: Skandagiri", Category: "Trek", WhenLabel: "Sat · 4:30 AM", Venue: "Nandi Hills", DistanceKm: f(38)},
			{ID: "e_t2", Name: "Western Ghats Weekend", Category: "Trek", WhenLabel: "Next Sat", Venue: "Kudremukh", DistanceKm: f(320)},
		}
	case "running":
		return []Event{{ID: "e_r1", Name: "Sunday Park Run 10K", Category: "Run", WhenLabel: "Sun · 6:00 AM", Venue: "Cubbon Park", DistanceKm: f(3.1)}}
	case "music":
		return []Event{
			{ID: "e_m1", Name: "Open Mic Night", Category: "Live", WhenLabel: "Fri · 8:00 PM", Venue: "The Humming Tree", DistanceKm: f(6.8)},
			{ID: "e_m2", Name: "Indie Gig: Local Train", Category: "Concert", WhenLabel: "Sat · 7:30 PM", Venue: "Phoenix Mall", DistanceKm: f(11)},
		}
	case "gaming":
		return []Event{{ID: "e_g1", Name: "Valorant Community Cup", Category: "Tournament", WhenLabel: "Sun · 2:00 PM", Venue: "Pixel Arena", DistanceKm: f(7.1)}}
	case "reading":
		return []Event{{ID: "e_b1", Name: "Author Meet & Signing", Category: "Talk", WhenLabel: "Thu · 6:30 PM", Venue: "Atta Galatta", DistanceKm: f(7)}}
	default:
		return []Event{}
	}
}

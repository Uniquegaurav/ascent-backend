package explore

import (
	"net/http"

	"github.com/kumargaurav/summit-backend/internal/domain"
	"github.com/kumargaurav/summit-backend/internal/httpx"
)

// TrekScope is one location-scoped row (state / country / world).
type TrekScope struct {
	Scope string               `json:"scope"`
	Title string               `json:"title"`
	Items []domain.ExploreItem `json:"items"`
}

type TreksResponse struct {
	State    string      `json:"state"`
	Country  string      `json:"country"`
	Sections []TrekScope `json:"sections"`
}

func trek(id, title, loc string, rating float64, total int) domain.ExploreItem {
	return domain.ExploreItem{
		ID: "trek_" + id, Title: title, Subtitle: loc, Category: "trekking", Kind: "TREK",
		LocationName: loc, Theme: "ALPINE", Rating: rating, RatingsTotal: total,
		Description: title + " — a trekking route worth the climb, near " + loc + ".",
	}.WithSummitCategory()
}

// A pool of distinct, proven mountain/landscape images. Each trek in a row gets a
// different one (assigned by position in init) so no two cards in a row repeat.
var trekImgs = []string{
	"1454496522488-7a8e488e8606", "1486870591958-9b9d0d1dda99", "1518602164578-cd0074062767",
	"1464822759023-fed622ff2c3b", "1506905925346-21bda4d32df4", "1454942901704-3c44c11b2ad1",
	"1469474968028-56623f02e42e", "1483728642387-6c3bdd6c93e5", "1428908728789-d2de25dbd4e2",
	"1418065460487-3e41a6c84dc5", "1501554728187-ce583db33af7", "1500530855697-b586d89ba3ee",
}

var worldTreks = []domain.ExploreItem{
	trek("ebc", "Everest Base Camp", "Nepal", 4.9, 12000),
	trek("inca", "Inca Trail", "Peru", 4.8, 9800),
	trek("tmb", "Tour du Mont Blanc", "France · Italy", 4.8, 7200),
	trek("kili", "Kilimanjaro", "Tanzania", 4.7, 8600),
	trek("torres", "Torres del Paine W", "Chile", 4.9, 5400),
	trek("annapurna", "Annapurna Circuit", "Nepal", 4.8, 6100),
	trek("laugavegur", "Laugavegur Trail", "Iceland", 4.7, 3300),
	trek("fuji", "Mount Fuji", "Japan", 4.6, 9100),
	trek("dolomites", "Alta Via 1", "Dolomites, Italy", 4.8, 2700),
	trek("k2", "K2 Base Camp", "Pakistan", 4.9, 1500),
	trek("gr20", "GR20", "Corsica, France", 4.7, 2100),
	trek("milford", "Milford Track", "New Zealand", 4.8, 4000),
}

var indiaTreks = []domain.ExploreItem{
	trek("valley", "Valley of Flowers", "Uttarakhand", 4.8, 4200),
	trek("kedarkantha", "Kedarkantha", "Uttarakhand", 4.7, 5600),
	trek("roopkund", "Roopkund", "Uttarakhand", 4.6, 3100),
	trek("hampta", "Hampta Pass", "Himachal Pradesh", 4.7, 4800),
	trek("chadar", "Chadar Trek", "Ladakh", 4.8, 2200),
	trek("sandakphu", "Sandakphu", "West Bengal", 4.7, 2600),
	trek("goechala", "Goechala", "Sikkim", 4.8, 1900),
	trek("brahmatal", "Brahmatal", "Uttarakhand", 4.6, 3400),
	trek("tarsar", "Tarsar Marsar", "Kashmir", 4.9, 1700),
	trek("stok", "Stok Kangri", "Ladakh", 4.7, 1200),
	trek("nagtibba", "Nag Tibba", "Uttarakhand", 4.5, 5100),
	trek("dzongri", "Dzongri", "Sikkim", 4.6, 1400),
}

var stateTreks = map[string][]domain.ExploreItem{
	"Karnataka": {
		trek("skandagiri", "Skandagiri", "Chikkaballapur", 4.6, 2100),
		trek("kumara", "Kumara Parvatha", "Kukke Subramanya", 4.7, 1800),
		trek("tadiandamol", "Tadiandamol", "Coorg", 4.6, 1500),
		trek("mullayanagiri", "Mullayanagiri", "Chikkamagaluru", 4.7, 2400),
		trek("kudremukh", "Kudremukh", "Chikkamagaluru", 4.8, 1600),
		trek("savandurga", "Savandurga", "Magadi", 4.4, 900),
	},
	"Maharashtra": {
		trek("kalsubai", "Kalsubai", "Ahmednagar", 4.7, 3100),
		trek("harishchandra", "Harishchandragad", "Ahmednagar", 4.8, 2700),
		trek("rajmachi", "Rajmachi", "Lonavala", 4.6, 2200),
		trek("lohagad", "Lohagad Fort", "Lonavala", 4.5, 4100),
	},
	"Himachal Pradesh": {
		trek("triund", "Triund", "Dharamshala", 4.6, 5200),
		trek("kheerganga", "Kheerganga", "Parvati Valley", 4.7, 4300),
		trek("bhrigu", "Bhrigu Lake", "Manali", 4.7, 1900),
		trek("hamptahp", "Hampta Pass", "Manali", 4.8, 4800),
	},
	"Uttarakhand": {
		trek("valleyuk", "Valley of Flowers", "Chamoli", 4.8, 4200),
		trek("kedarkanthauk", "Kedarkantha", "Uttarkashi", 4.7, 5600),
		trek("nagtibbauk", "Nag Tibba", "Mussoorie", 4.5, 5100),
		trek("chopta", "Chopta Chandrashila", "Rudraprayag", 4.7, 3800),
	},
}

// allTreks indexes every trek by id so /explore/{id} (and ascent creation) can resolve
// a trek the user taps — without it, trek cards were dead ends.
var allTreks = map[string]domain.ExploreItem{}

func init() {
	assignImgs(worldTreks)
	assignImgs(indiaTreks)
	for k := range stateTreks {
		assignImgs(stateTreks[k])
	}
	indexTreks(worldTreks)
	indexTreks(indiaTreks)
	for k := range stateTreks {
		indexTreks(stateTreks[k])
	}
}

func assignImgs(items []domain.ExploreItem) {
	for i := range items {
		if m, ok := trekMetas[items[i].ID]; ok && m.img != "" {
			items[i].ImageURL = img(m.img)
		} else {
			items[i].ImageURL = img(trekImgs[i%len(trekImgs)])
		}
	}
}

func indexTreks(items []domain.ExploreItem) {
	for _, it := range items {
		if _, exists := allTreks[it.ID]; !exists {
			allTreks[it.ID] = it
		}
	}
}

// trekByID resolves a tapped trek card to its full item (for detail + ascent creation).
func trekByID(id string) (domain.ExploreItem, bool) {
	it, ok := allTreks[id]
	return it, ok
}

// stateForLatLng buckets a coordinate into an Indian state via rough bounding boxes.
func stateForLatLng(lat, lng float64) string {
	switch {
	case lat >= 11.5 && lat <= 18.5 && lng >= 74.0 && lng <= 78.6:
		return "Karnataka"
	case lat >= 15.6 && lat <= 22.1 && lng >= 72.6 && lng <= 80.9:
		return "Maharashtra"
	case lat >= 30.2 && lat <= 33.3 && lng >= 75.5 && lng <= 79.0:
		return "Himachal Pradesh"
	case lat >= 28.7 && lat <= 31.5 && lng >= 77.5 && lng <= 81.0:
		return "Uttarakhand"
	default:
		return ""
	}
}

func countryName(code string) string {
	switch code {
	case "IN", "":
		return "India"
	default:
		return code
	}
}

func (h *Handler) Treks(w http.ResponseWriter, r *http.Request) {
	lat, lng := parseLatLng(r)
	cc := r.URL.Query().Get("country")
	state := stateForLatLng(lat, lng)

	stateItems := stateTreks[state]
	stateTitle := "Near you"
	if state != "" && len(stateItems) > 0 {
		stateTitle = "In " + state
	} else {
		// Fallback: surface the country's top treks as the "near you" row.
		stateItems = indiaTreks[:6]
	}

	// Supplement the hardcoded "near you" row with real hiking routes/peaks from
	// OpenStreetMap (Overpass). On any error/empty the provider fails soft and we
	// keep the hardcoded rows above.
	if (lat != 0 || lng != 0) && h.prov != nil && h.prov.Trail != nil {
		if trails := h.prov.Trail.Nearby(r.Context(), lat, lng, 25000); len(trails) > 0 {
			h.persistTrails(trails)
			nearby := make([]domain.ExploreItem, 0, len(trails))
			for _, t := range take2(trails, 12) {
				nearby = append(nearby, trailToItem(t))
			}
			stateItems = append(nearby, stateItems...)
			stateTitle = "Trails near you"
		}
	}

	resp := TreksResponse{
		State:   state,
		Country: countryName(cc),
		Sections: []TrekScope{
			{Scope: "state", Title: stateTitle, Items: stateItems},
			{Scope: "country", Title: "Across " + countryName(cc), Items: indiaTreks},
			{Scope: "world", Title: "Around the world", Items: worldTreks},
		},
	}
	httpx.JSON(w, http.StatusOK, resp)
}

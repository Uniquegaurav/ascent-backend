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

func trek(id, title, loc, imgID string, rating float64, total int) domain.ExploreItem {
	return domain.ExploreItem{
		ID: "trek_" + id, Title: title, Subtitle: loc, Category: "trekking", Kind: "TREK",
		LocationName: loc, ImageURL: img(imgID), Theme: "ALPINE", Rating: rating, RatingsTotal: total,
		Description: "A trekking route worth the climb.",
	}.WithSummitCategory()
}

// A small pool of mountain images, cycled so cards always have art.
var trekImgs = []string{
	"1454496522488-7a8e488e8606", "1486870591958-9b9d0d1dda99", "1518602164578-cd0074062767",
	"1464822759023-fed622ff2c3b", "1506905925346-21bda4d32df4", "1454942901704-3c44c11b2ad1",
}

func trekImg(i int) string { return trekImgs[i%len(trekImgs)] }

var worldTreks = []domain.ExploreItem{
	trek("ebc", "Everest Base Camp", "Nepal", trekImg(0), 4.9, 12000),
	trek("inca", "Inca Trail", "Peru", trekImg(1), 4.8, 9800),
	trek("tmb", "Tour du Mont Blanc", "France · Italy", trekImg(2), 4.8, 7200),
	trek("kili", "Kilimanjaro", "Tanzania", trekImg(3), 4.7, 8600),
	trek("torres", "Torres del Paine W", "Chile", trekImg(4), 4.9, 5400),
	trek("annapurna", "Annapurna Circuit", "Nepal", trekImg(5), 4.8, 6100),
	trek("laugavegur", "Laugavegur Trail", "Iceland", trekImg(0), 4.7, 3300),
	trek("fuji", "Mount Fuji", "Japan", trekImg(1), 4.6, 9100),
	trek("dolomites", "Alta Via 1", "Dolomites, Italy", trekImg(2), 4.8, 2700),
	trek("k2", "K2 Base Camp", "Pakistan", trekImg(3), 4.9, 1500),
	trek("gr20", "GR20", "Corsica, France", trekImg(4), 4.7, 2100),
	trek("milford", "Milford Track", "New Zealand", trekImg(5), 4.8, 4000),
}

var indiaTreks = []domain.ExploreItem{
	trek("valley", "Valley of Flowers", "Uttarakhand", trekImg(0), 4.8, 4200),
	trek("kedarkantha", "Kedarkantha", "Uttarakhand", trekImg(1), 4.7, 5600),
	trek("roopkund", "Roopkund", "Uttarakhand", trekImg(2), 4.6, 3100),
	trek("hampta", "Hampta Pass", "Himachal Pradesh", trekImg(3), 4.7, 4800),
	trek("chadar", "Chadar Trek", "Ladakh", trekImg(4), 4.8, 2200),
	trek("sandakphu", "Sandakphu", "West Bengal", trekImg(5), 4.7, 2600),
	trek("goechala", "Goechala", "Sikkim", trekImg(0), 4.8, 1900),
	trek("brahmatal", "Brahmatal", "Uttarakhand", trekImg(1), 4.6, 3400),
	trek("tarsar", "Tarsar Marsar", "Kashmir", trekImg(2), 4.9, 1700),
	trek("stok", "Stok Kangri", "Ladakh", trekImg(3), 4.7, 1200),
	trek("nagtibba", "Nag Tibba", "Uttarakhand", trekImg(4), 4.5, 5100),
	trek("dzongri", "Dzongri", "Sikkim", trekImg(5), 4.6, 1400),
}

var stateTreks = map[string][]domain.ExploreItem{
	"Karnataka": {
		trek("skandagiri", "Skandagiri", "Chikkaballapur", trekImg(0), 4.6, 2100),
		trek("kumara", "Kumara Parvatha", "Kukke Subramanya", trekImg(1), 4.7, 1800),
		trek("tadiandamol", "Tadiandamol", "Coorg", trekImg(2), 4.6, 1500),
		trek("mullayanagiri", "Mullayanagiri", "Chikkamagaluru", trekImg(3), 4.7, 2400),
		trek("kudremukh", "Kudremukh", "Chikkamagaluru", trekImg(4), 4.8, 1600),
		trek("savandurga", "Savandurga", "Magadi", trekImg(5), 4.4, 900),
	},
	"Maharashtra": {
		trek("kalsubai", "Kalsubai", "Ahmednagar", trekImg(0), 4.7, 3100),
		trek("harishchandra", "Harishchandragad", "Ahmednagar", trekImg(1), 4.8, 2700),
		trek("rajmachi", "Rajmachi", "Lonavala", trekImg(2), 4.6, 2200),
		trek("lohagad", "Lohagad Fort", "Lonavala", trekImg(3), 4.5, 4100),
	},
	"Himachal Pradesh": {
		trek("triund", "Triund", "Dharamshala", trekImg(0), 4.6, 5200),
		trek("kheerganga", "Kheerganga", "Parvati Valley", trekImg(1), 4.7, 4300),
		trek("bhrigu", "Bhrigu Lake", "Manali", trekImg(2), 4.7, 1900),
		trek("hamptahp", "Hampta Pass", "Manali", trekImg(3), 4.8, 4800),
	},
	"Uttarakhand": {
		trek("valleyuk", "Valley of Flowers", "Chamoli", trekImg(0), 4.8, 4200),
		trek("kedarkanthauk", "Kedarkantha", "Uttarkashi", trekImg(1), 4.7, 5600),
		trek("nagtibbauk", "Nag Tibba", "Mussoorie", trekImg(2), 4.5, 5100),
		trek("chopta", "Chopta Chandrashila", "Rudraprayag", trekImg(3), 4.7, 3800),
	},
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

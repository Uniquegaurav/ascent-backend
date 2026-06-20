package explore

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/kumargaurav/summit-backend/internal/domain"
	"github.com/kumargaurav/summit-backend/internal/httpx"
)

func img(id string) string {
	return "https://images.unsplash.com/photo-" + id + "?auto=format&fit=crop&w=1200&q=80"
}

// Catalog is the hardcoded editorial content surfaced on the Explore screen.
var catalog = []domain.ExploreItem{
	{ID: "ex_valley", Title: "Valley of Flowers Trek", Subtitle: "Alpine meadows in bloom", Category: "trekking", Kind: "TREK", Country: "India", LocationName: "Uttarakhand", Theme: "ALPINE", Popularity: 98, ImageURL: img("1454496522488-7a8e488e8606"), Description: "A UNESCO-listed monsoon trek through a valley carpeted with wildflowers, peaking at Hemkund Sahib."},
	{ID: "ex_hampi", Title: "Bouldering at Hampi", Subtitle: "World-class granite", Category: "fitness", Kind: "PLACE", Country: "India", LocationName: "Hampi", Theme: "DESERT", Popularity: 88, ImageURL: img("1522163182402-834f871fd851"), Description: "Sunset boulder fields among ancient ruins — a global climbing pilgrimage."},
	{ID: "ex_triund", Title: "Triund Trek", Subtitle: "Overnight ridge camp", Category: "trekking", Kind: "TREK", Country: "India", LocationName: "Dharamshala", Theme: "ALPINE", Popularity: 91, ImageURL: img("1486870591958-9b9d0d1dda99"), Description: "A beginner-friendly Himalayan trek with a ridgeline campsite under the Dhauladhar range."},
	{ID: "ex_goa_run", Title: "Goa Coastal Run", Subtitle: "Sunrise on the sand", Category: "running", Kind: "HOBBY", Country: "India", LocationName: "Goa", Theme: "OCEAN", Popularity: 72, ImageURL: img("1502904550040-7534597429ae"), Description: "A weekly beachfront running group — soft sand, salt air, easy pace."},
	{ID: "ex_guitar", Title: "Learn Guitar", Subtitle: "From zero to first song", Category: "music", Kind: "HOBBY", Country: "Global", LocationName: "Anywhere", Theme: "EMBER", Popularity: 80, ImageURL: img("1510915361894-db8b60106cb1"), Description: "A 6-week path to playing your first songs, with practice prompts."},
	{ID: "ex_pottery", Title: "Pottery Workshop", Subtitle: "Hands in clay", Category: "art", Kind: "EVENT", Country: "India", LocationName: "Bengaluru", Theme: "AURORA", Popularity: 64, ImageURL: img("1513364776144-60967b0f800f"), Description: "A weekend wheel-throwing intro — leave with your own bowl."},
	{ID: "ex_tokyo", Title: "Tokyo Street Photography", Subtitle: "Neon after dark", Category: "photography", Kind: "DESTINATION", Country: "Japan", LocationName: "Tokyo", Theme: "OCEAN", Popularity: 85, ImageURL: img("1540959733332-eab4deabeeaf"), Description: "Chase reflections and neon through Shinjuku and Shibuya."},
	{ID: "ex_patagonia", Title: "Patagonia W-Trek", Subtitle: "Granite towers", Category: "trekking", Kind: "DESTINATION", Country: "Chile", LocationName: "Torres del Paine", Theme: "ALPINE", Popularity: 95, ImageURL: img("1518602164578-cd0074062767"), Description: "The iconic multi-day circuit beneath the Torres del Paine."},
	{ID: "ex_bookclub", Title: "Sci-fi Book Club", Subtitle: "One novel a month", Category: "reading", Kind: "EVENT", Country: "Global", LocationName: "Online", Theme: "COSMIC", Popularity: 58, ImageURL: img("1512820790803-83ca734da794"), Description: "Read a sci-fi classic each month and meet to dissect it."},
	{ID: "ex_esports", Title: "Weekend Esports Arena", Subtitle: "Squad up", Category: "gaming", Kind: "PLACE", Country: "India", LocationName: "Bengaluru", Theme: "AURORA", Popularity: 70, ImageURL: img("1542751371-adc38448a05e"), Description: "LAN tournaments and casual nights at the city's best arena."},
	{ID: "ex_salsa", Title: "Salsa Classes", Subtitle: "Two left feet welcome", Category: "dance", Kind: "HOBBY", Country: "Global", LocationName: "Anywhere", Theme: "EMBER", Popularity: 66, ImageURL: img("1504609773096-104ff2c73ba4"), Description: "Beginner salsa socials — rhythm first, perfection later."},
	{ID: "ex_iceland", Title: "Iceland Ring Road", Subtitle: "Fire and ice", Category: "travel", Kind: "DESTINATION", Country: "Iceland", LocationName: "Route 1", Theme: "OCEAN", Popularity: 93, ImageURL: img("1504829857797-ddff29c27927"), Description: "Waterfalls, glaciers and black-sand beaches on the loop around Iceland."},
}

func byID(id string) (domain.ExploreItem, bool) {
	for _, it := range catalog {
		if it.ID == id {
			return it, true
		}
	}
	return domain.ExploreItem{}, false
}

func pick(ids ...string) []domain.ExploreItem {
	out := make([]domain.ExploreItem, 0, len(ids))
	for _, id := range ids {
		if it, ok := byID(id); ok {
			out = append(out, it)
		}
	}
	return out
}

// Feed builds the ordered, mixed-layout sections for the Explore screen.
func Feed(country string) domain.ExploreFeed {
	return domain.ExploreFeed{Sections: []domain.ExploreSection{
		{ID: "popular", Title: "Popular near you", Layout: "CARDS", Items: pick("ex_valley", "ex_hampi", "ex_triund", "ex_goa_run")},
		{ID: "treks", Title: "Trending treks", Layout: "CAROUSEL", Items: pick("ex_valley", "ex_triund", "ex_patagonia")},
		{ID: "hobbies", Title: "Start a new hobby", Layout: "GRID", Items: pick("ex_guitar", "ex_pottery", "ex_salsa", "ex_bookclub", "ex_esports", "ex_goa_run")},
		{ID: "world", Title: "Around the world", Layout: "SPOTLIGHT", Items: pick("ex_patagonia", "ex_tokyo", "ex_iceland")},
	}}
}

func Search(q string) []domain.ExploreItem {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return []domain.ExploreItem{}
	}
	out := []domain.ExploreItem{}
	for _, it := range catalog {
		hay := strings.ToLower(it.Title + " " + it.Category + " " + it.LocationName + " " + it.Country + " " + it.Subtitle)
		if strings.Contains(hay, q) {
			out = append(out, it)
		}
	}
	return out
}

// ---- HTTP -----------------------------------------------------------------

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func (h *Handler) Feed(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, Feed(r.URL.Query().Get("country")))
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{"items": Search(r.URL.Query().Get("q"))})
}

func (h *Handler) Item(w http.ResponseWriter, r *http.Request) {
	if it, ok := byID(chi.URLParam(r, "id")); ok {
		httpx.JSON(w, http.StatusOK, it)
		return
	}
	httpx.Error(w, http.StatusNotFound, "not found")
}

// Lookup is exported so the ascent package can build an ascent from an item.
func Lookup(id string) (domain.ExploreItem, bool) { return byID(id) }

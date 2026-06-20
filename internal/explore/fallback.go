package explore

import (
	"strings"

	"github.com/kumargaurav/summit-backend/internal/domain"
)

// Static catalog served when GOOGLE_PLACES_KEY is unset (offline/dev).
var fallbackCatalog = []domain.ExploreItem{
	{ID: "ex_valley", Title: "Valley of Flowers Trek", Subtitle: "Alpine meadows in bloom", Category: "trekking", Kind: "TREK", Country: "India", LocationName: "Uttarakhand", Theme: "ALPINE", ImageURL: img("1454496522488-7a8e488e8606"), Description: "A UNESCO monsoon trek through a valley of wildflowers."},
	{ID: "ex_hampi", Title: "Bouldering at Hampi", Subtitle: "World-class granite", Category: "fitness", Kind: "PLACE", Country: "India", LocationName: "Hampi", Theme: "DESERT", ImageURL: img("1522163182402-834f871fd851"), Description: "Sunset boulder fields among ancient ruins."},
	{ID: "ex_triund", Title: "Triund Trek", Subtitle: "Overnight ridge camp", Category: "trekking", Kind: "TREK", Country: "India", LocationName: "Dharamshala", Theme: "ALPINE", ImageURL: img("1486870591958-9b9d0d1dda99"), Description: "A beginner Himalayan trek with a ridgeline camp."},
	{ID: "ex_patagonia", Title: "Patagonia W-Trek", Subtitle: "Granite towers", Category: "trekking", Kind: "DESTINATION", Country: "Chile", LocationName: "Torres del Paine", Theme: "ALPINE", ImageURL: img("1518602164578-cd0074062767"), Description: "The iconic circuit beneath Torres del Paine."},
	{ID: "ex_tokyo", Title: "Tokyo Street Photography", Subtitle: "Neon after dark", Category: "photography", Kind: "DESTINATION", Country: "Japan", LocationName: "Tokyo", Theme: "OCEAN", ImageURL: img("1540959733332-eab4deabeeaf"), Description: "Chase reflections through Shinjuku and Shibuya."},
}

func fallbackFeed() domain.ExploreFeed {
	pick := func(ids ...string) []domain.ExploreItem {
		out := []domain.ExploreItem{}
		for _, id := range ids {
			if it, ok := fallbackItem(id); ok {
				out = append(out, it)
			}
		}
		return out
	}
	return domain.ExploreFeed{Sections: []domain.ExploreSection{
		{ID: "popular", Title: "Popular near you", Layout: "CARDS", Items: pick("ex_valley", "ex_hampi", "ex_triund")},
		{ID: "treks", Title: "Trending treks", Layout: "CAROUSEL", Items: pick("ex_valley", "ex_triund", "ex_patagonia")},
		{ID: "hobbies", Title: "Start a new hobby", Layout: "GRID", Items: hobbyLaunchers},
		{ID: "world", Title: "Around the world", Layout: "SPOTLIGHT", Items: pick("ex_patagonia", "ex_tokyo")},
	}}
}

func fallbackSearch(q string) []domain.ExploreItem {
	q = strings.ToLower(q)
	out := []domain.ExploreItem{}
	for _, it := range fallbackCatalog {
		if strings.Contains(strings.ToLower(it.Title+" "+it.Category+" "+it.LocationName), q) {
			out = append(out, it)
		}
	}
	return out
}

func fallbackItem(id string) (domain.ExploreItem, bool) {
	for _, it := range fallbackCatalog {
		if it.ID == id {
			return it, true
		}
	}
	return domain.ExploreItem{}, false
}

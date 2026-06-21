package explore

import (
	"strings"

	"github.com/kumargaurav/summit-backend/internal/domain"
)

// Static catalog served when GOOGLE_PLACES_KEY is unset (offline/dev).
var fallbackCatalog = []domain.ExploreItem{
	// Treks / summits
	{ID: "ex_valley", Title: "Valley of Flowers Trek", Subtitle: "Alpine meadows in bloom", Category: "trekking", Kind: "TREK", Country: "India", LocationName: "Uttarakhand", Theme: "ALPINE", Rating: 4.8, RatingsTotal: 2100, ImageURL: img("1454496522488-7a8e488e8606"), Description: "A UNESCO monsoon trek through a valley of wildflowers."},
	{ID: "ex_triund", Title: "Triund Trek", Subtitle: "Overnight ridge camp", Category: "trekking", Kind: "TREK", Country: "India", LocationName: "Dharamshala", Theme: "ALPINE", Rating: 4.6, RatingsTotal: 1500, ImageURL: img("1486870591958-9b9d0d1dda99"), Description: "A beginner Himalayan trek with a ridgeline camp."},
	{ID: "ex_patagonia", Title: "Patagonia W-Trek", Subtitle: "Granite towers", Category: "trekking", Kind: "DESTINATION", Country: "Chile", LocationName: "Torres del Paine", Theme: "ALPINE", Rating: 4.9, RatingsTotal: 980, ImageURL: img("1518602164578-cd0074062767"), Description: "The iconic circuit beneath Torres del Paine."},
	{ID: "ex_hampi", Title: "Bouldering at Hampi", Subtitle: "World-class granite", Category: "climbing", Kind: "PLACE", Country: "India", LocationName: "Hampi", Theme: "DESERT", Rating: 4.7, RatingsTotal: 760, ImageURL: img("1522163182402-834f871fd851"), Description: "Sunset boulder fields among ancient ruins."},
	// Places to explore
	{ID: "ex_toit", Title: "Toit Brewpub", Subtitle: "Craft beer & wood-fired pizza", Category: "restaurant", Kind: "PLACE", Country: "India", LocationName: "Indiranagar", Theme: "EMBER", Rating: 4.5, RatingsTotal: 12000, ImageURL: img("1436076863939-06870fe779c2"), Description: "Bengaluru's iconic microbrewery."},
	{ID: "ex_mtr", Title: "MTR 1924", Subtitle: "Legendary South Indian", Category: "restaurant", Kind: "PLACE", Country: "India", LocationName: "Lalbagh Rd", Theme: "DESERT", Rating: 4.4, RatingsTotal: 8800, ImageURL: img("1517248135467-4c7edcad34c4"), Description: "A century-old institution for dosa and filter coffee."},
	{ID: "ex_tokyo", Title: "Tokyo Street Photography", Subtitle: "Neon after dark", Category: "photography", Kind: "DESTINATION", Country: "Japan", LocationName: "Tokyo", Theme: "OCEAN", Rating: 4.8, RatingsTotal: 540, ImageURL: img("1540959733332-eab4deabeeaf"), Description: "Chase reflections through Shinjuku and Shibuya."},
	// Workshops
	{ID: "ex_pottery", Title: "Pottery Workshop", Subtitle: "Hands-on wheel throwing", Category: "art", Kind: "EVENT", Country: "India", LocationName: "Jayanagar", Theme: "AURORA", Rating: 4.9, RatingsTotal: 220, ImageURL: img("1513475382585-d06e58bcb0e0"), Description: "A 2-hour beginner pottery session."},
	{ID: "ex_cooking", Title: "Cooking Masterclass", Subtitle: "Regional Indian thali", Category: "learning", Kind: "EVENT", Country: "India", LocationName: "Koramangala", Theme: "EMBER", Rating: 4.7, RatingsTotal: 310, ImageURL: img("1556910103-1c02745aae4d"), Description: "Cook a full thali with a home chef."},
	// Unwind
	{ID: "ex_thirdwave", Title: "Third Wave Coffee", Subtitle: "Specialty roasters", Category: "cafe", Kind: "PLACE", Country: "India", LocationName: "HSR Layout", Theme: "FOREST", Rating: 4.3, RatingsTotal: 4200, ImageURL: img("1495474472287-4d71bcdd2085"), Description: "A calm corner for pour-overs and work."},
	{ID: "ex_cubbon", Title: "Cubbon Park", Subtitle: "Green city lungs", Category: "park", Kind: "PLACE", Country: "India", LocationName: "Central Bengaluru", Theme: "FOREST", Rating: 4.6, RatingsTotal: 31000, ImageURL: img("1441974231531-c6227db76b6e"), Description: "A sprawling park for morning walks."},
	{ID: "ex_lalbagh", Title: "Lalbagh Garden", Subtitle: "Botanical retreat", Category: "garden", Kind: "PLACE", Country: "India", LocationName: "Lalbagh", Theme: "FOREST", Rating: 4.6, RatingsTotal: 28000, ImageURL: img("1416879595882-3373a0480b5b"), Description: "A historic botanical garden and glasshouse."},
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
		{ID: "popular", Title: "Popular places near you", Layout: "CARDS", Items: pick("ex_toit", "ex_mtr", "ex_hampi", "ex_valley")},
		{ID: "world", Title: "In the spotlight", Layout: "SPOTLIGHT", Items: pick("ex_patagonia", "ex_tokyo", "ex_valley")},
		{ID: "treks", Title: "Trending treks", Layout: "CAROUSEL", Items: pick("ex_valley", "ex_triund", "ex_patagonia", "ex_hampi")},
		{ID: "workshops", Title: "Trending workshops", Layout: "CAROUSEL", Items: pick("ex_pottery", "ex_cooking")},
		{ID: "hobbies", Title: "Start a new hobby", Layout: "GRID", Items: hobbyLaunchers},
		{ID: "unwind", Title: "Unwind nearby", Layout: "UNWIND", Items: pick("ex_thirdwave", "ex_cubbon", "ex_lalbagh")},
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

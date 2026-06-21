package explore

import (
	"strconv"
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

func pickItems(ids ...string) []domain.ExploreItem {
	out := []domain.ExploreItem{}
	for _, id := range ids {
		if it, ok := fallbackItem(id); ok {
			out = append(out, it.WithSummitCategory())
		}
	}
	return out
}

// fallbackFeed builds the home feed personalized by the user's chosen hobby ids
// (used when Google Places is disabled). Trending becomes per-hobby sub-rows.
func fallbackFeed(hobbyIDs []string) domain.ExploreFeed {
	sections := []domain.ExploreSection{
		{ID: "popular", Title: "Popular places near you", Layout: "CARDS", Items: pickItems("ex_toit", "ex_mtr", "ex_hampi", "ex_valley")},
		{ID: "world", Title: "In the spotlight", Layout: "SPOTLIGHT", Items: pickItems("ex_patagonia", "ex_tokyo", "ex_valley")},
	}
	// Personalized trending sub-rows from the user's hobbies (max 4).
	seen := map[string]bool{}
	count := 0
	for _, id := range hobbyIDs {
		if id == "" || seen[id] || count >= 4 {
			continue
		}
		seen[id] = true
		count++
		sections = append(sections, trendingSection(id, capWord(id)))
	}
	// A default or two so Trending is never thin.
	if !seen["trekking"] {
		sections = append(sections, trendingSection("trekking", "treks"))
	}
	sections = append(sections,
		domain.ExploreSection{ID: "unwind", Title: "Unwind nearby", Layout: "UNWIND", Items: pickItems("ex_thirdwave", "ex_cubbon", "ex_lalbagh", "ex_mtr", "ex_toit")},
	)
	return domain.ExploreFeed{Sections: sections}
}

func capWord(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// ---- personalized trending (no Google key) -------------------------------

// Proven, working Unsplash ids mapped per hobby so trending cards always have art.
var hobbyImageID = map[string]string{
	"trekking": "1454496522488-7a8e488e8606", "adventure": "1518602164578-cd0074062767",
	"travel": "1540959733332-eab4deabeeaf", "photography": "1540959733332-eab4deabeeaf",
	"running": "1441974231531-c6227db76b6e", "football": "1441974231531-c6227db76b6e",
	"fitness": "1545205597-3d9d02c29597", "swimming": "1530549387789-4c1017266635",
	"yoga": "1545205597-3d9d02c29597", "dance": "1504609773096-104ff2c73ba4",
	"music": "1510915361894-db8b60106cb1", "art": "1513364776144-60967b0f800f",
	"reading": "1495474472287-4d71bcdd2085", "writing": "1495474472287-4d71bcdd2085",
	"learning": "1556910103-1c02745aae4d", "gaming": "1540959733332-eab4deabeeaf",
	"community": "1517248135467-4c7edcad34c4", "climbing": "1522163182402-834f871fd851",
}

type seed struct {
	name, loc string
	rating    float64
	total     int
}

var hobbySeeds = map[string][]seed{
	"running":     {{"Cubbon Park Loop", "Central", 4.8, 9200}, {"Kanteerava Track", "City", 4.6, 3100}, {"Agara Lake Trail", "HSR Layout", 4.5, 2400}},
	"fitness":     {{"Cult.fit HSR", "HSR Layout", 4.6, 1800}, {"The Yard CrossFit", "Koramangala", 4.8, 720}, {"Gold's Gym", "Indiranagar", 4.4, 2600}},
	"dance":       {{"Lourd Vijay's Studio", "Residency Rd", 4.7, 1600}, {"The Dance Centre", "Koramangala", 4.6, 840}, {"Shiamak Studio", "Whitefield", 4.5, 620}},
	"music":       {{"The Humming Tree", "Indiranagar", 4.6, 4100}, {"Bangalore School of Music", "RT Nagar", 4.7, 900}, {"Open Mic @ Roxie", "Koramangala", 4.5, 410}},
	"art":         {{"The Pottery Studio", "Jayanagar", 4.9, 430}, {"Chitrakala Parishath", "Kumara Krupa", 4.7, 2200}, {"Ochre Art Studio", "Indiranagar", 4.6, 310}},
	"photography": {{"Lalbagh Photo Walk", "Lalbagh", 4.6, 2800}, {"Nandi Hills Sunrise", "Chikkaballapur", 4.5, 9000}, {"Bangalore Palace", "Vasanth Nagar", 4.4, 1500}},
	"reading":     {{"Blossom Book House", "Church St", 4.8, 7400}, {"Atta Galatta", "Koramangala", 4.6, 2100}, {"Champaca Books", "Vasanth Nagar", 4.7, 1200}},
	"gaming":      {{"Smaaash", "Mantri Mall", 4.3, 5600}, {"Player's Lounge", "HSR Layout", 4.5, 420}, {"Dave & Buster's", "Phoenix Mall", 4.4, 980}},
	"football":    {{"PlayArena Turf", "Sarjapur Rd", 4.6, 3400}, {"Just Play Turf", "HSR Layout", 4.5, 900}, {"Powerplay Arena", "Bellandur", 4.4, 560}},
	"trekking":    {{"Skandagiri Trail", "Nandi Hills", 4.7, 2100}, {"Savandurga Trek", "Magadi", 4.5, 860}, {"Anthargange Caves", "Kolar", 4.4, 540}},
	"learning":    {{"Cooking Masterclass", "Koramangala", 4.7, 310}, {"Maker's Asylum", "HSR Layout", 4.6, 180}, {"Pottery 101", "Jayanagar", 4.8, 240}},
	"travel":      {{"Coorg Retreat", "Madikeri", 4.7, 5200}, {"Hampi Heritage Walk", "Hampi", 4.8, 8100}, {"Gokarna Beaches", "Gokarna", 4.6, 3400}},
}

// trendingSection builds a personalized "Trending <label>" carousel for one hobby.
func trendingSection(hobbyID, label string) domain.ExploreSection {
	imgID := hobbyImageID[hobbyID]
	if imgID == "" {
		imgID = "1500530855697-b586d89ba3ee"
	}
	theme := domain.ClassifyCategory(hobbyID, "HOBBY")
	seeds := hobbySeeds[hobbyID]
	if len(seeds) == 0 {
		seeds = []seed{{label + " Studio", "Indiranagar", 4.6, 420}, {label + " Collective", "Koramangala", 4.5, 260}, {label + " Club", "HSR Layout", 4.4, 180}}
	}
	items := make([]domain.ExploreItem, 0, len(seeds))
	for i, s := range seeds {
		items = append(items, domain.ExploreItem{
			ID: "t_" + hobbyID + "_" + strconv.Itoa(i), Title: s.name, Category: hobbyID, Kind: "PLACE",
			LocationName: s.loc, ImageURL: img(imgID), Rating: s.rating, RatingsTotal: s.total,
			Theme: themeForCategory(theme),
		}.WithSummitCategory())
	}
	return domain.ExploreSection{ID: "trending_" + hobbyID, Title: "Trending " + label, Layout: "CAROUSEL", Items: items}
}

func themeForCategory(summitCat string) string {
	switch summitCat {
	case domain.CategorySummit:
		return "ALPINE"
	case domain.CategoryUnwind:
		return "FOREST"
	case domain.CategoryExplore:
		return "OCEAN"
	default:
		return "EMBER"
	}
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

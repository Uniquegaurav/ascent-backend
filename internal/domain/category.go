package domain

import "strings"

// SummitCategory values returned to the client.
const (
	CategoryHobby   = "HOBBY"
	CategoryExplore = "EXPLORE"
	CategorySummit  = "SUMMIT"
	CategoryUnwind  = "UNWIND"
)

var (
	summitWords  = []string{"trek", "hike", "hiking", "mountain", "climb", "summit", "trail", "travel", "adventure", "camp", "expedition"}
	unwindWords  = []string{"cafe", "café", "coffee", "park", "garden", "spa", "relax", "unwind", "beach", "lounge", "tea"}
	exploreWords = []string{"restaurant", "dining", "dine", "food", "place", "store", "shop", "museum", "bar", "pub", "brewery", "movie", "event", "gallery", "market"}
)

// ClassifyCategory maps a free-form category string (+ kind) onto one of the four
// pursuit buckets. Single source of truth for category derivation — the client reads
// the resulting `summitCategory` field rather than classifying anything itself.
func ClassifyCategory(category, kind string) string {
	c := strings.ToLower(category)
	switch {
	case containsAny(c, summitWords):
		return CategorySummit
	case containsAny(c, unwindWords):
		return CategoryUnwind
	case containsAny(c, exploreWords):
		return CategoryExplore
	case strings.EqualFold(kind, "HOBBY"):
		return CategoryHobby
	case strings.EqualFold(kind, "PLACE"):
		return CategoryExplore
	default:
		return CategoryHobby
	}
}

func containsAny(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

// ---- Catalog DTOs ---------------------------------------------------------

type City struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	CountryCode string  `json:"countryCode"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	Theme       string  `json:"theme"`
}

// CategoryInfo is the display metadata for a pursuit category (so the client
// renders labels/actions from the backend instead of hardcoding them).
type CategoryInfo struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Plural       string `json:"plural"`
	ActionLabel  string `json:"actionLabel"`
	DoneLabel    string `json:"doneLabel"`
	DefaultTheme string `json:"defaultTheme"`
}

// Categories is the canonical list served at /catalog/categories.
var Categories = []CategoryInfo{
	{ID: CategoryHobby, Label: "Hobby", Plural: "Hobbies", ActionLabel: "Log a session", DoneLabel: "Logged", DefaultTheme: "EMBER"},
	{ID: CategoryExplore, Label: "Explore", Plural: "Places", ActionLabel: "Mark explored", DoneLabel: "Explored", DefaultTheme: "OCEAN"},
	{ID: CategorySummit, Label: "Summit", Plural: "Summits", ActionLabel: "Mark summited", DoneLabel: "Summited", DefaultTheme: "ALPINE"},
	{ID: CategoryUnwind, Label: "Unwind", Plural: "Unwind", ActionLabel: "Save to unwind", DoneLabel: "Saved", DefaultTheme: "FOREST"},
}

// HobbyPrefs is the user's customised home hobbies: ordered ids + theme overrides.
// Mirrors the client's HobbyPrefs so no parsing changes are needed.
type HobbyPrefs struct {
	HobbyIDs       []string          `json:"hobbyIds"`
	ThemeOverrides map[string]string `json:"themeOverrides"`
}

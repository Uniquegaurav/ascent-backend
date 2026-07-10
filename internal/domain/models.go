package domain

// JSON tags intentionally mirror the Kotlin app's DTOs so the client needs no
// parsing changes when it switches from the mock engine to this server.

type User struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Phone       string   `json:"phone"`
	Onboarded   bool     `json:"onboarded"`
	AvatarHue   float64  `json:"avatarHue"`
	AvatarURL   string   `json:"avatarUrl"`
	InterestIDs []string `json:"interestIds"`
}

type Interest struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Emoji    string `json:"emoji"`
	Theme    string `json:"theme"`
	ImageURL string `json:"imageUrl"`
	Vibe     string `json:"vibe"`
}

type Peak struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Emoji       string  `json:"emoji"`
	InterestID  string  `json:"interestId"`
	Theme       string  `json:"theme"`
	Progress    float64 `json:"progress"`
	MomentCount int     `json:"momentCount"`
	ImageURL    string  `json:"imageUrl"`
}

type Milestone struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Emoji       string  `json:"emoji"`
	Reached     bool    `json:"reached"`
	Progress    float64 `json:"progress"`
}

type Range struct {
	ClimberName  string      `json:"climberName"`
	Peaks        []Peak      `json:"peaks"`
	Milestones   []Milestone `json:"milestones"`
	TotalMoments int         `json:"totalMoments"`
}

type GeoLocation struct {
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
}

type Experience struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Category    string       `json:"category"`
	InterestID  string       `json:"interestId"`
	ImageURLs   []string     `json:"imageUrls"`
	Location    *GeoLocation `json:"location"`
	DateEpochMs int64        `json:"dateEpochMs"`
	MoodScore   int          `json:"moodScore"`
}

type Friend struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	AvatarHue     float64 `json:"avatarHue"`
	Status        string  `json:"status"`
	PeaksExplored int     `json:"peaksExplored"`
	LastAscent    *string `json:"lastAscent"`
}

type Challenge struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Emoji       string `json:"emoji"`
	InterestID  string `json:"interestId"`
	Target      int    `json:"target"`
	Progress    int    `json:"progress"`
	Unit        string `json:"unit"`
	Joined      bool   `json:"joined"`
}

// ---- Explore + Ascents (the new model) -----------------------------------

type ExploreItem struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Subtitle       string  `json:"subtitle"`
	Category       string  `json:"category"`
	Kind           string  `json:"kind"` // TREK | PLACE | EVENT | HOBBY | DESTINATION
	Country        string  `json:"country"`
	LocationName   string  `json:"locationName"`
	ImageURL       string  `json:"imageUrl"`
	Theme          string  `json:"theme"`
	Description    string  `json:"description"`
	Popularity     int     `json:"popularity"`
	Rating         float64 `json:"rating"`
	RatingsTotal   int     `json:"ratingsTotal"`
	PlaceID        string  `json:"placeId"`
	SearchQuery    string  `json:"searchQuery"`    // for HOBBY launchers
	SummitCategory string  `json:"summitCategory"` // HOBBY | EXPLORE | SUMMIT | UNWIND (server-derived)
}

// WithSummitCategory fills SummitCategory from the item's category/kind.
func (e ExploreItem) WithSummitCategory() ExploreItem {
	e.SummitCategory = ClassifyCategory(e.Category, e.Kind)
	return e
}

type ExploreReview struct {
	Author string  `json:"author"`
	Rating float64 `json:"rating"`
	Text   string  `json:"text"`
	When   string  `json:"when"`
}

// Fact is a labelled key fact (e.g. "Max altitude" → "5,364 m") shown on a detail screen.
type Fact struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type ExploreDetail struct {
	Item    ExploreItem     `json:"item"`
	Photos  []string        `json:"photos"`
	Reviews []ExploreReview `json:"reviews"`
	Address string          `json:"address"`
	Phone   string          `json:"phone"`
	Website string          `json:"website"`
	Hours   []string        `json:"hours"`
	// Facts is structured key info (treks: difficulty/altitude/season/…).
	Facts []Fact `json:"facts"`
	// InfoURL links out to an authoritative guide (IndiaHikes / AllTrails / official).
	InfoURL string `json:"infoUrl"`
	// Sources credits external content providers (Wikipedia / Wikimedia Commons)
	// when their extract/images are used, for attribution.
	Sources []string `json:"sources"`
}

type ExploreSection struct {
	ID     string        `json:"id"`
	Title  string        `json:"title"`
	Layout string        `json:"layout"` // CAROUSEL | GRID | CARDS | SPOTLIGHT
	Items  []ExploreItem `json:"items"`
}

type ExploreFeed struct {
	Sections []ExploreSection `json:"sections"`
}

type Ascent struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Category       string `json:"category"`
	Theme          string `json:"theme"`
	ImageURL       string `json:"imageUrl"`
	Kind           string `json:"kind"`
	LocationName   string `json:"locationName"`
	Status         string `json:"status"` // EXPLORING | ACTIVE | SUMMITED
	LogCount       int    `json:"logCount"`
	CreatedAtMs    int64  `json:"createdAtMs"`
	SummitCategory string `json:"summitCategory"` // HOBBY | EXPLORE | SUMMIT | UNWIND (server-derived)
	// My-Ascent hierarchy: ParentID set ⇒ a child item under a hobby parent;
	// InterestID is the hobby this ascent belongs to (for grouping/theming).
	ParentID   *string `json:"parentId"`
	InterestID *string `json:"interestId"`
}

// WithSummitCategory fills SummitCategory from the ascent's category/kind.
func (a Ascent) WithSummitCategory() Ascent {
	a.SummitCategory = ClassifyCategory(a.Category, a.Kind)
	return a
}

type Log struct {
	ID          string            `json:"id"`
	AscentID    string            `json:"ascentId"`
	Title       string            `json:"title"`
	Note        string            `json:"note"`
	MoodScore   int               `json:"moodScore"`
	ImageURLs   []string          `json:"imageUrls"`
	Location    *GeoLocation      `json:"location"`
	DateEpochMs int64             `json:"dateEpochMs"`
	Metrics     map[string]string `json:"metrics"`
}

type AscentDetail struct {
	Ascent Ascent `json:"ascent"`
	Logs   []Log  `json:"logs"`
}

// Wishlist is an Explore place the climber wants to visit. The JSON shape mirrors the
// Kotlin app's WishlistItemDto so the client needs no parsing changes.
type Wishlist struct {
	ID               string      `json:"id"`
	Item             ExploreItem `json:"item"`
	PlannedDateMs    *int64      `json:"plannedDateMs"`
	BookingURL       string      `json:"bookingUrl"`
	AddedToCalendar  bool        `json:"addedToCalendar"`
	InvitedFriendIDs []string    `json:"invitedFriendIds"`
	CreatedAtMs      int64       `json:"createdAtMs"`
}

type FeedItem struct {
	ID               string  `json:"id"`
	AuthorName       string  `json:"authorName"`
	AuthorHue        float64 `json:"authorHue"`
	IsSelf           bool    `json:"isSelf"`
	Title            string  `json:"title"`
	Description      string  `json:"description"`
	Category         string  `json:"category"`
	Location         *string `json:"location"`
	TimestampEpochMs int64   `json:"timestampEpochMs"`
	Reactions        int     `json:"reactions"`
	ReactedByMe      bool    `json:"reactedByMe"`
}

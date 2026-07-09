// Package hobby serves a rich, per-hobby "guide" — overview, skill levels, a starter
// plan, target metrics, things to watch, and useful links/integrations (Strava,
// AllTrails, …). Content is curated in Go (no DB row needed); hobbies without a
// curated entry get a sensible generic guide. The client overlays live "nearby"
// spots from the discovery API on top of this.
package hobby

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kumargaurav/summit-backend/internal/httpx"
)

// Section types the client knows how to render.
const (
	SectionTips    = "TIPS"    // title + detail bullets
	SectionLevels  = "LEVELS"  // skill tiers (title + detail)
	SectionPlan    = "PLAN"    // numbered steps (title + detail)
	SectionMetrics = "METRICS" // value + label chips (value + title + detail)
	SectionVideos  = "VIDEOS"  // watch & learn (title + subtitle + url)
	SectionLinks   = "LINKS"   // apps & integrations (title + subtitle + url)
)

type Item struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	Detail   string `json:"detail,omitempty"`
	URL      string `json:"url,omitempty"`
	VideoID  string `json:"videoId,omitempty"`
	Value    string `json:"value,omitempty"`
}

type Section struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	Items []Item `json:"items"`
}

type Guide struct {
	InterestID   string    `json:"interestId"`
	Title        string    `json:"title"`
	Emoji        string    `json:"emoji"`
	Theme        string    `json:"theme"`
	HeroImageURL string    `json:"heroImageUrl"`
	Tagline      string    `json:"tagline"`
	About        string    `json:"about"`
	Sections     []Section `json:"sections"`
}

// body is the curated content; meta (title/emoji/theme/image) is filled from the
// interests table so it stays consistent with the catalog.
type body struct {
	Tagline  string
	About    string
	Sections []Section
}

// ---- Repository -----------------------------------------------------------

type Repo struct{ pool *pgxpool.Pool }

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

func (r *Repo) Guide(ctx context.Context, id string) (Guide, error) {
	var label, emoji, theme, image string
	err := r.pool.QueryRow(ctx,
		`SELECT label, emoji, theme, image_url FROM interests WHERE id = $1`, id).
		Scan(&label, &emoji, &theme, &image)
	if errors.Is(err, pgx.ErrNoRows) {
		label, emoji, theme, image = titleize(id), "🧭", "ALPINE", ""
	} else if err != nil {
		return Guide{}, err
	}

	b, ok := curated[id]
	if !ok {
		b = generic(label)
	}
	return Guide{
		InterestID: id, Title: label, Emoji: emoji, Theme: theme, HeroImageURL: image,
		Tagline: b.Tagline, About: b.About, Sections: b.Sections,
	}, nil
}

// ---- Picks (curated discovery list, not location-based) -------------------

// Pick is one curated item in a hobby's discovery list (a book, an art form, a sport…).
type Pick struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	Meta     string `json:"meta,omitempty"`
}

// Picks is a named, hand-curated collection for a hobby — e.g. "Popular books" for reading,
// "Art forms" for art. Surfaced in the Ascent tab as a list (not a map of nearby places).
type Picks struct {
	InterestID string `json:"interestId"`
	Title      string `json:"title"`
	Items      []Pick `json:"items"`
}

func (r *Repo) Picks(id string) Picks {
	p, ok := picks[id]
	if !ok {
		return Picks{InterestID: id, Title: "", Items: []Pick{}}
	}
	p.InterestID = id
	return p
}

func (h *Handler) Picks(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	httpx.JSON(w, http.StatusOK, h.repo.Picks(id))
}

// ---- HTTP -----------------------------------------------------------------

type Handler struct{ repo *Repo }

func NewHandler(repo *Repo) *Handler { return &Handler{repo: repo} }

func (h *Handler) Guide(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		httpx.Error(w, http.StatusBadRequest, "missing hobby id")
		return
	}
	g, err := h.repo.Guide(r.Context(), id)
	if err != nil {
		httpx.Internal(w, r, err, "could not load hobby guide")
		return
	}
	httpx.JSON(w, http.StatusOK, g)
}

// ---- Helpers --------------------------------------------------------------

// yt returns a YouTube search URL for a query — resolves to relevant videos even
// without hardcoding (potentially stale) video ids.
func yt(query string) string {
	return "https://www.youtube.com/results?search_query=" + url.QueryEscape(query)
}

func titleize(id string) string {
	if id == "" {
		return "Hobby"
	}
	return strings.ToUpper(id[:1]) + id[1:]
}

func generic(label string) body {
	low := strings.ToLower(label)
	return body{
		Tagline: "Start where you are. Get a little better each week.",
		About:   "A quick guide to getting started with " + low + " — how to begin, how to improve, and where to find people and places near you.",
		Sections: []Section{
			{Type: SectionTips, Title: "Getting started", Items: []Item{
				{Title: "Start small", Detail: "Pick one easy session this week and just show up. Consistency beats intensity."},
				{Title: "Find your people", Detail: "Look for a local group or class — it makes " + low + " far more fun and sticky."},
				{Title: "Track it", Detail: "Log each session in Ascent so you can see your progress build over time."},
			}},
			{Type: SectionVideos, Title: "Watch & learn", Items: []Item{
				{Title: label + " for beginners", Subtitle: "YouTube", URL: yt(label + " for beginners")},
				{Title: label + " tips & technique", Subtitle: "YouTube", URL: yt(label + " tips technique")},
			}},
		},
	}
}

// ---- Curated guides -------------------------------------------------------

var curated = map[string]body{
	"running": {
		Tagline: "From first jog to finish line.",
		About:   "Running needs almost no gear and pays back fast. Build an aerobic base with easy runs, add one faster session a week, and let distance grow ~10% at a time.",
		Sections: []Section{
			{Type: SectionMetrics, Title: "Distance & pace goals", Items: []Item{
				{Value: "5K", Title: "Beginner goal", Detail: "~30–40 min"},
				{Value: "10K", Title: "Intermediate", Detail: "~55–70 min"},
				{Value: "21.1K", Title: "Half marathon", Detail: "~2:00–2:30"},
				{Value: "42.2K", Title: "Marathon", Detail: "~4:00–5:00"},
			}},
			{Type: SectionPlan, Title: "Couch → 5K (8 weeks)", Items: []Item{
				{Title: "Weeks 1–2", Detail: "Run 1 min / walk 2 min, repeat 8×. Three days a week."},
				{Title: "Weeks 3–4", Detail: "Run 3 min / walk 1 min. Add a 4th easy day if it feels good."},
				{Title: "Weeks 5–6", Detail: "Run 8–10 min blocks with short walk breaks. Keep it conversational."},
				{Title: "Weeks 7–8", Detail: "Run 25–30 min continuous. You're a 5K runner."},
			}},
			{Type: SectionLevels, Title: "Effort zones", Items: []Item{
				{Title: "Easy (Zone 2)", Detail: "Can hold a conversation. 80% of your running lives here."},
				{Title: "Tempo", Detail: "Comfortably hard, ~1 session/week. Builds your sustainable pace."},
				{Title: "Intervals", Detail: "Short, fast reps with recovery. Sharpens speed — use sparingly."},
			}},
			{Type: SectionLinks, Title: "Apps & integrations", Items: []Item{
				{Title: "Connect Strava", Subtitle: "Sync runs, segments & friends", URL: "https://www.strava.com/"},
				{Title: "Nike Run Club", Subtitle: "Guided runs & plans", URL: "https://www.nike.com/nrc-app"},
			}},
			{Type: SectionVideos, Title: "Watch & learn", Items: []Item{
				{Title: "Proper running form", Subtitle: "Technique", URL: yt("proper running form for beginners")},
				{Title: "Breathing while running", Subtitle: "Technique", URL: yt("how to breathe while running")},
				{Title: "Couch to 5K — week 1", Subtitle: "Follow along", URL: yt("couch to 5k week 1 run along")},
			}},
		},
	},
	"fitness": {
		Tagline: "Strong is built one session at a time.",
		About:   "A simple, sustainable strength routine beats a perfect one you can't keep. Train 3×/week, hit the big movements, progress the load slowly, and rest.",
		Sections: []Section{
			{Type: SectionLevels, Title: "Find your level", Items: []Item{
				{Title: "Beginner", Detail: "New or returning. Master bodyweight + form: squat, hinge, push, pull."},
				{Title: "Intermediate", Detail: "6+ months consistent. Add barbells and progressive overload."},
				{Title: "Advanced", Detail: "Years in. Periodise, target weak points, manage fatigue."},
			}},
			{Type: SectionPlan, Title: "Full-body 3-day split", Items: []Item{
				{Title: "Day A", Detail: "Squat · Bench/Push-up · Row — 3×8–10"},
				{Title: "Day B", Detail: "Deadlift/Hinge · Overhead press · Pull-up — 3×6–8"},
				{Title: "Day C", Detail: "Lunge · Incline press · Lat pulldown + core — 3×10–12"},
			}},
			{Type: SectionMetrics, Title: "Starter targets", Items: []Item{
				{Value: "3×", Title: "Sessions / week", Detail: "Leave a rest day between"},
				{Value: "8–12", Title: "Reps in reserve", Detail: "Stop 1–2 reps short of failure"},
				{Value: "+2.5kg", Title: "Weekly progress", Detail: "Add load when reps feel easy"},
			}},
			{Type: SectionLinks, Title: "Apps & integrations", Items: []Item{
				{Title: "Connect Strava", Subtitle: "Log workouts & share", URL: "https://www.strava.com/"},
				{Title: "Hevy — workout tracker", Subtitle: "Log sets, reps & PRs", URL: "https://www.hevyapp.com/"},
			}},
			{Type: SectionVideos, Title: "Watch & learn", Items: []Item{
				{Title: "Perfect squat form", Subtitle: "Technique", URL: yt("how to squat with proper form")},
				{Title: "Beginner full-body workout", Subtitle: "Follow along", URL: yt("beginner full body workout at gym")},
				{Title: "Progressive overload explained", Subtitle: "Concept", URL: yt("progressive overload explained")},
			}},
		},
	},
	"trekking": {
		Tagline: "The mountains are calling.",
		About:   "Trekking rewards preparation. Match the trail to your fitness, layer for the weather, break in your boots, and never underestimate water and daylight.",
		Sections: []Section{
			{Type: SectionLevels, Title: "Trail grades", Items: []Item{
				{Title: "Easy", Detail: "Day hikes, well-marked, <500m gain. Great first treks."},
				{Title: "Moderate", Detail: "1–3 days, 500–1200m/day. Some steep or rocky sections."},
				{Title: "Difficult", Detail: "High altitude / multi-day. Needs fitness, acclimatisation & a guide."},
			}},
			{Type: SectionPlan, Title: "Plan your trek", Items: []Item{
				{Title: "1 — Pick the trail", Detail: "Match distance & altitude to your level and the season."},
				{Title: "2 — Check the window", Detail: "Weather, permits, and daylight. Start early, turn back on time."},
				{Title: "3 — Pack smart", Detail: "Layers, 2–3L water, snacks, first-aid, headlamp, map offline."},
				{Title: "4 — Train", Detail: "Stair/incline walks with a loaded pack for 3–4 weeks before."},
			}},
			{Type: SectionMetrics, Title: "On-trail rules of thumb", Items: []Item{
				{Value: "3–4 km/h", Title: "Avg pace", Detail: "Add time for big climbs"},
				{Value: "0.5–1 L/hr", Title: "Water", Detail: "More in heat / altitude"},
				{Value: "+300m", Title: "Sleep height/day", Detail: "Above 3000m, go slow"},
			}},
			{Type: SectionLinks, Title: "Apps & maps", Items: []Item{
				{Title: "AllTrails", Subtitle: "Trail maps, reviews & GPS", URL: "https://www.alltrails.com/"},
				{Title: "Komoot — route planner", Subtitle: "Plan & download offline", URL: "https://www.komoot.com/"},
			}},
			{Type: SectionVideos, Title: "Watch & learn", Items: []Item{
				{Title: "Trekking for beginners", Subtitle: "Essentials", URL: yt("trekking for beginners guide")},
				{Title: "What to pack for a trek", Subtitle: "Gear", URL: yt("trekking packing list essentials")},
				{Title: "Altitude sickness basics", Subtitle: "Safety", URL: yt("how to avoid altitude sickness trekking")},
			}},
		},
	},
	"dance": {
		Tagline: "Find your rhythm.",
		About:   "Every style starts with the basic step and the count. Pick a style that excites you, drill the fundamentals slowly, then speed up to music.",
		Sections: []Section{
			{Type: SectionLevels, Title: "Popular styles", Items: []Item{
				{Title: "Hip-hop", Detail: "Groove, bounce & isolations. High-energy and beginner-friendly."},
				{Title: "Salsa / Bachata", Detail: "Partner social dances on an 8-count. Huge community scene."},
				{Title: "Contemporary", Detail: "Fluid, expressive, floor work. Builds control and lines."},
				{Title: "Classical (Bharatanatyam/Kathak)", Detail: "Rooted technique, rhythm & storytelling."},
			}},
			{Type: SectionPlan, Title: "Your first month", Items: []Item{
				{Title: "Week 1", Detail: "Learn the basic step & timing slowly, in front of a mirror."},
				{Title: "Week 2", Detail: "Add arms/isolations. Drill 15 min daily to a slow track."},
				{Title: "Week 3", Detail: "Join a beginner class or social night — dancing with others accelerates everything."},
				{Title: "Week 4", Detail: "String moves into an 8-count combo at full tempo."},
			}},
			{Type: SectionLinks, Title: "Where to learn", Items: []Item{
				{Title: "STEEZY Studio", Subtitle: "Online classes by style & level", URL: "https://www.steezy.co/"},
				{Title: "Find studios nearby", Subtitle: "See the map below", URL: ""},
			}},
			{Type: SectionVideos, Title: "Watch & learn", Items: []Item{
				{Title: "Hip-hop basics for beginners", Subtitle: "Hip-hop", URL: yt("hip hop dance basics for beginners")},
				{Title: "Salsa basic steps", Subtitle: "Salsa", URL: yt("salsa basic steps for beginners")},
				{Title: "Contemporary foundations", Subtitle: "Contemporary", URL: yt("contemporary dance for beginners")},
			}},
		},
	},
	"gaming": {
		Tagline: "Play more deliberately.",
		About:   "Improving at games is a skill of its own — pick a title, learn its fundamentals, review your own play, and practise with intent rather than just grinding.",
		Sections: []Section{
			{Type: SectionLevels, Title: "Pick your lane", Items: []Item{
				{Title: "FPS (Valorant/CS2)", Detail: "Aim, crosshair placement, map control & comms."},
				{Title: "MOBA (Dota 2/LoL)", Detail: "Last-hitting, map awareness, objectives & roles."},
				{Title: "Strategy / Sim", Detail: "Build orders, economy & long-term planning."},
				{Title: "Cozy / Story", Detail: "Just for joy — no leaderboard required."},
			}},
			{Type: SectionPlan, Title: "Improve with intent", Items: []Item{
				{Title: "1 — One game", Detail: "Go deep on a single title instead of spreading thin."},
				{Title: "2 — Warm up", Detail: "10 min aim trainer / practice tool before ranked."},
				{Title: "3 — Review", Detail: "Watch one of your own replays and find a single mistake to fix."},
				{Title: "4 — VOD review", Detail: "Watch a pro of your role and copy one habit per week."},
			}},
			{Type: SectionLinks, Title: "Tools & community", Items: []Item{
				{Title: "Discord — find a squad", Subtitle: "Communities for every game", URL: "https://discord.com/"},
				{Title: "Twitch — learn from pros", Subtitle: "Live high-level play", URL: "https://www.twitch.tv/"},
			}},
			{Type: SectionVideos, Title: "Watch & learn", Items: []Item{
				{Title: "Aim training that works", Subtitle: "FPS", URL: yt("fps aim training guide improve")},
				{Title: "Game sense explained", Subtitle: "Fundamentals", URL: yt("how to improve game sense")},
				{Title: "How to review your replays", Subtitle: "Improvement", URL: yt("how to review your own gameplay replays")},
			}},
		},
	},
	"music": {
		Tagline: "Make a little noise every day.",
		About:   "Pick an instrument, learn a few chords, and play songs you love — musicality grows fastest when practice is fun and frequent.",
		Sections: []Section{
			{Type: SectionLevels, Title: "Where to begin", Items: []Item{
				{Title: "Guitar", Detail: "4 open chords unlock hundreds of songs. Start with G–C–D–Em."},
				{Title: "Piano / Keys", Detail: "Learn the C major scale & basic triads. Great for theory."},
				{Title: "Singing", Detail: "Breath support & pitch matching. Record yourself to track progress."},
				{Title: "Production", Detail: "A free DAW + loops — make a 16-bar beat this week."},
			}},
			{Type: SectionPlan, Title: "First 30 days", Items: []Item{
				{Title: "Week 1", Detail: "Learn 2 chords & switch between them cleanly."},
				{Title: "Week 2", Detail: "Add a 3rd chord and strum a simple song."},
				{Title: "Week 3", Detail: "Play along to a track — keep time over perfection."},
				{Title: "Week 4", Detail: "Perform one full song, even just for yourself."},
			}},
			{Type: SectionLinks, Title: "Apps & tools", Items: []Item{
				{Title: "Yousician", Subtitle: "Guided lessons with feedback", URL: "https://yousician.com/"},
				{Title: "Ultimate Guitar — tabs", Subtitle: "Chords for any song", URL: "https://www.ultimate-guitar.com/"},
			}},
			{Type: SectionVideos, Title: "Watch & learn", Items: []Item{
				{Title: "4 chords, many songs", Subtitle: "Guitar", URL: yt("4 chords songs guitar beginner")},
				{Title: "Piano for absolute beginners", Subtitle: "Piano", URL: yt("piano for absolute beginners lesson 1")},
			}},
		},
	},
	"photography": {
		Tagline: "See light differently.",
		About:   "Great photos come from light and timing, not just gear. Learn the exposure triangle, shoot every day, and study why images you love work.",
		Sections: []Section{
			{Type: SectionLevels, Title: "Master the basics", Items: []Item{
				{Title: "Aperture", Detail: "Controls depth of field — blurry vs sharp backgrounds."},
				{Title: "Shutter speed", Detail: "Freezes or blurs motion. Fast for action, slow for trails."},
				{Title: "ISO", Detail: "Sensor sensitivity. Keep it low for clean images."},
			}},
			{Type: SectionPlan, Title: "Practice projects", Items: []Item{
				{Title: "Golden hour walk", Detail: "Shoot only in the hour after sunrise / before sunset."},
				{Title: "One lens, one week", Detail: "Constraint forces creativity & composition."},
				{Title: "Edit 5 frames", Detail: "Pick your best 5 and edit consistently."},
			}},
			{Type: SectionLinks, Title: "Tools", Items: []Item{
				{Title: "Lightroom Mobile", Subtitle: "Edit on the go (free)", URL: "https://www.adobe.com/products/photoshop-lightroom.html"},
				{Title: "Unsplash — study great photos", Subtitle: "Inspiration", URL: "https://unsplash.com/"},
			}},
			{Type: SectionVideos, Title: "Watch & learn", Items: []Item{
				{Title: "Exposure triangle explained", Subtitle: "Fundamentals", URL: yt("exposure triangle explained photography")},
				{Title: "Composition tips", Subtitle: "Technique", URL: yt("photography composition tips for beginners")},
			}},
		},
	},
}

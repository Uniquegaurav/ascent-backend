package explore

import (
	"net/url"

	"github.com/kumargaurav/summit-backend/internal/domain"
)

// tmeta is curated, factual detail for a trek (shown on the detail screen). Photos are
// representative mountain imagery; the InfoURL (computed) links to the authoritative
// guide (IndiaHikes / AllTrails) that hosts the exact route photos & day-by-day plan.
type tmeta struct {
	img   string
	desc  string
	facts []domain.Fact
}

func f(label, value string) domain.Fact { return domain.Fact{Label: label, Value: value} }

// trekMetas is keyed by full item id ("trek_<id>").
var trekMetas = map[string]tmeta{
	// ---- World ----
	"trek_ebc": {img: "1486870591958-9b9d0d1dda99",
		desc: "The legendary trek to the foot of Everest (8,849 m), through Sherpa villages, Tengboche monastery and the Khumbu icefall.",
		facts: []domain.Fact{f("Difficulty", "Challenging"), f("Max altitude", "5,364 m (Base Camp)"), f("Duration", "12–14 days"), f("Best season", "Mar–May · Sep–Nov"), f("Distance", "~130 km round trip"), f("Start", "Lukla, Nepal")}},
	"trek_inca": {img: "1464822759023-fed622ff2c3b",
		desc: "The classic 4-day route to Machu Picchu over Andean passes and Inca ruins. Permits are limited and sell out months ahead.",
		facts: []domain.Fact{f("Difficulty", "Moderate–Hard"), f("Max altitude", "4,215 m (Dead Woman's Pass)"), f("Duration", "4 days"), f("Best season", "May–Sep"), f("Distance", "~43 km"), f("Start", "Cusco, Peru · permit required")}},
	"trek_tmb": {img: "1464822759023-fed622ff2c3b",
		desc: "A circuit around the Mont Blanc massif through France, Italy and Switzerland — alpine meadows, glaciers and mountain huts.",
		facts: []domain.Fact{f("Difficulty", "Moderate"), f("Max altitude", "~2,665 m (Col des Fours)"), f("Duration", "7–11 days"), f("Best season", "Jun–Sep"), f("Distance", "~170 km"), f("Start", "Les Houches, France")}},
	"trek_kili": {img: "1500530855697-b586d89ba3ee",
		desc: "Africa's highest free-standing mountain. The summit, Uhuru Peak, crosses five climate zones from rainforest to arctic.",
		facts: []domain.Fact{f("Difficulty", "Challenging"), f("Max altitude", "5,895 m (Uhuru Peak)"), f("Duration", "6–8 days"), f("Best season", "Jan–Mar · Jun–Oct"), f("Route", "Machame / Lemosho"), f("Start", "Moshi, Tanzania")}},
	"trek_torres": {img: "1518602164578-cd0074062767",
		desc: "Patagonia's iconic W-Trek beneath the granite towers of Torres del Paine — glaciers, lakes and the famous sunrise viewpoint.",
		facts: []domain.Fact{f("Difficulty", "Moderate"), f("Duration", "4–5 days"), f("Best season", "Nov–Mar"), f("Distance", "~80 km"), f("Start", "Torres del Paine NP, Chile")}},
	"trek_annapurna": {img: "1506905925346-21bda4d32df4",
		desc: "A grand Himalayan circuit over the Thorong La pass, through diverse landscapes and Gurung & Manangi villages.",
		facts: []domain.Fact{f("Difficulty", "Moderate–Hard"), f("Max altitude", "5,416 m (Thorong La)"), f("Duration", "12–16 days"), f("Best season", "Mar–May · Oct–Nov"), f("Distance", "~160–230 km"), f("Start", "Besisahar, Nepal")}},
	"trek_fuji": {img: "1469474968028-56623f02e42e",
		desc: "Japan's sacred 3,776 m volcano. The official climbing window is short; many hike overnight to catch sunrise from the summit.",
		facts: []domain.Fact{f("Difficulty", "Moderate"), f("Max altitude", "3,776 m"), f("Duration", "1–2 days"), f("Best season", "Early Jul–early Sep"), f("Route", "Yoshida Trail"), f("Start", "Subaru Line 5th Station")}},
	"trek_milford": {img: "1469474968028-56623f02e42e",
		desc: "New Zealand's most famous Great Walk through Fiordland — rainforest, the Mackinnon Pass and Sutherland Falls.",
		facts: []domain.Fact{f("Difficulty", "Moderate"), f("Max altitude", "1,154 m (Mackinnon Pass)"), f("Duration", "4 days"), f("Best season", "Oct–Apr"), f("Distance", "53.5 km"), f("Start", "Glade Wharf · booking required")}},

	// ---- India ----
	"trek_valley": {img: "1501554728187-ce583db33af7",
		desc: "A UNESCO World Heritage monsoon trek through a Himalayan valley carpeted with hundreds of wildflower species.",
		facts: []domain.Fact{f("Difficulty", "Easy–Moderate"), f("Max altitude", "3,858 m (4,329 m w/ Hemkund)"), f("Duration", "6 days"), f("Best season", "Jul–Sep (bloom)"), f("Region", "Uttarakhand"), f("Base", "Govindghat")}},
	"trek_kedarkantha": {img: "1483728642387-6c3bdd6c93e5",
		desc: "India's most popular winter summit trek — snow-laden pine forests, frozen meadows and a 360° Himalayan summit view.",
		facts: []domain.Fact{f("Difficulty", "Easy–Moderate"), f("Max altitude", "3,810 m (summit)"), f("Duration", "6 days"), f("Best season", "Dec–Apr (snow)"), f("Region", "Uttarakhand"), f("Base", "Sankri")}},
	"trek_roopkund": {img: "1454942901704-3c44c11b2ad1",
		desc: "A high-altitude trek to the mysterious 'skeleton lake', through the rolling Bugyal meadows of Ali and Bedni.",
		facts: []domain.Fact{f("Difficulty", "Difficult"), f("Max altitude", "5,029 m"), f("Duration", "8 days"), f("Best season", "May–Jun · Sep–Oct"), f("Region", "Uttarakhand"), f("Base", "Lohajung")}},
	"trek_hampta": {img: "1428908728789-d2de25dbd4e2",
		desc: "A dramatic crossover trek from the green Kullu valley to the stark desert of Lahaul over the Hampta Pass.",
		facts: []domain.Fact{f("Difficulty", "Moderate"), f("Max altitude", "4,270 m (pass)"), f("Duration", "5 days"), f("Best season", "Jun–Sep"), f("Region", "Himachal Pradesh"), f("Base", "Manali")}},
	"trek_chadar": {img: "1454496522488-7a8e488e8606",
		desc: "The surreal frozen-river trek on the icy 'Chadar' of the Zanskar — one of the world's most unique winter walks.",
		facts: []domain.Fact{f("Difficulty", "Difficult"), f("Max altitude", "3,390 m"), f("Duration", "9 days"), f("Best season", "Jan–Feb"), f("Region", "Ladakh"), f("Base", "Leh")}},
	"trek_sandakphu": {img: "1500530855697-b586d89ba3ee",
		desc: "Walk the Singalila ridge to the highest point in West Bengal, with views of four of the world's five highest peaks.",
		facts: []domain.Fact{f("Difficulty", "Moderate"), f("Max altitude", "3,636 m"), f("Duration", "6–7 days"), f("Best season", "Mar–May · Oct–Dec"), f("Region", "West Bengal"), f("Base", "Manebhanjan")}},
	"trek_goechala": {img: "1454942901704-3c44c11b2ad1",
		desc: "A demanding Sikkim trek to a high pass with a sunrise face-to-face view of Kanchenjunga, the world's third-highest peak.",
		facts: []domain.Fact{f("Difficulty", "Difficult"), f("Max altitude", "4,940 m"), f("Duration", "9 days"), f("Best season", "Mar–May · Sep–Nov"), f("Region", "Sikkim"), f("Base", "Yuksom")}},
	"trek_brahmatal": {img: "1486870591958-9b9d0d1dda99",
		desc: "A gentle winter trek to a high-altitude frozen lake with grand views of Mt Trishul and Nanda Ghunti.",
		facts: []domain.Fact{f("Difficulty", "Easy–Moderate"), f("Max altitude", "3,734 m"), f("Duration", "6 days"), f("Best season", "Dec–Mar (snow)"), f("Region", "Uttarakhand"), f("Base", "Lohajung")}},
	"trek_tarsar": {img: "1506905925346-21bda4d32df4",
		desc: "A trek between the twin alpine lakes of Tarsar and Marsar in Kashmir, through some of the greenest meadows in the Himalayas.",
		facts: []domain.Fact{f("Difficulty", "Moderate"), f("Max altitude", "~4,000 m"), f("Duration", "7 days"), f("Best season", "Jul–Sep"), f("Region", "Kashmir"), f("Base", "Aru")}},
	"trek_nagtibba": {img: "1418065460487-3e41a6c84dc5",
		desc: "An easy weekend trek to the 'Serpent's Peak' near Mussoorie — great for first-timers, with a snowy ridge in winter.",
		facts: []domain.Fact{f("Difficulty", "Easy"), f("Max altitude", "3,022 m"), f("Duration", "2 days"), f("Best season", "Year-round (snow Dec–Mar)"), f("Region", "Uttarakhand"), f("Base", "Pantwari")}},

	// ---- State / weekend ----
	"trek_triund": {img: "1469474968028-56623f02e42e",
		desc: "A short, rewarding ridge trek above McLeod Ganj with a front-row view of the Dhauladhar range. Popular as an overnight camp.",
		facts: []domain.Fact{f("Difficulty", "Easy"), f("Max altitude", "2,850 m"), f("Duration", "1–2 days"), f("Best season", "Mar–Jun · Sep–Nov"), f("Region", "Himachal Pradesh"), f("Base", "McLeod Ganj")}},
	"trek_kheerganga": {img: "1428908728789-d2de25dbd4e2",
		desc: "A lush trek up the Parvati valley to natural hot springs — pine forest, waterfalls and a soak with a view.",
		facts: []domain.Fact{f("Difficulty", "Easy–Moderate"), f("Max altitude", "3,050 m"), f("Duration", "2 days"), f("Best season", "Apr–Jun · Sep–Nov"), f("Region", "Himachal Pradesh"), f("Base", "Barshaini")}},
	"trek_kalsubai": {img: "1454942901704-3c44c11b2ad1",
		desc: "A steep climb to the highest peak in Maharashtra (with iron-ladder sections), best done at dawn or on a full moon.",
		facts: []domain.Fact{f("Difficulty", "Moderate"), f("Max altitude", "1,646 m"), f("Duration", "1 day"), f("Best season", "Jun–Feb"), f("Region", "Maharashtra"), f("Base", "Bari village")}},
	"trek_skandagiri": {img: "1500530855697-b586d89ba3ee",
		desc: "A popular night trek near Bangalore, summiting at dawn above a sea of clouds.",
		facts: []domain.Fact{f("Difficulty", "Moderate"), f("Max altitude", "1,350 m"), f("Duration", "Night trek"), f("Best season", "Oct–Mar"), f("Region", "Karnataka"), f("Base", "Chikkaballapur")}},
}

// worldTrekIDs marks treks shown in the "Around the world" row (for the right guide link).
var worldTrekIDs = map[string]bool{}

func init() {
	for _, t := range worldTreks {
		worldTrekIDs[t.ID] = true
	}
}

// trekInfoURL points to the authoritative guide that hosts the real route photos & plan.
func trekInfoURL(it domain.ExploreItem) string {
	q := url.QueryEscape(it.Title)
	if worldTrekIDs[it.ID] {
		return "https://www.alltrails.com/search?q=" + q
	}
	return "https://indiahikes.com/?s=" + q
}

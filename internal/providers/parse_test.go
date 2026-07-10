package providers

import "testing"

func TestParseWikiSummary(t *testing.T) {
	cases := []struct {
		name        string
		body        string
		wantOK      bool
		wantExtract string
		wantThumb   string
		wantSource  string
	}{
		{
			name: "standard page",
			body: `{
				"type": "standard",
				"title": "Everest Base Camp",
				"extract": "Everest Base Camp is a camp used by climbers.",
				"thumbnail": {"source": "https://upload.wikimedia.org/ebc.jpg"},
				"content_urls": {"desktop": {"page": "https://en.wikipedia.org/wiki/Everest_Base_Camp"}}
			}`,
			wantOK:      true,
			wantExtract: "Everest Base Camp is a camp used by climbers.",
			wantThumb:   "https://upload.wikimedia.org/ebc.jpg",
			wantSource:  "https://en.wikipedia.org/wiki/Everest_Base_Camp",
		},
		{
			name:   "disambiguation is rejected",
			body:   `{"type": "disambiguation", "extract": "May refer to several things."}`,
			wantOK: false,
		},
		{
			name:   "empty extract is rejected",
			body:   `{"type": "standard", "extract": "   "}`,
			wantOK: false,
		},
		{
			name:   "malformed json is rejected",
			body:   `{not json`,
			wantOK: false,
		},
		{
			name:        "extract only, no thumbnail",
			body:        `{"type":"standard","extract":"A trek route."}`,
			wantOK:      true,
			wantExtract: "A trek route.",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseWikiSummary([]byte(tc.body))
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if got.Extract != tc.wantExtract {
				t.Errorf("extract = %q, want %q", got.Extract, tc.wantExtract)
			}
			if got.ThumbnailURL != tc.wantThumb {
				t.Errorf("thumb = %q, want %q", got.ThumbnailURL, tc.wantThumb)
			}
			if got.SourceURL != tc.wantSource {
				t.Errorf("source = %q, want %q", got.SourceURL, tc.wantSource)
			}
		})
	}
}

func TestParseTrails(t *testing.T) {
	body := `{
		"elements": [
			{"type":"way","id":123,"center":{"lat":30.1,"lon":79.2},"tags":{"route":"hiking","name":"Valley Trail"}},
			{"type":"node","id":456,"lat":30.5,"lon":79.6,"tags":{"natural":"peak","name":"Kedarkantha"}},
			{"type":"way","id":789,"center":{"lat":31.0,"lon":78.0},"tags":{"route":"hiking"}}
		]
	}`
	got := parseTrails([]byte(body))
	if len(got) != 2 {
		t.Fatalf("got %d trails, want 2 (unnamed way skipped)", len(got))
	}
	way := got[0]
	if way.ID != "way/123" || way.Name != "Valley Trail" || way.Kind != "TREK" {
		t.Errorf("way parsed wrong: %+v", way)
	}
	if way.Lat != 30.1 || way.Lng != 79.2 {
		t.Errorf("way center coords wrong: %+v", way)
	}
	if way.Source != "openstreetmap" {
		t.Errorf("source = %q", way.Source)
	}
	peak := got[1]
	if peak.ID != "node/456" || peak.Kind != "PEAK" {
		t.Errorf("peak parsed wrong: %+v", peak)
	}
	if peak.Lat != 30.5 || peak.Lng != 79.6 {
		t.Errorf("peak node coords wrong: %+v", peak)
	}
}

func TestParseTrailsBadJSON(t *testing.T) {
	if got := parseTrails([]byte(`{bad`)); got != nil {
		t.Fatalf("expected nil on bad json, got %v", got)
	}
}

func TestParseCommonsImages(t *testing.T) {
	body := `{
		"query": {
			"pages": {
				"100": {"title":"File:A.jpg","imageinfo":[{"url":"https://commons/A.jpg"}]},
				"200": {"title":"File:B.jpg","imageinfo":[{"url":"https://commons/B.jpg"}]}
			}
		}
	}`
	got := parseCommonsImages([]byte(body), 5)
	if len(got) != 2 {
		t.Fatalf("got %d images, want 2", len(got))
	}
	// Map iteration order is unstable; just assert both URLs are present.
	seen := map[string]bool{}
	for _, u := range got {
		seen[u] = true
	}
	if !seen["https://commons/A.jpg"] || !seen["https://commons/B.jpg"] {
		t.Errorf("missing expected urls: %v", got)
	}
}

func TestParseCommonsImagesLimit(t *testing.T) {
	body := `{"query":{"pages":{
		"1":{"imageinfo":[{"url":"https://c/1.jpg"}]},
		"2":{"imageinfo":[{"url":"https://c/2.jpg"}]},
		"3":{"imageinfo":[{"url":"https://c/3.jpg"}]}
	}}}`
	if got := parseCommonsImages([]byte(body), 2); len(got) != 2 {
		t.Fatalf("limit not respected: got %d, want 2", len(got))
	}
}

func TestParseElevations(t *testing.T) {
	body := `{"results":[
		{"latitude":30.1,"longitude":79.2,"elevation":3200.0},
		{"latitude":30.5,"longitude":79.6,"elevation":3810.5}
	]}`
	got := parseElevations([]byte(body))
	if len(got) != 2 {
		t.Fatalf("got %d elevations, want 2", len(got))
	}
	if got[0] != 3200.0 || got[1] != 3810.5 {
		t.Errorf("elevations parsed wrong: %v", got)
	}
	if parseElevations([]byte(`{bad`)) != nil {
		t.Error("expected nil on bad json")
	}
}

package places

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const base = "https://maps.googleapis.com/maps/api"

// Client wraps the Google Places + Geocoding (legacy) APIs. The key stays here,
// server-side. When the key is empty, callers fall back to sample data.
type Client struct {
	key  string
	http *http.Client
}

func New(key string) *Client {
	return &Client{key: key, http: &http.Client{Timeout: 10 * time.Second}}
}

func (c *Client) Enabled() bool { return c.key != "" }

// ---- raw response shapes --------------------------------------------------

type rawPlace struct {
	PlaceID          string   `json:"place_id"`
	Name             string   `json:"name"`
	FormattedAddress string   `json:"formatted_address"`
	Vicinity         string   `json:"vicinity"`
	Rating           float64  `json:"rating"`
	UserRatingsTotal int      `json:"user_ratings_total"`
	Types            []string `json:"types"`
	Photos           []struct {
		PhotoReference string `json:"photo_reference"`
	} `json:"photos"`
	Geometry struct {
		Location struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"location"`
	} `json:"geometry"`
}

type searchResp struct {
	Results []rawPlace `json:"results"`
	Status  string     `json:"status"`
}

// Place is the trimmed result used to build explore items.
type Place struct {
	PlaceID      string
	Name         string
	Address      string
	Rating       float64
	RatingsTotal int
	Types        []string
	PhotoRef     string
	Lat, Lng     float64
}

func (p rawPlace) toPlace() Place {
	addr := p.FormattedAddress
	if addr == "" {
		addr = p.Vicinity
	}
	ref := ""
	if len(p.Photos) > 0 {
		ref = p.Photos[0].PhotoReference
	}
	return Place{
		PlaceID: p.PlaceID, Name: p.Name, Address: addr, Rating: p.Rating,
		RatingsTotal: p.UserRatingsTotal, Types: p.Types, PhotoRef: ref,
		Lat: p.Geometry.Location.Lat, Lng: p.Geometry.Location.Lng,
	}
}

func (c *Client) get(ctx context.Context, path string, q url.Values, out any) error {
	q.Set("key", c.key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path+"?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

// TextSearch powers free-text search and the location-based explore rails.
func (c *Client) TextSearch(ctx context.Context, query string, lat, lng float64) ([]Place, error) {
	q := url.Values{}
	q.Set("query", query)
	if lat != 0 || lng != 0 {
		q.Set("location", fmt.Sprintf("%f,%f", lat, lng))
		q.Set("radius", "30000")
	}
	var r searchResp
	if err := c.get(ctx, "/place/textsearch/json", q, &r); err != nil {
		return nil, err
	}
	return mapPlaces(r.Results), nil
}

func mapPlaces(raw []rawPlace) []Place {
	out := make([]Place, 0, len(raw))
	for _, p := range raw {
		out = append(out, p.toPlace())
	}
	return out
}

// ---- Details --------------------------------------------------------------

type Review struct {
	Author string  `json:"author"`
	Rating float64 `json:"rating"`
	Text   string  `json:"text"`
	When   string  `json:"when"`
}

type Detail struct {
	PlaceID      string
	Name         string
	Address      string
	Phone        string
	Website      string
	Rating       float64
	RatingsTotal int
	Hours        []string
	PhotoRefs    []string
	Reviews      []Review
	Lat, Lng     float64
	Types        []string
}

type detailResp struct {
	Result struct {
		PlaceID          string   `json:"place_id"`
		Name             string   `json:"name"`
		FormattedAddress string   `json:"formatted_address"`
		FormattedPhone   string   `json:"formatted_phone_number"`
		Website          string   `json:"website"`
		Rating           float64  `json:"rating"`
		UserRatingsTotal int      `json:"user_ratings_total"`
		Types            []string `json:"types"`
		OpeningHours     struct {
			WeekdayText []string `json:"weekday_text"`
		} `json:"opening_hours"`
		Photos []struct {
			PhotoReference string `json:"photo_reference"`
		} `json:"photos"`
		Reviews []struct {
			AuthorName string  `json:"author_name"`
			Rating     float64 `json:"rating"`
			Text       string  `json:"text"`
			Relative   string  `json:"relative_time_description"`
		} `json:"reviews"`
		Geometry struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
		} `json:"geometry"`
	} `json:"result"`
	Status string `json:"status"`
}

func (c *Client) Details(ctx context.Context, placeID string) (Detail, error) {
	q := url.Values{}
	q.Set("place_id", placeID)
	q.Set("fields", "place_id,name,formatted_address,formatted_phone_number,website,rating,user_ratings_total,types,opening_hours,photos,reviews,geometry")
	var r detailResp
	if err := c.get(ctx, "/place/details/json", q, &r); err != nil {
		return Detail{}, err
	}
	res := r.Result
	d := Detail{
		PlaceID: res.PlaceID, Name: res.Name, Address: res.FormattedAddress, Phone: res.FormattedPhone,
		Website: res.Website, Rating: res.Rating, RatingsTotal: res.UserRatingsTotal,
		Hours: res.OpeningHours.WeekdayText, Types: res.Types,
		Lat: res.Geometry.Location.Lat, Lng: res.Geometry.Location.Lng,
	}
	for _, p := range res.Photos {
		d.PhotoRefs = append(d.PhotoRefs, p.PhotoReference)
	}
	for _, rv := range res.Reviews {
		d.Reviews = append(d.Reviews, Review{Author: rv.AuthorName, Rating: rv.Rating, Text: rv.Text, When: rv.Relative})
	}
	return d, nil
}

// Photo streams a Place photo (the API key never leaves the server).
func (c *Client) Photo(ctx context.Context, ref string, maxWidth int, w io.Writer) (string, error) {
	q := url.Values{}
	q.Set("maxwidth", fmt.Sprintf("%d", maxWidth))
	q.Set("photo_reference", ref)
	q.Set("key", c.key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/place/photo?"+q.Encode(), nil)
	if err != nil {
		return "", err
	}
	resp, err := c.http.Do(req) // follows the redirect to the image bytes
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	ct := resp.Header.Get("Content-Type")
	if _, err := io.Copy(w, resp.Body); err != nil {
		return ct, err
	}
	return ct, nil
}

// ---- Reverse geocode (city label for the top bar) -------------------------

type geocodeResp struct {
	Results []struct {
		AddressComponents []struct {
			LongName  string   `json:"long_name"`
			ShortName string   `json:"short_name"`
			Types     []string `json:"types"`
		} `json:"address_components"`
	} `json:"results"`
	Status string `json:"status"`
}

func (c *Client) ReverseGeocode(ctx context.Context, lat, lng float64) (string, error) {
	q := url.Values{}
	q.Set("latlng", fmt.Sprintf("%f,%f", lat, lng))
	var r geocodeResp
	if err := c.get(ctx, "/geocode/json", q, &r); err != nil {
		return "", err
	}
	city, country := "", ""
	for _, res := range r.Results {
		for _, comp := range res.AddressComponents {
			for _, t := range comp.Types {
				if t == "locality" && city == "" {
					city = comp.LongName
				}
				if t == "country" && country == "" {
					country = comp.ShortName
				}
			}
		}
		if city != "" && country != "" {
			break
		}
	}
	if city == "" {
		return country, nil
	}
	if country != "" {
		return city + ", " + country, nil
	}
	return city, nil
}

// Package integration manages a user's connected third-party apps (Strava, Cult.fit, …).
// Connection state is persisted per user. "Connecting" records intent and hands the client
// the provider's real authorize/app URL to open.
//
// NOTE: completing a true OAuth handshake (token exchange + webhook sync) for Strava needs a
// registered API client id/secret and a callback endpoint — see PRODUCTION.md. With
// STRAVA_CLIENT_ID set, the authorize URL below is the real Strava OAuth consent screen.
package integration

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kumargaurav/summit-backend/internal/httpx"
)

type Provider struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	URL         string `json:"url"`
	Connected   bool   `json:"connected"`
}

// catalog is the set of apps a user can connect.
var catalog = []Provider{
	{ID: "strava", Name: "Strava", Description: "Sync runs, rides & segments", Category: "fitness", URL: stravaAuthURL()},
	{ID: "cultfit", Name: "Cult.fit", Description: "Book classes & track workouts", Category: "fitness", URL: "https://www.cult.fit/"},
	{ID: "nike_run", Name: "Nike Run Club", Description: "Guided runs & training plans", Category: "running", URL: "https://www.nike.com/nrc-app"},
	{ID: "alltrails", Name: "AllTrails", Description: "Trail maps & recorded hikes", Category: "trekking", URL: "https://www.alltrails.com/"},
	{ID: "komoot", Name: "Komoot", Description: "Plan & navigate routes", Category: "trekking", URL: "https://www.komoot.com/"},
	{ID: "strv", Name: "Spotify", Description: "Soundtrack your sessions", Category: "music", URL: "https://www.spotify.com/"},
}

// stravaAuthURL is the real OAuth consent URL when STRAVA_CLIENT_ID is configured,
// otherwise the public site (so the button still leads somewhere useful).
func stravaAuthURL() string {
	cid := strings.TrimSpace(os.Getenv("STRAVA_CLIENT_ID"))
	if cid == "" {
		return "https://www.strava.com/"
	}
	redirect := strings.TrimSpace(os.Getenv("STRAVA_REDIRECT_URI"))
	if redirect == "" {
		redirect = "https://ascent-api-rzft.onrender.com/integrations/strava/callback"
	}
	return "https://www.strava.com/oauth/authorize?client_id=" + url.QueryEscape(cid) +
		"&response_type=code&approval_prompt=auto&scope=read,activity:read" +
		"&redirect_uri=" + url.QueryEscape(redirect)
}

type Repo struct{ pool *pgxpool.Pool }

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

func (r *Repo) connectedSet(ctx context.Context, userID string) map[string]bool {
	out := map[string]bool{}
	if r.pool == nil {
		return out
	}
	rows, err := r.pool.Query(ctx, `SELECT provider FROM user_integrations WHERE user_id = $1`, userID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var p string
		if rows.Scan(&p) == nil {
			out[p] = true
		}
	}
	return out
}

func (r *Repo) list(ctx context.Context, userID string) []Provider {
	connected := r.connectedSet(ctx, userID)
	out := make([]Provider, len(catalog))
	for i, p := range catalog {
		p.Connected = connected[p.ID]
		out[i] = p
	}
	return out
}

func validProvider(id string) bool {
	for _, p := range catalog {
		if p.ID == id {
			return true
		}
	}
	return false
}

// ---- HTTP -----------------------------------------------------------------

type Handler struct{ repo *Repo }

func NewHandler(repo *Repo) *Handler { return &Handler{repo: repo} }

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{"integrations": h.repo.list(r.Context(), httpx.UserID(r))})
}

func (h *Handler) Connect(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !validProvider(id) {
		httpx.Error(w, http.StatusNotFound, "unknown provider")
		return
	}
	_, err := h.repo.pool.Exec(r.Context(),
		`INSERT INTO user_integrations (user_id, provider) VALUES ($1, $2)
		 ON CONFLICT (user_id, provider) DO UPDATE SET connected_at = now()`,
		httpx.UserID(r), id)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not connect")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"integrations": h.repo.list(r.Context(), httpx.UserID(r))})
}

func (h *Handler) Disconnect(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := h.repo.pool.Exec(r.Context(),
		`DELETE FROM user_integrations WHERE user_id = $1 AND provider = $2`, httpx.UserID(r), id); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not disconnect")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"integrations": h.repo.list(r.Context(), httpx.UserID(r))})
}

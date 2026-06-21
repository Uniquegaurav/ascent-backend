package catalog

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kumargaurav/summit-backend/internal/domain"
	"github.com/kumargaurav/summit-backend/internal/httpx"
)

// Repo reads the seeded catalogs (interests, cities, popular searches).
type Repo struct{ pool *pgxpool.Pool }

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

func (r *Repo) Interests(ctx context.Context) ([]domain.Interest, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, label, emoji, theme, image_url, vibe FROM interests ORDER BY label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Interest{}
	for rows.Next() {
		var i domain.Interest
		if err := rows.Scan(&i.ID, &i.Label, &i.Emoji, &i.Theme, &i.ImageURL, &i.Vibe); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *Repo) Cities(ctx context.Context) ([]domain.City, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, country_code, lat, lng, theme FROM cities ORDER BY position`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.City{}
	for rows.Next() {
		var c domain.City
		if err := rows.Scan(&c.ID, &c.Name, &c.CountryCode, &c.Lat, &c.Lng, &c.Theme); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repo) PopularSearches(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT query FROM popular_searches ORDER BY position`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var q string
		if err := rows.Scan(&q); err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

// ---- HTTP -----------------------------------------------------------------

type Handler struct{ repo *Repo }

func NewHandler(repo *Repo) *Handler { return &Handler{repo: repo} }

func (h *Handler) Interests(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.Interests(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load interests")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"interests": items})
}

func (h *Handler) Cities(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.Cities(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load cities")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"cities": items})
}

func (h *Handler) PopularSearches(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.PopularSearches(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load searches")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"queries": items})
}

func (h *Handler) Categories(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{"categories": domain.Categories})
}

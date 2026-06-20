package user

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kumargaurav/summit-backend/internal/domain"
	"github.com/kumargaurav/summit-backend/internal/httpx"
)

type Repo struct{ pool *pgxpool.Pool }

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

func (r *Repo) Get(ctx context.Context, userID string) (domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, phone, onboarded, avatar_hue FROM users WHERE id = $1`, userID).
		Scan(&u.ID, &u.Name, &u.Phone, &u.Onboarded, &u.AvatarHue)
	if err != nil {
		return domain.User{}, err
	}
	u.InterestIDs, err = r.interestIDs(ctx, userID)
	return u, err
}

func (r *Repo) interestIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT interest_id FROM user_interests WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *Repo) UpdateName(ctx context.Context, userID, name string) (domain.User, error) {
	if _, err := r.pool.Exec(ctx, `UPDATE users SET name = $2 WHERE id = $1`, userID, name); err != nil {
		return domain.User{}, err
	}
	return r.Get(ctx, userID)
}

func (r *Repo) CompleteOnboarding(ctx context.Context, userID, name string, interestIDs []string) (domain.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE users SET name = $2, onboarded = TRUE WHERE id = $1`, userID, name); err != nil {
		return domain.User{}, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return domain.User{}, err
	}
	for _, id := range interestIDs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO user_interests (user_id, interest_id) VALUES ($1, $2)
			 ON CONFLICT DO NOTHING`, userID, id); err != nil {
			return domain.User{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, err
	}
	return r.Get(ctx, userID)
}

// ---- HTTP -----------------------------------------------------------------

type Handler struct{ repo *Repo }

func NewHandler(repo *Repo) *Handler { return &Handler{repo: repo} }

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	u, err := h.repo.Get(r.Context(), httpx.UserID(r))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Error(w, http.StatusNotFound, "user not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "could not load user")
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}

type nameReq struct {
	Name string `json:"name"`
}

func (h *Handler) UpdateName(w http.ResponseWriter, r *http.Request) {
	var req nameReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		httpx.Error(w, http.StatusBadRequest, "name required")
		return
	}
	u, err := h.repo.UpdateName(r.Context(), httpx.UserID(r), name)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not update")
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}

type onboardingReq struct {
	Name        string   `json:"name"`
	InterestIDs []string `json:"interestIds"`
}

func (h *Handler) Onboarding(w http.ResponseWriter, r *http.Request) {
	var req onboardingReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "Climber"
	}
	if len(req.InterestIDs) == 0 {
		httpx.Error(w, http.StatusBadRequest, "pick at least one peak")
		return
	}
	u, err := h.repo.CompleteOnboarding(r.Context(), httpx.UserID(r), name, req.InterestIDs)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not save")
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}

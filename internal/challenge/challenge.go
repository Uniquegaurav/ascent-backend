// Package challenge serves the community challenges surface (design ref 3):
// a catalog of goals a user can join, each showing live progress. The catalog
// + demo base_progress are seeded in migration 0002; joins/progress live in
// user_challenges. Real progress is derived from the user's logged activity in
// the challenge's interest, added to the seeded base so demo accounts look alive.
package challenge

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kumargaurav/summit-backend/internal/domain"
	"github.com/kumargaurav/summit-backend/internal/httpx"
)

type Repo struct{ pool *pgxpool.Pool }

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// List returns the challenge catalog with this user's joined state + progress.
// Progress = seeded base_progress + logs the user has written under an ascent
// whose interest matches the challenge, capped at the target.
func (r *Repo) List(ctx context.Context, userID string) ([]domain.Challenge, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.title, c.description, c.emoji, c.interest_id, c.target, c.unit,
		       c.base_progress,
		       COALESCE(uc.joined, FALSE) AS joined,
		       COALESCE(uc.progress, 0)   AS user_progress,
		       (SELECT COUNT(*) FROM logs l
		          JOIN ascents a ON a.id = l.ascent_id
		         WHERE l.user_id = $1 AND a.interest_id = c.interest_id)::int AS logged
		FROM challenges c
		LEFT JOIN user_challenges uc ON uc.challenge_id = c.id AND uc.user_id = $1
		ORDER BY c.title`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Challenge{}
	for rows.Next() {
		var c domain.Challenge
		var base, userProgress, logged int
		if err := rows.Scan(&c.ID, &c.Title, &c.Description, &c.Emoji, &c.InterestID,
			&c.Target, &c.Unit, &base, &c.Joined, &userProgress, &logged); err != nil {
			return nil, err
		}
		progress := base + userProgress + logged
		if progress > c.Target {
			progress = c.Target
		}
		c.Progress = progress
		out = append(out, c)
	}
	return out, rows.Err()
}

// SetJoined joins/leaves a challenge (idempotent upsert).
func (r *Repo) SetJoined(ctx context.Context, userID, challengeID string, joined bool) error {
	var exists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM challenges WHERE id = $1)`, challengeID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return errNotFound
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_challenges (user_id, challenge_id, joined) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, challenge_id) DO UPDATE SET joined = EXCLUDED.joined`,
		userID, challengeID, joined)
	return err
}

var errNotFound = errors.New("challenge not found")

type Handler struct{ repo *Repo }

func NewHandler(repo *Repo) *Handler { return &Handler{repo: repo} }

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.List(r.Context(), httpx.UserID(r))
	if err != nil {
		httpx.Internal(w, r, err, "could not load challenges")
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (h *Handler) Join(w http.ResponseWriter, r *http.Request)  { h.setJoined(w, r, true) }
func (h *Handler) Leave(w http.ResponseWriter, r *http.Request) { h.setJoined(w, r, false) }

func (h *Handler) setJoined(w http.ResponseWriter, r *http.Request, joined bool) {
	err := h.repo.SetJoined(r.Context(), httpx.UserID(r), chi.URLParam(r, "id"), joined)
	if err != nil {
		if errors.Is(err, errNotFound) {
			httpx.Error(w, http.StatusNotFound, "challenge not found")
			return
		}
		httpx.Internal(w, r, err, "could not update challenge")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

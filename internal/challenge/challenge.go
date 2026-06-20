package challenge

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kumargaurav/summit-backend/internal/domain"
	"github.com/kumargaurav/summit-backend/internal/httpx"
)

type Repo struct{ pool *pgxpool.Pool }

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

func (r *Repo) List(ctx context.Context, userID string) ([]domain.Challenge, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.title, c.description, c.emoji, c.interest_id, c.target, c.unit,
		       COALESCE(uc.progress, c.base_progress) AS progress,
		       COALESCE(uc.joined, FALSE) AS joined
		FROM challenges c
		LEFT JOIN user_challenges uc ON uc.challenge_id = c.id AND uc.user_id = $1
		ORDER BY c.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Challenge{}
	for rows.Next() {
		var c domain.Challenge
		if err := rows.Scan(&c.ID, &c.Title, &c.Description, &c.Emoji, &c.InterestID, &c.Target, &c.Unit, &c.Progress, &c.Joined); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repo) Join(ctx context.Context, userID, challengeID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_challenges (user_id, challenge_id, joined, progress)
		VALUES ($1, $2, TRUE, COALESCE((SELECT base_progress FROM challenges WHERE id = $2), 0))
		ON CONFLICT (user_id, challenge_id) DO UPDATE SET joined = TRUE`, userID, challengeID)
	return err
}

func (r *Repo) Complete(ctx context.Context, userID, challengeID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_challenges (user_id, challenge_id, joined, progress)
		VALUES ($1, $2, TRUE, COALESCE((SELECT target FROM challenges WHERE id = $2), 0))
		ON CONFLICT (user_id, challenge_id) DO UPDATE
		SET progress = COALESCE((SELECT target FROM challenges WHERE id = $2), 0), joined = TRUE`, userID, challengeID)
	return err
}

// ---- HTTP -----------------------------------------------------------------

type Handler struct{ repo *Repo }

func NewHandler(repo *Repo) *Handler { return &Handler{repo: repo} }

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.List(r.Context(), httpx.UserID(r))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load challenges")
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (h *Handler) Join(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.Join(r.Context(), httpx.UserID(r), chi.URLParam(r, "id")); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not join")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.Complete(r.Context(), httpx.UserID(r), chi.URLParam(r, "id")); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not complete")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

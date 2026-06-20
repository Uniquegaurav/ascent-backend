package friend

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

func (r *Repo) List(ctx context.Context, userID string) ([]domain.Friend, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.name, u.avatar_hue, u.last_ascent,
		       COALESCE(f.status, '') AS my_status,
		       (SELECT COUNT(*) FROM user_interests ui WHERE ui.user_id = u.id)::int AS peaks
		FROM users u
		LEFT JOIN friendships f ON f.user_id = $1 AND f.other_id = u.id
		WHERE u.is_demo = TRUE AND u.id <> $1
		ORDER BY u.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Friend{}
	for rows.Next() {
		var fr domain.Friend
		var status string
		if err := rows.Scan(&fr.ID, &fr.Name, &fr.AvatarHue, &fr.LastAscent, &status, &fr.PeaksExplored); err != nil {
			return nil, err
		}
		if status == "" {
			status = "SUGGESTED"
		}
		fr.Status = status
		out = append(out, fr)
	}
	return out, rows.Err()
}

func (r *Repo) MatchContacts(ctx context.Context, userID string, hashes []string) ([]domain.Friend, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.name, u.avatar_hue, u.last_ascent,
		       COALESCE(f.status, '') AS my_status,
		       (SELECT COUNT(*) FROM user_interests ui WHERE ui.user_id = u.id)::int AS peaks
		FROM users u
		LEFT JOIN friendships f ON f.user_id = $1 AND f.other_id = u.id
		WHERE u.is_demo = TRUE AND u.id <> $1 AND u.phone_hash = ANY($2)
		ORDER BY u.name`, userID, hashes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Friend{}
	for rows.Next() {
		var fr domain.Friend
		var status string
		if err := rows.Scan(&fr.ID, &fr.Name, &fr.AvatarHue, &fr.LastAscent, &status, &fr.PeaksExplored); err != nil {
			return nil, err
		}
		if status == "" {
			status = "SUGGESTED"
		}
		fr.Status = status
		out = append(out, fr)
	}
	return out, rows.Err()
}

func (r *Repo) SetStatus(ctx context.Context, userID, otherID, status string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO friendships (user_id, other_id, status) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, other_id) DO UPDATE SET status = EXCLUDED.status`,
		userID, otherID, status)
	return err
}

// ---- HTTP -----------------------------------------------------------------

type Handler struct{ repo *Repo }

func NewHandler(repo *Repo) *Handler { return &Handler{repo: repo} }

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.List(r.Context(), httpx.UserID(r))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load friends")
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

type contact struct {
	Name      string `json:"name"`
	PhoneHash string `json:"phoneHash"`
}

type syncReq struct {
	Contacts []contact `json:"contacts"`
}

func (h *Handler) SyncContacts(w http.ResponseWriter, r *http.Request) {
	var req syncReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	hashes := make([]string, 0, len(req.Contacts))
	for _, c := range req.Contacts {
		if c.PhoneHash != "" {
			hashes = append(hashes, c.PhoneHash)
		}
	}
	items, err := h.repo.MatchContacts(r.Context(), httpx.UserID(r), hashes)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not sync contacts")
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (h *Handler) Request(w http.ResponseWriter, r *http.Request) {
	h.mutate(w, r, "REQUESTED")
}

func (h *Handler) Accept(w http.ResponseWriter, r *http.Request) {
	h.mutate(w, r, "FOLLOWING")
}

func (h *Handler) mutate(w http.ResponseWriter, r *http.Request, status string) {
	otherID := chi.URLParam(r, "id")
	if otherID == "" {
		httpx.Error(w, http.StatusBadRequest, "missing id")
		return
	}
	if err := h.repo.SetStatus(r.Context(), httpx.UserID(r), otherID, status); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not update")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

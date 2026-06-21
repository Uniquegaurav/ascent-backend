package user

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
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
	if err != nil {
		return domain.User{}, err
	}
	var hasAvatar bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM avatars WHERE user_id = $1)`, userID).Scan(&hasAvatar); err == nil && hasAvatar {
		u.AvatarURL = "/avatars/" + userID
	}
	return u, nil
}

func (r *Repo) SaveAvatar(ctx context.Context, userID string, data []byte, contentType string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO avatars (user_id, data, content_type, updated_at) VALUES ($1, $2, $3, now())
		ON CONFLICT (user_id) DO UPDATE SET data = EXCLUDED.data, content_type = EXCLUDED.content_type, updated_at = now()`,
		userID, data, contentType)
	return err
}

func (r *Repo) LoadAvatar(ctx context.Context, userID string) (data []byte, contentType string, err error) {
	err = r.pool.QueryRow(ctx, `SELECT data, content_type FROM avatars WHERE user_id = $1`, userID).Scan(&data, &contentType)
	return data, contentType, err
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

// GetHobbies returns the user's customised home hobbies. Falls back to their
// onboarding interests, then to the first few catalog interests, so the home is
// never empty for a new user.
func (r *Repo) GetHobbies(ctx context.Context, userID string) (domain.HobbyPrefs, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT interest_id, theme FROM user_hobbies WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return domain.HobbyPrefs{}, err
	}
	defer rows.Close()
	ids := []string{}
	overrides := map[string]string{}
	for rows.Next() {
		var id string
		var theme *string
		if err := rows.Scan(&id, &theme); err != nil {
			return domain.HobbyPrefs{}, err
		}
		ids = append(ids, id)
		if theme != nil && *theme != "" {
			overrides[id] = *theme
		}
	}
	if err := rows.Err(); err != nil {
		return domain.HobbyPrefs{}, err
	}
	if len(ids) == 0 {
		ids, err = r.defaultHobbyIDs(ctx, userID)
		if err != nil {
			return domain.HobbyPrefs{}, err
		}
	}
	return domain.HobbyPrefs{HobbyIDs: ids, ThemeOverrides: overrides}, nil
}

func (r *Repo) defaultHobbyIDs(ctx context.Context, userID string) ([]string, error) {
	ids, err := r.interestIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(ids) > 0 {
		return ids, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT id FROM interests ORDER BY label LIMIT 5`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// SetHobbies replaces the user's home hobbies (ordered) and theme overrides.
func (r *Repo) SetHobbies(ctx context.Context, userID string, prefs domain.HobbyPrefs) (domain.HobbyPrefs, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.HobbyPrefs{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM user_hobbies WHERE user_id = $1`, userID); err != nil {
		return domain.HobbyPrefs{}, err
	}
	for pos, id := range prefs.HobbyIDs {
		var theme *string
		if t, ok := prefs.ThemeOverrides[id]; ok && t != "" {
			tv := t
			theme = &tv
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO user_hobbies (user_id, interest_id, position, theme) VALUES ($1, $2, $3, $4)
			 ON CONFLICT (user_id, interest_id) DO UPDATE SET position = EXCLUDED.position, theme = EXCLUDED.theme`,
			userID, id, pos, theme); err != nil {
			return domain.HobbyPrefs{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.HobbyPrefs{}, err
	}
	return r.GetHobbies(ctx, userID)
}

// ---- HTTP -----------------------------------------------------------------

type Handler struct{ repo *Repo }

func NewHandler(repo *Repo) *Handler { return &Handler{repo: repo} }

func (h *Handler) Hobbies(w http.ResponseWriter, r *http.Request) {
	prefs, err := h.repo.GetHobbies(r.Context(), httpx.UserID(r))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load hobbies")
		return
	}
	httpx.JSON(w, http.StatusOK, prefs)
}

func (h *Handler) SetHobbies(w http.ResponseWriter, r *http.Request) {
	var req domain.HobbyPrefs
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	if req.ThemeOverrides == nil {
		req.ThemeOverrides = map[string]string{}
	}
	prefs, err := h.repo.SetHobbies(r.Context(), httpx.UserID(r), req)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not save hobbies")
		return
	}
	httpx.JSON(w, http.StatusOK, prefs)
}

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

type avatarReq struct {
	Data        string `json:"data"`        // base64-encoded image bytes
	ContentType string `json:"contentType"` // e.g. "image/jpeg"
}

// SetAvatar accepts a base64 image, stores it, and returns the updated user.
func (h *Handler) SetAvatar(w http.ResponseWriter, r *http.Request) {
	var req avatarReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(req.Data))
	if err != nil || len(raw) == 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid image data")
		return
	}
	if len(raw) > 5*1024*1024 {
		httpx.Error(w, http.StatusRequestEntityTooLarge, "image too large (max 5MB)")
		return
	}
	ct := req.ContentType
	if ct == "" {
		ct = "image/jpeg"
	}
	uid := httpx.UserID(r)
	if err := h.repo.SaveAvatar(r.Context(), uid, raw, ct); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not save avatar")
		return
	}
	u, err := h.repo.Get(r.Context(), uid)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load user")
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}

// ServeAvatar streams a user's stored avatar (public — image tags can't send a JWT).
func (h *Handler) ServeAvatar(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	data, ct, err := h.repo.LoadAvatar(r.Context(), id)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "no avatar")
		return
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

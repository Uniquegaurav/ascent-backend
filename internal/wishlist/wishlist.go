package wishlist

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kumargaurav/summit-backend/internal/domain"
	"github.com/kumargaurav/summit-backend/internal/httpx"
)

type Repo struct{ pool *pgxpool.Pool }

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const cols = `id, item, planned_date, booking_url, added_to_calendar, invited_friend_ids, created_at`

func (r *Repo) List(ctx context.Context, userID string) ([]domain.Wishlist, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+cols+` FROM wishlists WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Wishlist{}
	for rows.Next() {
		w, err := scanWishlist(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// Add upserts the place by (user, explore item) so saving twice is idempotent and never
// clobbers an existing plan.
func (r *Repo) Add(ctx context.Context, userID string, item domain.ExploreItem) (domain.Wishlist, error) {
	itemJSON, err := json.Marshal(item.WithSummitCategory())
	if err != nil {
		return domain.Wishlist{}, err
	}
	row := r.pool.QueryRow(ctx, `
		INSERT INTO wishlists (user_id, explore_item_id, item)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, explore_item_id) DO UPDATE SET item = EXCLUDED.item
		RETURNING `+cols, userID, item.ID, itemJSON)
	return scanWishlist(row)
}

func (r *Repo) UpdatePlan(ctx context.Context, userID, id string, plannedDateMs *int64, bookingURL string, addedToCalendar bool, invited []string) (domain.Wishlist, error) {
	if invited == nil {
		invited = []string{}
	}
	var planned *time.Time
	if plannedDateMs != nil {
		t := time.UnixMilli(*plannedDateMs).UTC()
		planned = &t
	}
	row := r.pool.QueryRow(ctx, `
		UPDATE wishlists
		SET planned_date = $3, booking_url = $4, added_to_calendar = $5, invited_friend_ids = $6
		WHERE id = $1 AND user_id = $2
		RETURNING `+cols, id, userID, planned, bookingURL, addedToCalendar, invited)
	return scanWishlist(row)
}

func (r *Repo) Remove(ctx context.Context, userID, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM wishlists WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

type scannable interface{ Scan(dest ...any) error }

func scanWishlist(row scannable) (domain.Wishlist, error) {
	var (
		w         domain.Wishlist
		itemJSON  []byte
		planned   *time.Time
		invited   []string
		createdAt time.Time
	)
	if err := row.Scan(&w.ID, &itemJSON, &planned, &w.BookingURL, &w.AddedToCalendar, &invited, &createdAt); err != nil {
		return domain.Wishlist{}, err
	}
	if len(itemJSON) > 0 {
		_ = json.Unmarshal(itemJSON, &w.Item)
	}
	if planned != nil {
		ms := planned.UnixMilli()
		w.PlannedDateMs = &ms
	}
	w.InvitedFriendIDs = invited
	if w.InvitedFriendIDs == nil {
		w.InvitedFriendIDs = []string{}
	}
	w.CreatedAtMs = createdAt.UnixMilli()
	return w, nil
}

// ---- HTTP -----------------------------------------------------------------

type Handler struct{ repo *Repo }

func NewHandler(repo *Repo) *Handler { return &Handler{repo: repo} }

type listResp struct {
	Items []domain.Wishlist `json:"items"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.List(r.Context(), httpx.UserID(r))
	if err != nil {
		httpx.Internal(w, r, err, "could not load wishlist")
		return
	}
	httpx.JSON(w, http.StatusOK, listResp{Items: items})
}

type addReq struct {
	Item domain.ExploreItem `json:"item"`
}

func (h *Handler) Add(w http.ResponseWriter, r *http.Request) {
	var req addReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	if req.Item.ID == "" {
		httpx.Error(w, http.StatusBadRequest, "item id required")
		return
	}
	saved, err := h.repo.Add(r.Context(), httpx.UserID(r), req.Item)
	if err != nil {
		httpx.Internal(w, r, err, "could not save to wishlist")
		return
	}
	httpx.JSON(w, http.StatusCreated, saved)
}

type planReq struct {
	PlannedDateMs    *int64   `json:"plannedDateMs"`
	BookingURL       string   `json:"bookingUrl"`
	AddedToCalendar  bool     `json:"addedToCalendar"`
	InvitedFriendIDs []string `json:"invitedFriendIds"`
}

func (h *Handler) UpdatePlan(w http.ResponseWriter, r *http.Request) {
	var req planReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	updated, err := h.repo.UpdatePlan(r.Context(), httpx.UserID(r), chi.URLParam(r, "id"),
		req.PlannedDateMs, req.BookingURL, req.AddedToCalendar, req.InvitedFriendIDs)
	if err != nil {
		if err == pgx.ErrNoRows {
			httpx.Error(w, http.StatusNotFound, "wishlist item not found")
			return
		}
		httpx.Internal(w, r, err, "could not update plan")
		return
	}
	httpx.JSON(w, http.StatusOK, updated)
}

func (h *Handler) Remove(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.Remove(r.Context(), httpx.UserID(r), chi.URLParam(r, "id")); err != nil {
		httpx.Internal(w, r, err, "could not remove")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

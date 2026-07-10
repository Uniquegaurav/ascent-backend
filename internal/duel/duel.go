// Package duel serves head-to-head competitions (design ref 2): two friends
// race on logged activity over a fixed window. Both sides' scores are counted
// live from the logs table, so there is no counter to keep in sync.
package duel

import (
	"context"
	"errors"
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

var (
	errNotFriends = errors.New("can only duel a friend")
	errNotFound   = errors.New("duel not found")
)

// Create opens a duel from challenger against opponent. Opponent must be a
// friend (accepted, either direction) or a demo climber. Default window 7 days.
func (r *Repo) Create(ctx context.Context, challenger, opponent, interestID string, days int) (domain.Duel, error) {
	if days <= 0 {
		days = 7
	}
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM users WHERE id = $2 AND is_demo)
		    OR EXISTS(
		       SELECT 1 FROM friendships
		       WHERE status = 'FOLLOWING'
		         AND ((user_id = $1 AND other_id = $2) OR (user_id = $2 AND other_id = $1)))`,
		challenger, opponent).Scan(&ok)
	if err != nil {
		return domain.Duel{}, err
	}
	if !ok {
		return domain.Duel{}, errNotFriends
	}
	var id string
	var interest *string
	if interestID != "" {
		interest = &interestID
	}
	// Demo opponents auto-accept so the duel is immediately live.
	status := "PENDING"
	var demo bool
	_ = r.pool.QueryRow(ctx, `SELECT is_demo FROM users WHERE id = $1`, opponent).Scan(&demo)
	if demo {
		status = "ACTIVE"
	}
	err = r.pool.QueryRow(ctx, `
		INSERT INTO duels (challenger, opponent, interest_id, ends_at, status)
		VALUES ($1, $2, $3, now() + make_interval(days => $4), $5)
		RETURNING id`, challenger, opponent, interest, days, status).Scan(&id)
	if err != nil {
		return domain.Duel{}, err
	}
	list, err := r.List(ctx, challenger)
	if err != nil {
		return domain.Duel{}, err
	}
	for _, d := range list {
		if d.ID == id {
			return d, nil
		}
	}
	return domain.Duel{}, errNotFound
}

// List returns the user's duels (either side) with both live scores.
func (r *Repo) List(ctx context.Context, userID string) ([]domain.Duel, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT d.id, d.challenger, d.opponent, d.interest_id, d.status, d.ends_at,
		       cu.name, ou.name,
		       (SELECT COUNT(*) FROM logs l JOIN ascents a ON a.id = l.ascent_id
		         WHERE l.user_id = d.challenger AND l.created_at >= d.starts_at AND l.created_at <= d.ends_at
		           AND (d.interest_id IS NULL OR a.interest_id = d.interest_id))::int,
		       (SELECT COUNT(*) FROM logs l JOIN ascents a ON a.id = l.ascent_id
		         WHERE l.user_id = d.opponent AND l.created_at >= d.starts_at AND l.created_at <= d.ends_at
		           AND (d.interest_id IS NULL OR a.interest_id = d.interest_id))::int
		FROM duels d
		JOIN users cu ON cu.id = d.challenger
		JOIN users ou ON ou.id = d.opponent
		WHERE (d.challenger = $1 OR d.opponent = $1) AND d.status <> 'DECLINED'
		ORDER BY d.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Duel{}
	for rows.Next() {
		var d domain.Duel
		var challenger, opponent string
		var interest *string
		var endsAt time.Time
		if err := rows.Scan(&d.ID, &challenger, &opponent, &interest, &d.Status, &endsAt,
			&d.ChallengerName, &d.OpponentName, &d.ChallengerScore, &d.OpponentScore); err != nil {
			return nil, err
		}
		// Orient the duel from the caller's perspective: "you" vs "them".
		d.IsChallenger = challenger == userID
		if d.IsChallenger {
			d.YouScore, d.ThemScore, d.ThemName = d.ChallengerScore, d.OpponentScore, d.OpponentName
		} else {
			d.YouScore, d.ThemScore, d.ThemName = d.OpponentScore, d.ChallengerScore, d.ChallengerName
		}
		if interest != nil {
			d.InterestID = *interest
		}
		d.EndsAtMs = endsAt.UnixMilli()
		out = append(out, d)
	}
	return out, rows.Err()
}

// SetStatus accepts or declines a duel; only the opponent may respond.
func (r *Repo) SetStatus(ctx context.Context, userID, duelID, status string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE duels SET status = $3 WHERE id = $1 AND opponent = $2 AND status = 'PENDING'`,
		duelID, userID, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errNotFound
	}
	return nil
}

// ---- HTTP -----------------------------------------------------------------

type Handler struct{ repo *Repo }

func NewHandler(repo *Repo) *Handler { return &Handler{repo: repo} }

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.List(r.Context(), httpx.UserID(r))
	if err != nil {
		httpx.Internal(w, r, err, "could not load duels")
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

type createReq struct {
	OpponentID string `json:"opponentId"`
	InterestID string `json:"interestId"`
	Days       int    `json:"days"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createReq
	if err := httpx.Decode(r, &req); err != nil || req.OpponentID == "" {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	if req.OpponentID == httpx.UserID(r) {
		httpx.Error(w, http.StatusBadRequest, "cannot duel yourself")
		return
	}
	d, err := h.repo.Create(r.Context(), httpx.UserID(r), req.OpponentID, req.InterestID, req.Days)
	if err != nil {
		if errors.Is(err, errNotFriends) {
			httpx.Error(w, http.StatusForbidden, err.Error())
			return
		}
		httpx.Internal(w, r, err, "could not create duel")
		return
	}
	httpx.JSON(w, http.StatusCreated, d)
}

func (h *Handler) Accept(w http.ResponseWriter, r *http.Request)  { h.respond(w, r, "ACTIVE") }
func (h *Handler) Decline(w http.ResponseWriter, r *http.Request) { h.respond(w, r, "DECLINED") }

func (h *Handler) respond(w http.ResponseWriter, r *http.Request, status string) {
	err := h.repo.SetStatus(r.Context(), httpx.UserID(r), chi.URLParam(r, "id"), status)
	if err != nil {
		if errors.Is(err, errNotFound) || errors.Is(err, pgx.ErrNoRows) {
			httpx.Error(w, http.StatusNotFound, "duel not found")
			return
		}
		httpx.Internal(w, r, err, "could not update duel")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

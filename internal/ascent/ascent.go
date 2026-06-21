package ascent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kumargaurav/summit-backend/internal/domain"
	"github.com/kumargaurav/summit-backend/internal/explore"
	"github.com/kumargaurav/summit-backend/internal/httpx"
)

type Repo struct{ pool *pgxpool.Pool }

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

func (r *Repo) List(ctx context.Context, userID string) ([]domain.Ascent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.title, a.category, a.theme, a.image_url, a.kind, a.location_name, a.status,
		       (SELECT COUNT(*) FROM logs l WHERE l.ascent_id = a.id)::int AS log_count, a.created_at
		FROM ascents a WHERE a.user_id = $1 ORDER BY a.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAscents(rows)
}

func (r *Repo) Get(ctx context.Context, userID, id string) (domain.Ascent, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT a.id, a.title, a.category, a.theme, a.image_url, a.kind, a.location_name, a.status,
		       (SELECT COUNT(*) FROM logs l WHERE l.ascent_id = a.id)::int AS log_count, a.created_at
		FROM ascents a WHERE a.id = $1 AND a.user_id = $2`, id, userID)
	return scanAscent(row)
}

// FindBySource returns an existing ascent the user already added from this explore item.
func (r *Repo) FindBySource(ctx context.Context, userID, sourceID string) (domain.Ascent, bool, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT a.id, a.title, a.category, a.theme, a.image_url, a.kind, a.location_name, a.status,
		       (SELECT COUNT(*) FROM logs l WHERE l.ascent_id = a.id)::int AS log_count, a.created_at
		FROM ascents a WHERE a.user_id = $1 AND a.source_item_id = $2 LIMIT 1`, userID, sourceID)
	a, err := scanAscentRow(row)
	if err == pgx.ErrNoRows {
		return domain.Ascent{}, false, nil
	}
	if err != nil {
		return domain.Ascent{}, false, err
	}
	return a, true, nil
}

// AllLogs returns every log the user has written, across all ascents (central feed).
func (r *Repo) AllLogs(ctx context.Context, userID string) ([]domain.Log, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, ascent_id, title, note, mood_score, location_name, lat, lng, image_urls, metrics, created_at
		FROM logs WHERE user_id = $1 ORDER BY created_at DESC LIMIT 100`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLogs(rows)
}

func (r *Repo) Create(ctx context.Context, userID string, a domain.Ascent, sourceItemID string) (domain.Ascent, error) {
	var id string
	var createdAt time.Time
	err := r.pool.QueryRow(ctx, `
		INSERT INTO ascents (user_id, title, category, theme, image_url, kind, location_name, source_item_id, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'ACTIVE')
		RETURNING id, created_at`,
		userID, a.Title, a.Category, a.Theme, a.ImageURL, a.Kind, nullStr(a.LocationName), nullStr(sourceItemID)).
		Scan(&id, &createdAt)
	if err != nil {
		return domain.Ascent{}, err
	}
	a.ID = id
	a.Status = "ACTIVE"
	a.CreatedAtMs = createdAt.UnixMilli()
	return a.WithSummitCategory(), nil
}

func (r *Repo) AddLog(ctx context.Context, userID, ascentID string, l domain.Log) (domain.Log, error) {
	var owns bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ascents WHERE id = $1 AND user_id = $2)`, ascentID, userID).Scan(&owns); err != nil {
		return domain.Log{}, err
	}
	if !owns {
		return domain.Log{}, errNotFound
	}
	var locName *string
	var lat, lng *float64
	if l.Location != nil {
		locName = &l.Location.Name
		lat = &l.Location.Lat
		lng = &l.Location.Lng
	}
	images := l.ImageURLs
	if images == nil {
		images = []string{}
	}
	if l.Metrics == nil {
		l.Metrics = map[string]string{}
	}
	metricsJSON, err := json.Marshal(l.Metrics)
	if err != nil {
		return domain.Log{}, err
	}
	var id string
	var createdAt time.Time
	err = r.pool.QueryRow(ctx, `
		INSERT INTO logs (ascent_id, user_id, title, note, mood_score, location_name, lat, lng, image_urls, metrics)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id, created_at`,
		ascentID, userID, l.Title, l.Note, l.MoodScore, locName, lat, lng, images, metricsJSON).Scan(&id, &createdAt)
	if err != nil {
		return domain.Log{}, err
	}
	l.ID = id
	l.AscentID = ascentID
	l.ImageURLs = images
	l.DateEpochMs = createdAt.UnixMilli()
	return l, nil
}

func (r *Repo) Logs(ctx context.Context, ascentID string) ([]domain.Log, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, ascent_id, title, note, mood_score, location_name, lat, lng, image_urls, metrics, created_at
		FROM logs WHERE ascent_id = $1 ORDER BY created_at DESC`, ascentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Log{}
	for rows.Next() {
		var l domain.Log
		var images []string
		var locName *string
		var lat, lng *float64
		var metrics []byte
		var createdAt time.Time
		if err := rows.Scan(&l.ID, &l.AscentID, &l.Title, &l.Note, &l.MoodScore, &locName, &lat, &lng, &images, &metrics, &createdAt); err != nil {
			return nil, err
		}
		l.ImageURLs = images
		if l.ImageURLs == nil {
			l.ImageURLs = []string{}
		}
		l.Metrics = map[string]string{}
		if len(metrics) > 0 {
			_ = json.Unmarshal(metrics, &l.Metrics)
		}
		if locName != nil && lat != nil && lng != nil {
			l.Location = &domain.GeoLocation{Name: *locName, Lat: *lat, Lng: *lng}
		}
		l.DateEpochMs = createdAt.UnixMilli()
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *Repo) Summit(ctx context.Context, userID, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE ascents SET status = 'SUMMITED' WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func (r *Repo) Feed(ctx context.Context, userID string) ([]domain.FeedItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT l.id, u.name, u.avatar_hue, (l.user_id = $1) AS is_self,
		       l.title, l.note, COALESCE(a.category, ''), l.location_name, l.created_at
		FROM logs l
		JOIN users u ON u.id = l.user_id
		LEFT JOIN ascents a ON a.id = l.ascent_id
		WHERE l.user_id = $1 OR u.is_demo = TRUE
		ORDER BY l.created_at DESC LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.FeedItem{}
	for rows.Next() {
		var fi domain.FeedItem
		var locName *string
		var createdAt time.Time
		if err := rows.Scan(&fi.ID, &fi.AuthorName, &fi.AuthorHue, &fi.IsSelf, &fi.Title, &fi.Description, &fi.Category, &locName, &createdAt); err != nil {
			return nil, err
		}
		if fi.IsSelf {
			fi.AuthorName = "You"
		}
		fi.Location = locName
		fi.TimestampEpochMs = createdAt.UnixMilli()
		out = append(out, fi)
	}
	return out, rows.Err()
}

func (r *Repo) Invite(ctx context.Context, fromUser, toUser, ascentID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO ascent_invites (from_user, to_user, ascent_id) VALUES ($1, $2, $3)`,
		fromUser, toUser, ascentID)
	return err
}

func (r *Repo) FriendLogs(ctx context.Context, friendID string) ([]domain.Log, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, ascent_id, title, note, mood_score, location_name, lat, lng, image_urls, metrics, created_at
		FROM logs WHERE user_id = $1 ORDER BY created_at DESC LIMIT 20`, friendID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Log{}
	for rows.Next() {
		var l domain.Log
		var images []string
		var locName *string
		var lat, lng *float64
		var metrics []byte
		var createdAt time.Time
		if err := rows.Scan(&l.ID, &l.AscentID, &l.Title, &l.Note, &l.MoodScore, &locName, &lat, &lng, &images, &metrics, &createdAt); err != nil {
			return nil, err
		}
		l.ImageURLs = images
		if l.ImageURLs == nil {
			l.ImageURLs = []string{}
		}
		l.Metrics = map[string]string{}
		if len(metrics) > 0 {
			_ = json.Unmarshal(metrics, &l.Metrics)
		}
		l.DateEpochMs = createdAt.UnixMilli()
		out = append(out, l)
	}
	return out, rows.Err()
}

var errNotFound = errors.New("ascent not found")

func nullStr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

func scanAscents(rows pgx.Rows) ([]domain.Ascent, error) {
	out := []domain.Ascent{}
	for rows.Next() {
		a, err := scanAscentRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

type scannable interface {
	Scan(dest ...any) error
}

func scanAscent(row scannable) (domain.Ascent, error) { return scanAscentRow(row) }

func scanLogs(rows pgx.Rows) ([]domain.Log, error) {
	out := []domain.Log{}
	for rows.Next() {
		var l domain.Log
		var images []string
		var locName *string
		var lat, lng *float64
		var metrics []byte
		var createdAt time.Time
		if err := rows.Scan(&l.ID, &l.AscentID, &l.Title, &l.Note, &l.MoodScore, &locName, &lat, &lng, &images, &metrics, &createdAt); err != nil {
			return nil, err
		}
		l.ImageURLs = images
		if l.ImageURLs == nil {
			l.ImageURLs = []string{}
		}
		l.Metrics = map[string]string{}
		if len(metrics) > 0 {
			_ = json.Unmarshal(metrics, &l.Metrics)
		}
		if locName != nil && lat != nil && lng != nil {
			l.Location = &domain.GeoLocation{Name: *locName, Lat: *lat, Lng: *lng}
		}
		l.DateEpochMs = createdAt.UnixMilli()
		out = append(out, l)
	}
	return out, rows.Err()
}

func scanAscentRow(row scannable) (domain.Ascent, error) {
	var a domain.Ascent
	var locName *string
	var createdAt time.Time
	if err := row.Scan(&a.ID, &a.Title, &a.Category, &a.Theme, &a.ImageURL, &a.Kind, &locName, &a.Status, &a.LogCount, &createdAt); err != nil {
		return domain.Ascent{}, err
	}
	if locName != nil {
		a.LocationName = *locName
	}
	a.CreatedAtMs = createdAt.UnixMilli()
	return a.WithSummitCategory(), nil
}

// ---- HTTP -----------------------------------------------------------------

type Handler struct{ repo *Repo }

func NewHandler(repo *Repo) *Handler { return &Handler{repo: repo} }

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.List(r.Context(), httpx.UserID(r))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load ascents")
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

type createReq struct {
	FromExploreItemID string `json:"fromExploreItemId"`
	Title             string `json:"title"`
	Category          string `json:"category"`
	Theme             string `json:"theme"`
	Kind              string `json:"kind"`
	LocationName      string `json:"locationName"`
	ImageURL          string `json:"imageUrl"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	uid := httpx.UserID(r)
	source := strings.TrimSpace(req.FromExploreItemID)

	// Dedup: if this explore item is already on the user's range, return it.
	if source != "" {
		if existing, ok, err := h.repo.FindBySource(r.Context(), uid, source); err == nil && ok {
			httpx.JSON(w, http.StatusOK, existing)
			return
		}
	}

	var a domain.Ascent
	switch {
	case strings.TrimSpace(req.Title) != "":
		// App sends the item's fields directly (works for real Google places too).
		a = domain.Ascent{
			Title: strings.TrimSpace(req.Title), Category: req.Category, Theme: orDefault(req.Theme, "ALPINE"),
			ImageURL: req.ImageURL, Kind: orDefault(req.Kind, "HOBBY"), LocationName: req.LocationName,
		}
	case source != "":
		// Fallback: resolve a static/hobby item by id.
		item, ok := explore.Lookup(source)
		if !ok {
			httpx.Error(w, http.StatusBadRequest, "unknown explore item")
			return
		}
		a = domain.Ascent{Title: item.Title, Category: item.Category, Theme: item.Theme, ImageURL: item.ImageURL, Kind: item.Kind, LocationName: item.LocationName}
	default:
		httpx.Error(w, http.StatusBadRequest, "title required")
		return
	}

	created, err := h.repo.Create(r.Context(), uid, a, source)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not create ascent")
		return
	}
	httpx.JSON(w, http.StatusCreated, created)
}

func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	uid := httpx.UserID(r)
	id := chi.URLParam(r, "id")
	a, err := h.repo.Get(r.Context(), uid, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Error(w, http.StatusNotFound, "ascent not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "could not load ascent")
		return
	}
	logs, err := h.repo.Logs(r.Context(), id)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load logs")
		return
	}
	httpx.JSON(w, http.StatusOK, domain.AscentDetail{Ascent: a, Logs: logs})
}

type logReq struct {
	Title     string              `json:"title"`
	Note      string              `json:"note"`
	MoodScore int                 `json:"moodScore"`
	ImageURLs []string            `json:"imageUrls"`
	Location  *domain.GeoLocation `json:"location"`
	Metrics   map[string]string   `json:"metrics"`
}

func (h *Handler) AddLog(w http.ResponseWriter, r *http.Request) {
	var req logReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		httpx.Error(w, http.StatusBadRequest, "title required")
		return
	}
	mood := req.MoodScore
	if mood < 1 || mood > 5 {
		mood = 4
	}
	log, err := h.repo.AddLog(r.Context(), httpx.UserID(r), chi.URLParam(r, "id"), domain.Log{
		Title: strings.TrimSpace(req.Title), Note: strings.TrimSpace(req.Note), MoodScore: mood, ImageURLs: req.ImageURLs, Location: req.Location, Metrics: req.Metrics,
	})
	if err != nil {
		if errors.Is(err, errNotFound) {
			httpx.Error(w, http.StatusNotFound, "ascent not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "could not add log")
		return
	}
	httpx.JSON(w, http.StatusCreated, log)
}

func (h *Handler) Summit(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.Summit(r.Context(), httpx.UserID(r), chi.URLParam(r, "id")); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not summit")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) Feed(w http.ResponseWriter, r *http.Request) {
	feed, err := h.repo.Feed(r.Context(), httpx.UserID(r))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load feed")
		return
	}
	httpx.JSON(w, http.StatusOK, feed)
}

// AllLogs is the central log feed for the My Ascents screen (all logs, any ascent).
func (h *Handler) AllLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := h.repo.AllLogs(r.Context(), httpx.UserID(r))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load logs")
		return
	}
	httpx.JSON(w, http.StatusOK, logs)
}

type inviteReq struct {
	AscentID string `json:"ascentId"`
}

func (h *Handler) Invite(w http.ResponseWriter, r *http.Request) {
	var req inviteReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	if err := h.repo.Invite(r.Context(), httpx.UserID(r), chi.URLParam(r, "id"), req.AscentID); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not invite")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) FriendLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := h.repo.FriendLogs(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load logs")
		return
	}
	httpx.JSON(w, http.StatusOK, logs)
}

func orDefault(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

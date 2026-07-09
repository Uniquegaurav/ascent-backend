package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kumargaurav/summit-backend/internal/domain"
	"github.com/kumargaurav/summit-backend/internal/httpx"
)

const otpTTL = 5 * time.Minute

// ---- SMS ------------------------------------------------------------------

// SMSSender delivers a one-time code. LogSender is the dev implementation;
// MSG91Sender (sms.go) is the production one, selected via SMS_PROVIDER.
type SMSSender interface {
	Send(ctx context.Context, phone, code string) error
}

type LogSender struct{}

func (LogSender) Send(_ context.Context, phone, code string) error {
	slog.Info("OTP (dev)", "phone", phone, "code", code)
	return nil
}

// ---- JWT ------------------------------------------------------------------

type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenManager(secret string, ttl time.Duration) TokenManager {
	return TokenManager{secret: []byte(secret), ttl: ttl}
}

func (t TokenManager) Issue(userID string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(t.ttl)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
}

func (t TokenManager) Validate(token string) (string, error) {
	parsed, err := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{}, func(tok *jwt.Token) (any, error) {
		if _, ok := tok.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return t.secret, nil
	})
	if err != nil {
		return "", err
	}
	claims, ok := parsed.Claims.(*jwt.RegisteredClaims)
	if !ok || !parsed.Valid {
		return "", errors.New("invalid token")
	}
	return claims.Subject, nil
}

// ---- Repository -----------------------------------------------------------

type Repo struct{ pool *pgxpool.Pool }

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

func (r *Repo) SaveOTP(ctx context.Context, phone, hash string, expires time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO otp_codes (phone, code_hash, expires_at) VALUES ($1, $2, $3)`,
		phone, hash, expires)
	return err
}

// ConsumeOTP marks a matching, unexpired, unconsumed code as used. Returns true if found.
func (r *Repo) ConsumeOTP(ctx context.Context, phone, hash string) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE otp_codes SET consumed = TRUE
		WHERE id = (
			SELECT id FROM otp_codes
			WHERE phone = $1 AND code_hash = $2 AND consumed = FALSE AND expires_at > now()
			ORDER BY created_at DESC LIMIT 1
		)`, phone, hash)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// SaveRefresh stores a hashed refresh token for a user.
func (r *Repo) SaveRefresh(ctx context.Context, userID, hash string, expires time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, hash, expires)
	return err
}

// ConsumeRefresh validates a refresh token hash, revokes it (single-use rotation),
// and returns the owning user id. Returns ok=false if missing/expired/already used.
func (r *Repo) ConsumeRefresh(ctx context.Context, hash string) (userID string, ok bool, err error) {
	err = r.pool.QueryRow(ctx, `
		UPDATE refresh_tokens SET revoked = TRUE
		WHERE id = (
			SELECT id FROM refresh_tokens
			WHERE token_hash = $1 AND revoked = FALSE AND expires_at > now()
			LIMIT 1
		)
		RETURNING user_id`, hash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return userID, true, nil
}

// CleanupExpired deletes stale OTP rows and dead refresh tokens so the auth
// tables don't grow without bound. Meant to run periodically (see cmd/api).
func (r *Repo) CleanupExpired(ctx context.Context) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM otp_codes WHERE expires_at < now() - interval '1 day'`); err != nil {
		return err
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE expires_at < now() OR revoked = TRUE`)
	return err
}

func (r *Repo) UpsertUserByPhone(ctx context.Context, phone string) (domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (phone) VALUES ($1)
		ON CONFLICT (phone) DO UPDATE SET phone = EXCLUDED.phone
		RETURNING id, name, phone, onboarded, avatar_hue`, phone).
		Scan(&u.ID, &u.Name, &u.Phone, &u.Onboarded, &u.AvatarHue)
	if err != nil {
		return domain.User{}, err
	}
	u.InterestIDs, err = r.interestIDs(ctx, u.ID)
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

// ---- Service --------------------------------------------------------------

// Store is the persistence surface the service depends on (Repo in prod,
// fakes in tests).
type Store interface {
	SaveOTP(ctx context.Context, phone, hash string, expires time.Time) error
	ConsumeOTP(ctx context.Context, phone, hash string) (bool, error)
	SaveRefresh(ctx context.Context, userID, hash string, expires time.Time) error
	ConsumeRefresh(ctx context.Context, hash string) (string, bool, error)
	UpsertUserByPhone(ctx context.Context, phone string) (domain.User, error)
}

type ServiceOpts struct {
	// DevCode always verifies in Dev mode; outside dev it verifies only for
	// ReviewPhone (app-store review accounts). Empty disables the backdoor.
	DevCode     string
	ReviewPhone string
	Dev         bool
	RefreshTTL  time.Duration
}

type Service struct {
	repo         Store
	sms          SMSSender
	tokens       TokenManager
	opts         ServiceOpts
	requestLimit *httpx.RateLimiter // OTP sends per phone
	verifyLimit  *httpx.RateLimiter // verify attempts per phone
}

func NewService(repo Store, sms SMSSender, tokens TokenManager, opts ServiceOpts) *Service {
	return &Service{
		repo: repo, sms: sms, tokens: tokens, opts: opts,
		requestLimit: httpx.NewRateLimiter(3, time.Hour),
		verifyLimit:  httpx.NewRateLimiter(10, 10*time.Minute),
	}
}

func (s *Service) Tokens() TokenManager { return s.tokens }

// issueTokens mints a short-lived access JWT and a stored, rotating refresh token.
func (s *Service) issueTokens(ctx context.Context, userID string) (access, refresh string, err error) {
	access, err = s.tokens.Issue(userID)
	if err != nil {
		return "", "", err
	}
	refresh, err = randomToken()
	if err != nil {
		return "", "", err
	}
	if err := s.repo.SaveRefresh(ctx, userID, hashToken(refresh), time.Now().Add(s.opts.RefreshTTL)); err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

// Refresh validates and rotates a refresh token, returning a fresh access+refresh pair.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (access, refresh string, err error) {
	userID, ok, err := s.repo.ConsumeRefresh(ctx, hashToken(refreshToken))
	if err != nil {
		return "", "", err
	}
	if !ok {
		return "", "", errInvalidRefresh
	}
	return s.issueTokens(ctx, userID)
}

// normalizePhone strips formatting (spaces, dashes, parens), keeping a single
// leading + so numbers compare and store consistently.
func normalizePhone(p string) string {
	p = strings.TrimSpace(p)
	var b strings.Builder
	for i, r := range p {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else if r == '+' && i == 0 {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// validPhone accepts E.164-style numbers: optional +, 7–15 digits.
func validPhone(p string) bool {
	digits := strings.TrimPrefix(p, "+")
	if len(digits) < 7 || len(digits) > 15 {
		return false
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func hashCode(phone, code string) string {
	sum := sha256.Sum256([]byte(phone + ":" + code))
	return hex.EncodeToString(sum[:])
}

func (s *Service) RequestOTP(ctx context.Context, phone string) error {
	phone = normalizePhone(phone)
	if !validPhone(phone) {
		return errInvalidPhone
	}
	if !s.requestLimit.Allow(phone) {
		return errRateLimited
	}
	code, err := randomCode()
	if err != nil {
		return err
	}
	if err := s.repo.SaveOTP(ctx, phone, hashCode(phone, code), time.Now().Add(otpTTL)); err != nil {
		return err
	}
	return s.sms.Send(ctx, phone, code)
}

// devCodeMatches reports whether the backdoor code applies: always in dev,
// only for the designated review phone outside dev, never when unset.
func (s *Service) devCodeMatches(phone, code string) bool {
	if s.opts.DevCode == "" || code != s.opts.DevCode {
		return false
	}
	return s.opts.Dev || (s.opts.ReviewPhone != "" && phone == normalizePhone(s.opts.ReviewPhone))
}

func (s *Service) VerifyOTP(ctx context.Context, phone, code string) (access, refresh string, user domain.User, err error) {
	phone = normalizePhone(phone)
	if !validPhone(phone) {
		return "", "", domain.User{}, errInvalidCode
	}
	if !s.verifyLimit.Allow(phone) {
		return "", "", domain.User{}, errRateLimited
	}
	ok := s.devCodeMatches(phone, code)
	if !ok {
		found, ferr := s.repo.ConsumeOTP(ctx, phone, hashCode(phone, code))
		if ferr != nil {
			return "", "", domain.User{}, ferr
		}
		ok = found
	}
	if !ok {
		return "", "", domain.User{}, errInvalidCode
	}
	user, err = s.repo.UpsertUserByPhone(ctx, phone)
	if err != nil {
		return "", "", domain.User{}, err
	}
	access, refresh, err = s.issueTokens(ctx, user.ID)
	if err != nil {
		return "", "", domain.User{}, err
	}
	return access, refresh, user, nil
}

var (
	errInvalidCode    = errors.New("invalid or expired code")
	errInvalidRefresh = errors.New("invalid or expired refresh token")
	errInvalidPhone   = errors.New("invalid phone number")
	errRateLimited    = errors.New("too many attempts, try again later")
)

func randomCode() (string, error) {
	var b strings.Builder
	for i := 0; i < 6; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		b.WriteString(n.String())
	}
	return b.String(), nil
}

// randomToken returns a 256-bit opaque token, hex-encoded.
func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// ---- HTTP -----------------------------------------------------------------

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type requestOTPReq struct {
	Phone string `json:"phone"`
}

func (h *Handler) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var req requestOTPReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	if err := h.svc.RequestOTP(r.Context(), req.Phone); err != nil {
		switch {
		case errors.Is(err, errInvalidPhone):
			httpx.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, errRateLimited):
			httpx.Error(w, http.StatusTooManyRequests, err.Error())
		default:
			httpx.Internal(w, r, err, "could not send code")
		}
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type verifyOTPReq struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

func (h *Handler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req verifyOTPReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	access, refresh, user, err := h.svc.VerifyOTP(r.Context(), req.Phone, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, errInvalidCode):
			httpx.Error(w, http.StatusUnauthorized, err.Error())
		case errors.Is(err, errRateLimited):
			httpx.Error(w, http.StatusTooManyRequests, err.Error())
		default:
			httpx.Internal(w, r, err, "could not verify")
		}
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"token": access, "refreshToken": refresh, "user": user})
}

type refreshReq struct {
	RefreshToken string `json:"refreshToken"`
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	access, refresh, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, errInvalidRefresh) {
			httpx.Error(w, http.StatusUnauthorized, err.Error())
			return
		}
		httpx.Internal(w, r, err, "could not refresh")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"token": access, "refreshToken": refresh})
}

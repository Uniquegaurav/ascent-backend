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
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kumargaurav/summit-backend/internal/domain"
	"github.com/kumargaurav/summit-backend/internal/httpx"
)

const otpTTL = 5 * time.Minute

// ---- SMS ------------------------------------------------------------------

// SMSSender delivers a one-time code. Swap LogSender for Twilio/MSG91 in prod.
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

type Service struct {
	repo    *Repo
	sms     SMSSender
	tokens  TokenManager
	devCode string
	dev     bool
}

func NewService(repo *Repo, sms SMSSender, tokens TokenManager, devCode string, dev bool) *Service {
	return &Service{repo: repo, sms: sms, tokens: tokens, devCode: devCode, dev: dev}
}

func (s *Service) Tokens() TokenManager { return s.tokens }

func normalizePhone(p string) string { return strings.TrimSpace(p) }

func hashCode(phone, code string) string {
	sum := sha256.Sum256([]byte(phone + ":" + code))
	return hex.EncodeToString(sum[:])
}

func (s *Service) RequestOTP(ctx context.Context, phone string) error {
	phone = normalizePhone(phone)
	if len(phone) < 7 {
		return errors.New("invalid phone")
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

func (s *Service) VerifyOTP(ctx context.Context, phone, code string) (string, domain.User, error) {
	phone = normalizePhone(phone)
	ok := s.dev && code == s.devCode
	if !ok {
		found, err := s.repo.ConsumeOTP(ctx, phone, hashCode(phone, code))
		if err != nil {
			return "", domain.User{}, err
		}
		ok = found
	}
	if !ok {
		return "", domain.User{}, errInvalidCode
	}
	user, err := s.repo.UpsertUserByPhone(ctx, phone)
	if err != nil {
		return "", domain.User{}, err
	}
	token, err := s.tokens.Issue(user.ID)
	if err != nil {
		return "", domain.User{}, err
	}
	return token, user, nil
}

var errInvalidCode = errors.New("invalid or expired code")

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
		httpx.Error(w, http.StatusBadRequest, err.Error())
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
	token, user, err := h.svc.VerifyOTP(r.Context(), req.Phone, req.Code)
	if err != nil {
		if errors.Is(err, errInvalidCode) {
			httpx.Error(w, http.StatusUnauthorized, err.Error())
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "could not verify")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"token": token, "user": user})
}

package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ctxKey int

const (
	userIDKey ctxKey = iota
	requestIDKey
)

// JSON writes v as a JSON response with the given status.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// Error writes a JSON error envelope.
func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]string{"error": msg})
}

// Internal logs the real error with request context, then writes a generic 500
// so internals never leak to clients but always reach the server logs.
func Internal(w http.ResponseWriter, r *http.Request, err error, msg string) {
	slog.Error("internal error", "err", err, "method", r.Method, "path", r.URL.Path, "request_id", GetRequestID(r))
	Error(w, http.StatusInternalServerError, msg)
}

// Decode parses a JSON request body into dst.
func Decode(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

// UserID returns the authenticated user id placed by RequireAuth.
func UserID(r *http.Request) string {
	if v, ok := r.Context().Value(userIDKey).(string); ok {
		return v
	}
	return ""
}

// RequireAuth validates the Bearer token via validate and stores the user id.
func RequireAuth(validate func(token string) (string, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			token := strings.TrimPrefix(header, "Bearer ")
			if token == "" || token == header {
				Error(w, http.StatusUnauthorized, "missing bearer token")
				return
			}
			uid, err := validate(token)
			if err != nil {
				Error(w, http.StatusUnauthorized, "invalid token")
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, uid)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Pagination parses ?limit= and ?before= (epoch millis) query params, used by
// the list endpoints. Responses stay plain arrays for app compatibility; the
// timestamp of the last returned row goes in the X-Next-Cursor header.
func Pagination(r *http.Request, defLimit, maxLimit int) (limit int, before *time.Time) {
	limit = defLimit
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 {
		limit = min(v, maxLimit)
	}
	if v, err := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64); err == nil && v > 0 {
		t := time.UnixMilli(v)
		before = &t
	}
	return limit, before
}

// NextCursor advertises the cursor for the following page when a full page
// was returned.
func NextCursor(w http.ResponseWriter, gotFullPage bool, lastEpochMs int64) {
	if gotFullPage && lastEpochMs > 0 {
		w.Header().Set("X-Next-Cursor", strconv.FormatInt(lastEpochMs, 10))
	}
}

// OptionalAuth resolves the Bearer token when present but lets anonymous
// requests through — used for guest-browsable endpoints (explore, catalog).
func OptionalAuth(validate func(token string) (string, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			token := strings.TrimPrefix(header, "Bearer ")
			if token != "" && token != header {
				if uid, err := validate(token); err == nil {
					r = r.WithContext(context.WithValue(r.Context(), userIDKey, uid))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GetRequestID returns the request id placed by RequestID (empty if absent).
func GetRequestID(r *http.Request) string {
	if v, ok := r.Context().Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// RequestID assigns each request a correlation id (honoring an inbound
// X-Request-ID) and echoes it on the response.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			buf := make([]byte, 8)
			if _, err := rand.Read(buf); err == nil {
				id = hex.EncodeToString(buf)
			}
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Recover converts panics into 500s.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic", "err", rec, "path", r.URL.Path)
				Error(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// RequestLogger logs each request with latency.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		slog.Info("request", "method", r.Method, "path", r.URL.Path, "status", sw.status, "ms", time.Since(start).Milliseconds(), "request_id", GetRequestID(r))
	})
}

// CORS allows the mobile client / local tools to call the API.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

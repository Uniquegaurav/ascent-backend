package config

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

// loadDotEnv loads KEY=VALUE pairs from a .env file in the working dir, without
// overriding variables already set in the environment. Best-effort (no-op if absent).
func loadDotEnv() {
	f, err := os.Open(".env")
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key, val = strings.TrimSpace(key), strings.TrimSpace(val)
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
}

// defaultJWTSecret is only ever acceptable in dev mode; Validate rejects it in prod.
const defaultJWTSecret = "dev-secret-change-me-please-0000000000"

// Config holds runtime configuration sourced from the environment.
type Config struct {
	Env             string
	Port            string
	DatabaseURL     string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	// OTPDevCode always verifies in dev; in prod it verifies only for OTPReviewPhone
	// (app-store review accounts). Empty disables the backdoor entirely.
	OTPDevCode      string
	OTPReviewPhone  string
	GooglePlacesKey string
	// SMSProvider selects OTP delivery: "log" (dev, codes to stdout) or "msg91".
	SMSProvider     string
	MSG91AuthKey    string
	MSG91TemplateID string
}

func Load() Config {
	loadDotEnv()
	return Config{
		// Fail closed: an unset ENV means prod. Local dev must opt in via ENV=dev.
		Env:             getenv("ENV", "prod"),
		Port:            getenv("PORT", "8080"),
		DatabaseURL:     getenv("DATABASE_URL", "postgres://summit:summit@localhost:5432/summit?sslmode=disable"),
		JWTSecret:       getenv("JWT_SECRET", defaultJWTSecret),
		AccessTokenTTL:  getdur("ACCESS_TOKEN_TTL", 1*time.Hour),
		RefreshTokenTTL: getdur("REFRESH_TOKEN_TTL", 2160*time.Hour), // 90 days
		OTPDevCode:      getenv("OTP_DEV_CODE", ""),
		OTPReviewPhone:  getenv("OTP_REVIEW_PHONE", ""),
		GooglePlacesKey: getenv("GOOGLE_PLACES_KEY", ""),
		SMSProvider:     getenv("SMS_PROVIDER", "log"),
		MSG91AuthKey:    getenv("MSG91_AUTH_KEY", ""),
		MSG91TemplateID: getenv("MSG91_TEMPLATE_ID", ""),
	}
}

func (c Config) IsDev() bool { return c.Env == "dev" }

// Validate rejects configurations that would be unsafe to run outside dev and
// disables unsafe optional features (pointer receiver: it may amend fields).
func (c *Config) Validate() error {
	if c.IsDev() {
		return nil
	}
	if c.JWTSecret == defaultJWTSecret {
		return fmt.Errorf("JWT_SECRET is the built-in dev default; set a real secret (ENV=%q)", c.Env)
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters (got %d)", len(c.JWTSecret))
	}
	// A dev code without a review phone would be a global backdoor in prod.
	// Disable the code (fail closed on the feature) rather than refusing to
	// boot — a failed deploy would leave the previous build serving instead.
	if c.OTPDevCode != "" && c.OTPReviewPhone == "" {
		slog.Warn("OTP_DEV_CODE set without OTP_REVIEW_PHONE in non-dev env; disabling the dev code. Set OTP_REVIEW_PHONE to scope it to a review/demo phone.")
		c.OTPDevCode = ""
	}
	switch c.SMSProvider {
	case "msg91":
		if c.MSG91AuthKey == "" || c.MSG91TemplateID == "" {
			return fmt.Errorf("SMS_PROVIDER=msg91 requires MSG91_AUTH_KEY and MSG91_TEMPLATE_ID")
		}
	case "log", "":
		// Boot is allowed so demos keep working, but nobody without the review
		// phone can log in: codes only go to stdout.
		slog.Warn("SMS_PROVIDER=log in non-dev environment; OTP codes are not delivered to users")
	default:
		return fmt.Errorf("unknown SMS_PROVIDER %q (want log or msg91)", c.SMSProvider)
	}
	return nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getdur(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

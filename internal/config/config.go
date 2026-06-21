package config

import (
	"bufio"
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

// Config holds runtime configuration sourced from the environment.
type Config struct {
	Env             string
	Port            string
	DatabaseURL     string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	OTPDevCode      string
	GooglePlacesKey string
}

func Load() Config {
	loadDotEnv()
	return Config{
		Env:             getenv("ENV", "dev"),
		Port:            getenv("PORT", "8080"),
		DatabaseURL:     getenv("DATABASE_URL", "postgres://summit:summit@localhost:5432/summit?sslmode=disable"),
		JWTSecret:       getenv("JWT_SECRET", "dev-secret-change-me-please-0000000000"),
		AccessTokenTTL:  getdur("ACCESS_TOKEN_TTL", 1*time.Hour),
		RefreshTokenTTL: getdur("REFRESH_TOKEN_TTL", 2160*time.Hour), // 90 days
		OTPDevCode:      getenv("OTP_DEV_CODE", "111111"),
		GooglePlacesKey: getenv("GOOGLE_PLACES_KEY", ""),
	}
}

func (c Config) IsDev() bool { return c.Env == "dev" || c.Env == "" }

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

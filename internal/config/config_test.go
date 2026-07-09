package config

import (
	"strings"
	"testing"
	"time"
)

func prodConfig() Config {
	return Config{
		Env:             "prod",
		JWTSecret:       strings.Repeat("s", 40),
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 2160 * time.Hour,
		SMSProvider:     "log",
	}
}

func TestValidateProdRejectsDefaultSecret(t *testing.T) {
	c := prodConfig()
	c.JWTSecret = defaultJWTSecret
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for default JWT secret in prod")
	}
}

func TestValidateProdRejectsShortSecret(t *testing.T) {
	c := prodConfig()
	c.JWTSecret = "short"
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for short JWT secret in prod")
	}
}

func TestValidateProdDisablesUnscopedDevCode(t *testing.T) {
	c := prodConfig()
	c.OTPDevCode = "111111"
	if err := c.Validate(); err != nil {
		t.Fatalf("unscoped dev code should not block boot: %v", err)
	}
	if c.OTPDevCode != "" {
		t.Fatal("unscoped dev code must be disabled in prod")
	}

	c = prodConfig()
	c.OTPDevCode = "111111"
	c.OTPReviewPhone = "+919999999999"
	if err := c.Validate(); err != nil {
		t.Fatalf("review-phone-scoped dev code should be allowed: %v", err)
	}
	if c.OTPDevCode != "111111" {
		t.Fatal("scoped dev code must be preserved")
	}
}

func TestValidateProdRejectsIncompleteMSG91(t *testing.T) {
	c := prodConfig()
	c.SMSProvider = "msg91"
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for msg91 without credentials")
	}
	c.MSG91AuthKey, c.MSG91TemplateID = "key", "tpl"
	if err := c.Validate(); err != nil {
		t.Fatalf("complete msg91 config should validate: %v", err)
	}
}

func TestValidateDevAllowsAnything(t *testing.T) {
	c := Config{Env: "dev", JWTSecret: defaultJWTSecret}
	if err := c.Validate(); err != nil {
		t.Fatalf("dev mode should not enforce prod constraints: %v", err)
	}
}

func TestEmptyEnvIsNotDev(t *testing.T) {
	// Fail closed: a missing ENV must be treated as prod, never dev.
	if (Config{Env: ""}).IsDev() {
		t.Fatal("empty ENV must not be dev mode")
	}
	if (Config{Env: "prod"}).IsDev() {
		t.Fatal("prod must not be dev mode")
	}
	if !(Config{Env: "dev"}).IsDev() {
		t.Fatal("ENV=dev must be dev mode")
	}
}

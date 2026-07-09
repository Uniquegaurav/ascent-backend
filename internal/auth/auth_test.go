package auth

import (
	"context"
	"testing"
	"time"

	"github.com/kumargaurav/summit-backend/internal/domain"
)

// fakeStore is an in-memory Store for exercising Service logic without Postgres.
type fakeStore struct {
	otps      map[string]string // phone -> hash
	refreshes map[string]string // hash -> userID
}

func newFakeStore() *fakeStore {
	return &fakeStore{otps: map[string]string{}, refreshes: map[string]string{}}
}

func (f *fakeStore) SaveOTP(_ context.Context, phone, hash string, _ time.Time) error {
	f.otps[phone] = hash
	return nil
}

func (f *fakeStore) ConsumeOTP(_ context.Context, phone, hash string) (bool, error) {
	if f.otps[phone] == hash {
		delete(f.otps, phone)
		return true, nil
	}
	return false, nil
}

func (f *fakeStore) SaveRefresh(_ context.Context, userID, hash string, _ time.Time) error {
	f.refreshes[hash] = userID
	return nil
}

func (f *fakeStore) ConsumeRefresh(_ context.Context, hash string) (string, bool, error) {
	uid, ok := f.refreshes[hash]
	if ok {
		delete(f.refreshes, hash)
	}
	return uid, ok, nil
}

func (f *fakeStore) UpsertUserByPhone(_ context.Context, phone string) (domain.User, error) {
	return domain.User{ID: "user-" + phone, Phone: phone}, nil
}

// captureSender records the last code instead of sending SMS.
type captureSender struct {
	phone, code string
}

func (c *captureSender) Send(_ context.Context, phone, code string) error {
	c.phone, c.code = phone, code
	return nil
}

func newTestService(opts ServiceOpts) (*Service, *fakeStore, *captureSender) {
	if opts.RefreshTTL == 0 {
		opts.RefreshTTL = time.Hour
	}
	store := newFakeStore()
	sender := &captureSender{}
	tokens := NewTokenManager("test-secret-0123456789-0123456789", time.Hour)
	return NewService(store, sender, tokens, opts), store, sender
}

func TestTokenManagerRoundTrip(t *testing.T) {
	tm := NewTokenManager("secret-0123456789-0123456789", time.Hour)
	tok, err := tm.Issue("u1")
	if err != nil {
		t.Fatal(err)
	}
	uid, err := tm.Validate(tok)
	if err != nil || uid != "u1" {
		t.Fatalf("got uid=%q err=%v", uid, err)
	}
	other := NewTokenManager("different-secret-9876543210987654", time.Hour)
	if _, err := other.Validate(tok); err == nil {
		t.Fatal("token signed with another secret must not validate")
	}
}

func TestNormalizeAndValidatePhone(t *testing.T) {
	if got := normalizePhone(" +91 98765-43210 "); got != "+919876543210" {
		t.Fatalf("normalize: got %q", got)
	}
	for phone, want := range map[string]bool{
		"+919876543210": true,
		"9876543210":    true,
		"12345":         false,
		"+1234567890123456789": false,
	} {
		if validPhone(phone) != want {
			t.Errorf("validPhone(%q) != %v", phone, want)
		}
	}
}

func TestOTPFullFlow(t *testing.T) {
	svc, _, sender := newTestService(ServiceOpts{Dev: false})
	ctx := context.Background()

	if err := svc.RequestOTP(ctx, "+919876543210"); err != nil {
		t.Fatal(err)
	}
	if len(sender.code) != 6 {
		t.Fatalf("expected 6-digit code, got %q", sender.code)
	}
	access, refresh, user, err := svc.VerifyOTP(ctx, "+91 98765 43210", sender.code)
	if err != nil {
		t.Fatal(err)
	}
	if access == "" || refresh == "" || user.Phone != "+919876543210" {
		t.Fatalf("unexpected result: %q %q %+v", access, refresh, user)
	}
	// Single use: same code must not verify twice.
	if _, _, _, err := svc.VerifyOTP(ctx, "+919876543210", sender.code); err == nil {
		t.Fatal("consumed OTP must not verify again")
	}
}

func TestDevCodeScopedToReviewPhoneInProd(t *testing.T) {
	svc, _, _ := newTestService(ServiceOpts{
		Dev: false, DevCode: "111111", ReviewPhone: "+919999999999",
	})
	ctx := context.Background()

	if _, _, _, err := svc.VerifyOTP(ctx, "+919876543210", "111111"); err == nil {
		t.Fatal("dev code must not work for arbitrary phones in prod")
	}
	if _, _, _, err := svc.VerifyOTP(ctx, "+919999999999", "111111"); err != nil {
		t.Fatalf("dev code must work for the review phone: %v", err)
	}
}

func TestDevCodeDisabledWhenEmpty(t *testing.T) {
	svc, _, _ := newTestService(ServiceOpts{Dev: true, DevCode: ""})
	if _, _, _, err := svc.VerifyOTP(context.Background(), "+919876543210", ""); err == nil {
		t.Fatal("empty dev code must never verify")
	}
}

func TestDevCodeWorksInDev(t *testing.T) {
	svc, _, _ := newTestService(ServiceOpts{Dev: true, DevCode: "000000"})
	if _, _, _, err := svc.VerifyOTP(context.Background(), "+919876543210", "000000"); err != nil {
		t.Fatalf("dev code must work in dev: %v", err)
	}
}

func TestRequestOTPRateLimit(t *testing.T) {
	svc, _, _ := newTestService(ServiceOpts{Dev: true})
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if err := svc.RequestOTP(ctx, "+919876543210"); err != nil {
			t.Fatalf("request %d: %v", i+1, err)
		}
	}
	if err := svc.RequestOTP(ctx, "+919876543210"); err != errRateLimited {
		t.Fatalf("4th request should be rate limited, got %v", err)
	}
	// Other phones are unaffected.
	if err := svc.RequestOTP(ctx, "+919876500000"); err != nil {
		t.Fatalf("other phone should not be limited: %v", err)
	}
}

func TestVerifyOTPAttemptLimit(t *testing.T) {
	svc, _, _ := newTestService(ServiceOpts{Dev: false})
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		if _, _, _, err := svc.VerifyOTP(ctx, "+919876543210", "000000"); err != errInvalidCode {
			t.Fatalf("attempt %d: expected invalid code, got %v", i+1, err)
		}
	}
	if _, _, _, err := svc.VerifyOTP(ctx, "+919876543210", "000000"); err != errRateLimited {
		t.Fatalf("11th attempt should be rate limited, got %v", err)
	}
}

func TestRefreshRotation(t *testing.T) {
	svc, store, _ := newTestService(ServiceOpts{Dev: true, DevCode: "000000"})
	ctx := context.Background()
	_, refresh, _, err := svc.VerifyOTP(ctx, "+919876543210", "000000")
	if err != nil {
		t.Fatal(err)
	}
	access2, refresh2, err := svc.Refresh(ctx, refresh)
	if err != nil || access2 == "" || refresh2 == "" {
		t.Fatalf("rotation failed: %v", err)
	}
	// Old refresh token is single-use.
	if _, _, err := svc.Refresh(ctx, refresh); err != errInvalidRefresh {
		t.Fatalf("reused refresh token should fail, got %v", err)
	}
	if len(store.refreshes) != 1 {
		t.Fatalf("expected exactly 1 live refresh token, got %d", len(store.refreshes))
	}
}

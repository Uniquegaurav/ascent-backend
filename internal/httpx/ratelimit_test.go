package httpx

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterBlocksOverLimit(t *testing.T) {
	l := NewRateLimiter(2, time.Minute)
	if !l.Allow("a") || !l.Allow("a") {
		t.Fatal("first two hits should pass")
	}
	if l.Allow("a") {
		t.Fatal("third hit should be blocked")
	}
	if !l.Allow("b") {
		t.Fatal("other keys are independent")
	}
}

func TestRateLimiterWindowResets(t *testing.T) {
	l := NewRateLimiter(1, 10*time.Millisecond)
	if !l.Allow("a") {
		t.Fatal("first hit should pass")
	}
	if l.Allow("a") {
		t.Fatal("second hit inside window should be blocked")
	}
	time.Sleep(15 * time.Millisecond)
	if !l.Allow("a") {
		t.Fatal("hit after window should pass")
	}
}

func TestClientIP(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	if got := ClientIP(r); got != "10.0.0.1" {
		t.Fatalf("remote addr: got %q", got)
	}
	r.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.1")
	if got := ClientIP(r); got != "203.0.113.7" {
		t.Fatalf("forwarded: got %q", got)
	}
}

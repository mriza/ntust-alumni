package handlers

import (
	"testing"
	"time"
)

func TestOTPRateLimiterAllowsUnderMax(t *testing.T) {
	rl := NewRateLimiter()
	for i := 0; i < 5; i++ {
		allowed, _ := rl.CheckAndIncrement("tok1", 5)
		if !allowed {
			t.Fatalf("expected allowed at %d", i)
		}
	}
}

func TestOTPRateLimiterBlocksAfterMax(t *testing.T) {
	rl := NewRateLimiter()
	for i := 0; i < 5; i++ {
		rl.CheckAndIncrement("tok2", 5)
	}
	allowed, lockUntil := rl.CheckAndIncrement("tok2", 5)
	if allowed {
		t.Fatal("expected blocked after max attempts")
	}
	if lockUntil.IsZero() {
		t.Fatal("expected lockUntil to be set")
	}
}

func TestOTPRateLimiterResetsAfterExpiry(t *testing.T) {
	rl := NewRateLimiter()
	// Force expiry
	rl.CheckAndIncrement("tok3", 1)
	rl.attempts["tok3"] = rateLimitEntry{
		count:   5,
		firstAt: time.Now().Add(-20 * time.Minute),
		lastAt:  time.Now().Add(-20 * time.Minute),
	}
	allowed, _ := rl.CheckAndIncrement("tok3", 5)
	if !allowed {
		t.Fatal("expected allowed after expiry")
	}
}

func TestRateLimiterCleanupEvictsExpiredNonLockedEntries(t *testing.T) {
	rl := NewRateLimiter()
	rl.attempts["stale"] = rateLimitEntry{
		count:   1,
		firstAt: time.Now().Add(-20 * time.Minute),
		lastAt:  time.Now().Add(-20 * time.Minute),
	}
	rl.attempts["fresh"] = rateLimitEntry{
		count:   1,
		firstAt: time.Now(),
		lastAt:  time.Now(),
	}
	rl.Cleanup()
	if _, ok := rl.attempts["stale"]; ok {
		t.Fatal("expected stale unlocked entry to be evicted")
	}
	if _, ok := rl.attempts["fresh"]; !ok {
		t.Fatal("expected fresh entry to remain")
	}
}

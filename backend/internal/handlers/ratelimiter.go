package handlers

import (
	"sync"
	"time"
)

const (
	rateLimitWindow       = 15 * time.Minute
	maxRateLimitEntries   = 10000
	cleanupInterval       = time.Minute
	cleanupTriggerEntries = 512
)

// RateLimiter menghitung per cobaan per kunci dengan expiry TTL.
type RateLimiter struct {
	mu          sync.Mutex
	attempts    map[string]rateLimitEntry
	lastCleanup time.Time
}

type rateLimitEntry struct {
	count       int
	firstAt     time.Time
	lastAt      time.Time
	lockedUntil time.Time
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		attempts:    make(map[string]rateLimitEntry),
		lastCleanup: time.Now(),
	}
}

func (rl *RateLimiter) cleanupLocked(now time.Time) {
	for k, v := range rl.attempts {
		if (!v.lockedUntil.IsZero() && now.After(v.lockedUntil)) ||
			(v.lockedUntil.IsZero() && now.Sub(v.firstAt) > rateLimitWindow) {
			delete(rl.attempts, k)
		}
	}
	rl.lastCleanup = now
}

// CheckAndIncrement memeriksa apakah kunci masih di bawah limit, lalu menambahkan 1 attempt.
// Jika sudah tercapai limit (max), kembalikan lockedUntil.
func (rl *RateLimiter) CheckAndIncrement(key string, max int) (bool, time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if (len(rl.attempts) >= cleanupTriggerEntries && now.Sub(rl.lastCleanup) > cleanupInterval) ||
		len(rl.attempts) >= maxRateLimitEntries {
		rl.cleanupLocked(now)
	}
	if _, exists := rl.attempts[key]; !exists && len(rl.attempts) >= maxRateLimitEntries {
		// Batasi ukuran map agar tidak memicu OOM jika semua entri aktif
		for k := range rl.attempts {
			delete(rl.attempts, k)
			if len(rl.attempts) < maxRateLimitEntries {
				break
			}
		}
	}

	e, ok := rl.attempts[key]
	if !ok {
		e = rateLimitEntry{firstAt: now, lastAt: now}
	}

	// Jika ada lock aktif, kembalikan sisa waktu
	if !e.lockedUntil.IsZero() && now.Before(e.lockedUntil) {
		return false, e.lockedUntil
	}

	// Jika lama kedaluwarsa, reset window
	if !e.firstAt.IsZero() && now.Sub(e.firstAt) > rateLimitWindow {
		e = rateLimitEntry{firstAt: now, lastAt: now}
	}

	e.count++
	e.lastAt = now

	allowed := e.count <= max
	var lockUntil time.Time
	if !allowed {
		lockUntil = now.Add(rateLimitWindow)
		e.lockedUntil = lockUntil
	}

	rl.attempts[key] = e
	return allowed, lockUntil
}

// Cleanup menghapus entri yang sudah kedaluwarsa.
func (rl *RateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.cleanupLocked(time.Now())
}

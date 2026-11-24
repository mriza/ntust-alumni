package handlers

import (
	"sync"
	"time"
)

// RateLimiter menghitung per cobaan per kunci dengan expiry TTL.
type RateLimiter struct {
	mu      sync.Mutex
	attempts map[string]rateLimitEntry
}

type rateLimitEntry struct {
	count      int
	firstAt    time.Time
	lastAt     time.Time
	lockedUntil time.Time
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{attempts: make(map[string]rateLimitEntry)}
}

// CheckAndIncrement memeriksa apakah kunci masih di bawah limit, lalu menambahkan 1 attempt.
// Jika sudah tercapai limit (max), kembalikan lockedUntil.
func (rl *RateLimiter) CheckAndIncrement(key string, max int) (bool, time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	e, ok := rl.attempts[key]
	if !ok {
		e = rateLimitEntry{firstAt: time.Now(), lastAt: time.Now()}
	}

	// Jika ada lock aktif, kembalikan sisa waktu
	if !e.lockedUntil.IsZero() && time.Now().Before(e.lockedUntil) {
		return false, e.lockedUntil
	}

	// Jika lama kedaluwarsa dan bukan lock, reset
	now := time.Now()
	if !e.firstAt.IsZero() && now.Sub(e.firstAt) > 15*time.Minute {
		e = rateLimitEntry{firstAt: now, lastAt: now}
	}

	e.count++
	e.lastAt = now

	allowed := e.count <= max
	var lockUntil time.Time
	if !allowed {
		lockUntil = now.Add(15 * time.Minute)
		e.lockedUntil = lockUntil
	}

	rl.attempts[key] = e
	return allowed, lockUntil
}

// Cleanup menghapus entri yang sudah kedaluwarsa
func (rl *RateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for k, v := range rl.attempts {
		if !v.lockedUntil.IsZero() && time.Now().After(v.lockedUntil) {
			delete(rl.attempts, k)
		}
	}
}

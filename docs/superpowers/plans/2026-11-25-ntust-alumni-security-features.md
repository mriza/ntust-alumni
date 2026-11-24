# NTUST Alumni Security Enhancements - Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix all security audit findings (Critical, High, Medium, Low) and add directory search, filter, profile photo upload, and health check detail.

**Architecture:** Smallest focused changes — no new deps, no architecture changes. Backend changes in auth, verification, alumni, middleware, email. Frontend changes in app.js only.

**Tech Stack:** Go 1.22 + Gin, CouchDB, Vanilla JS SPA. Same as existing.

**Spec:** `docs/superpowers/specs/2026-11-25-ntust-alumni-security-features.md`

## Global Constraints

- JWT_SECRET: required env var, no default
- OTP: max 5 attempts per token, then lockout 15 min
- Photo: max 2MB, base64 in CouchDB, MIME type validation
- Security headers: X-Frame-Options, X-Content-Type-Options, CSP basics
- Rate limiting: login = 10/5min/IP, request-verification = 5/10min/user

## Review Focus

1. **JWT_SECRET default in main.go** — should be removed, app must fail when not set
2. **OTP brute-force on Verify endpoint** — must track attempts per verification token
3. **Frontend XSS** — all innerHTML with user data must use textContent
4. **Login rate limiting** — per-IP and per-email with cooldown
5. **Email enumeration in Register** — generic response for duplicate
6. **Security headers missing** — in both nginx and Go backend
7. **Health check with CouchDB status** — must not panic if CouchDB down
8. **Photo upload validation** — max 2MB, valid image MIME, stored in CouchDB doc
9. **Search/filter query params** — must be optional and backward-compatible

---

### Task 1: JWT_SECRET removal (Critical)

**Files:**
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/middleware/auth.go`

**Interfaces:**
- Consumes: nothing
- Produces: same — `InitJWT(secret string)`, `SetJWTKey(key []byte)`

- [ ] **Step 1: Write failing test**

Test that NewCouchDB fails when JWT_SECRET is empty.

```go
func TestMainFailsWithoutJWTSecret(t *testing.T) {
    // Set env empty
    os.Setenv("JWT_SECRET", "")
    // Should fail - we check via env check in main
}
```

- [ ] **Step 2: Implement JWT_SECRET validation in main.go**

```go
jwtSecret := os.Getenv("JWT_SECRET")
if jwtSecret == "" {
    log.Fatal("JWT_SECRET environment variable is required; refusing to start with insecure default")
}
```

- [ ] **Step 3: Run tests**

`go test ./cmd/server/... -v` → PASS

- [ ] **Step 4: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "fix: remove hardcoded JWT_SECRET default, fail fast when missing"
```

---

### Task 2: OTP Rate Limiting (High)

**Files:**
- Create: `backend/internal/handlers/ratelimit.go`
- Modify: `backend/internal/handlers/verification.go`

**Interfaces:**
- Consumes: none
- Produces: `ratelimit.CheckAndIncrement(key string, max int) (allowed bool, resetsAt time.Time)`

- [ ] **Step 1: Write failing test for OTP rate limiter**

```go
func TestOTPRateLimit(t *testing.T) {
    rl := NewOTPRateLimiter()
    // 5 attempts ok
    for i := 0; i < 5; i++ {
        allowed, _ := rl.CheckAndIncrement("token1", 5)
        assert.True(allowed)
    }
    // 6th fails
    allowed, _ := rl.CheckAndIncrement("token1", 5)
    assert.False(allowed)
}
```

- [ ] **Step 2: Implement ratelimiter.go**

Simple in-memory counter with TTL (15 min) — supports token-based and IP-based keys.

- [ ] **Step 3: Integrate into Verify handler**

Add rate limit check before OTP comparison.

- [ ] **Step 4: Run tests and commit**

```bash
git add backend/internal/handlers/ratelimit.go backend/internal/handlers/verification.go backend/internal/handlers/verification_test.go
git commit -m "fix: add OTP brute-force rate limiting (max 5 attempts, 15 min lock)"
```

---

### Task 3: Frontend XSS Fix (High)

**Files:**
- Modify: `frontend/js/app.js`

**Interfaces:**
- Consumes: none
- Produces: same API, but safe rendering

- [ ] **Step 1: Write tests**

Node-based HTML safety test: verify all innerHTML usages escaped, or use textContent.

- [ ] **Step 2: Create safe render helpers**

```javascript
function esc(str) { return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#x27;'); }
```

- [ ] **Step 3: Replace innerHTML with textContent or escaped content**

Replace every `innerHTML` with user data using textContent.

- [ ] **Step 4: Run tests and commit**

```bash
git add frontend/js/app.js
git commit -m "fix: escape HTML in frontend (XSS via innerHTML)"
```

---

### Task 4: Login Rate Limiting (Medium/High)

**Files:**
- Modify: `backend/internal/handlers/auth.go`
- Modify: `backend/internal/handlers/auth_test.go`

**Interfaces:**
- Consumes: ratelimiter from Task 2
- Produces: login with rate limiting

- [ ] **Step 1: Write failing test**

```go
func TestLoginRateLimit(t *testing.T) {
    // 5 attempts ok, 6th fails
}
```

- [ ] **Step 2: Add rate limiting to Login handler**

Use `CheckAndIncrement(email, 5)` + `CheckAndIncrement(remoteAddr, 10)`

- [ ] **Step 3: Run tests and commit**

---

### Task 5: Register Email Enumeration Fix (Low)

**Files:**
- Modify: `backend/internal/handlers/auth.go`

**Interfaces:**
- Produces: same, but same response for duplicate/new

- [ ] **Step 1: Write failing test**

- [ ] **Step 2: Change 409 to generic 200 for duplicates**

- [ ] **Step 3: Run tests and commit**

---

### Task 6: Security Headers (Low)

**Files:**
- Modify: `backend/cmd/server/main.go`
- Create: `backend/Dockerfile` (minimal CSP config)

**Interfaces:**
- Produces: Go server with security headers

- [ ] **Step 1: Write test for headers**

- [ ] **Step 2: Add middleware to main.go**

```go
router.Use(func(c *gin.Context) {
    c.Header("X-Content-Type-Options", "nosniff")
    c.Header("X-Frame-Options", "SAMEORIGIN")
    c.Header("X-XSS-Protection", "1; mode=block")
    c.Header("Strict-Transport-Security", "max-age=31536000")
    c.Next()
})
```

- [ ] **Step 3: Run tests and commit**

---

### Task 7: Health Check Detail (New)

**Files:**
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/db/couch.go`

**Interfaces:**
- Produces: `GET /health` returns `{"status":"ok","couchdb":{"status":"ok","latency_ms":5}}`

- [ ] **Step 1: Write failing test**

- [ ] **Step 2: Add Ping method to db package**

```go
func (c *CouchDB) PingWithLatency() (time.Duration, error)
```

- [ ] **Step 3: Update health endpoint**

- [ ] **Step 4: Run tests and commit**

---

### Task 8: Directory Search & Filter (New)

**Files:**
- Modify: `backend/internal/handlers/alumni.go`
- Modify: `backend/internal/db/couch.go`
- Modify: `frontend/js/app.js`

**Interfaces:**
- Produces: `GET /api/alumni?q=search&wilayah=X&jurusan=Y&limit=N&offset=M`

- [ ] **Step 1: Write failing test**

- [ ] **Step 2: Add search/filter to ListAlumni**

- [ ] **Step 3: Add to frontend**

---

### Task 9: Profile Photo Upload (New)

**Files:**
- Modify: `backend/internal/handlers/alumni.go`

**Interfaces:**
- Produces: `POST /api/alumni/:id` accepts `foto_profil` (base64, max 2MB)

- [ ] **Step 1: Write failing test**

- [ ] **Step 2: Add photo validation and storage**

- [ ] **Step 3: Run tests and commit**

---

## Review Focus

1. JWT_SECRET fail-fast (Task 1)
2. OTP rate limiting (Task 2)
3. XSS in frontend (Task 3)
4. Login rate limiting (Task 4)
5. Email enumeration (Task 5)
6. Security headers (Task 6)
7. Health check details (Task 7)
8. Search/filter (Task 8)
9. Photo upload validation (Task 9)

# NTUST Alumni Security-First Enhancement Design

## Intent & Why
Complete the Alumni NTUST Indonesia application (Go backend + CouchDB + Vanilla JS SPA). Based on the security audit, several critical/high issues must be fixed before the app can be used in production. Additionally, add a few high-value features to make the directory more useful.

## Goals
- Fix all security audit findings (Critical, High, Medium, Low)
- Add search + filter to alumni directory
- Add simple profile photo upload
- Add health check detail endpoint
- Prepare app for production deployment

## Non-Goals
- Big infrastructure changes (no changes to Dockerfiles of existing services)
- Mobile-first design (still desktop-first)
- SSO integration (already exists via NTUST email verification)

## Anti-Goals
- Don't introduce new authentication methods
- Don't add new DB technologies
- Don't change the overall architecture

## Constraints
- Backend: Go 1.22 + Gin
- Frontend: Vanilla JS SPA (no build tools)
- Database: CouchDB
- Must keep existing routes and compatibility

---

## Approaches Considered

### A. Security-First + Core Fitur Baru (DIREKOMENDASIKAN) ✅
Fix semua security findings + tambah search/filter/photo/health-check. Fokus pada bug sebelum fitur baru.

### B. Minimal - Security Fixes Only
Fix findings tanpa fitur baru. Budget terendah, fokus pada yang kritis.

### C. All Features
Security fixes + fitur lengkap (photo, notifications, dashboard).

*Selected: A (Security-First + Core Fitur Baru).*

---

## Security Fixes (Dari Audit)

| F | Finding | Fix |
|---|---------|-----|
| F1 | JWT_SECRET default hardcoded | Remove default, fail fast when env not set |
| F2 | Frontend XSS (innerHTML) | Escape user content via textContent/createElement |
| F3 | OTP brute-force (no rate limit) | Add per-token attempt counter + lockout after 5 attempts |
| F4 | Register email enumeration (409 vs 201) | Return generic 200 for both cases |
| F5 | Login no rate limiting | Add per-email/IP rate limiting + exponential backoff |
| F6 | Security headers missing | Add: X-Frame-Options, X-Content-Type-Options, CSP basics |

## New Features

### 1. Health Check Detail
- `GET /health` returns `{"status": "ok", "couchdb": {"status": "ok", "latency_ms": 12}}`
- Backend pings CouchDB before returning ok

### 2. Search Directory Alumni
- `GET /api/alumni?q=search&page=1` — Search by name/nim/wilayah/pekerjaan
- Pagination support

### 3. Filter Directory
- `GET /api/alumni?wilayah=Jakarta&jurusan=Teknik`
- Multiple filters

### 4. Profile Photo Upload
- `POST /api/alumni/:id` — field `foto_profil` (base64 encoded, max 2MB)
- Store as base64 in CouchDB doc
- Validate image size and MIME type

### 5. Directory Improvements
- Already has wilayah filter (done in Task 6)
- Add search box (frontend)
- Add sort by name

---

## Parts of How (Decided by User)
- Security fixes: Active on deployment, not optional
- Photo: max 2MB, base64 in CouchDB
- Search: `?q=` param, case-insensitive match
- Health check: Poll every 30s on frontend, shows status indicator

---

##Builder check plan:
After writing this design, say "write the plan" and I will generate `docs/superpowers/plans/2026-11-25-ntust-alumni-security-features.md`.

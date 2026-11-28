# Design: Upgrade Forum ke Apache Answer Q&A + OAuth2 SSO

## Intent & Why
Mengganti forum diskusi datar sederhana (`forum_posts` di CouchDB) sepenuhnya dengan platform Q&A lengkap **Apache Answer** yang terintegrasi melalui **OAuth2 Single Sign-On (SSO)** ke portal Alumni NTUST Indonesia.

## Goals
- Menyediakan forum Q&A lengkap (pertanyaan, jawaban, accepted answer, voting, tag, pencarian) menggunakan **Apache Answer** di container `ntust-answer` (`http://localhost:9080`).
- Mengintegrasikan autentikasi satu pintu (SSO OAuth2) antara portal Alumni NTUST (`ntust-backend` / `ntust-frontend`) dan Apache Answer melalui plugin `connector-basic`: hanya alumni berstatus `verified` yang dapat login ke Q&A tanpa membuat akun terpisah.
- Mengganti halaman `#/forum` pada SPA frontend sepenuhnya dengan tampilan daftar pertanyaan Q&A langsung dari Apache Answer API dan tombol akses cepat/SSO ke Apache Answer.
- Menjaga seluruh kredensial (`ANSWER_ADMIN_PASSWORD`, `OAUTH_CLIENT_SECRET`, dll.) di file `.env` yang ter-gitignore.

## Architecture
- **Container Topology (`docker-compose.yml`)**:
  - `couchdb` (`ntust-couchdb`, port `5984`)
  - `backend` (`ntust-backend`, port `8080`) — penyedia REST API portal + OAuth2 Authorization/Token/UserInfo endpoints
  - `answer` (`ntust-answer`, port `9080:80`) — custom image dari `docker.io/apache/answer:latest` dengan plugin `github.com/apache/answer-plugins/connector-basic`, `AUTO_INSTALL=true`, SQLite persisten di volume `answer-data:/data`
  - `frontend` (`ntust-frontend`, port `8081:80`) — Nginx static SPA + reverse proxy `/api/` → `backend:8080` dan `/answer-api/` → `answer:80/answer/api/v1/`

- **Alur OAuth2 SSO**:
  1. User klik **Login dengan Alumni NTUST** di Apache Answer (`:9080`) → diarahkan ke `http://localhost:8081/#/oauth/authorize?client_id=...&redirect_uri=...&state=...`.
  2. Jika user belum login di portal, `#/oauth/authorize` menampilkan login form portal; jika masih `pending`, diarahkan ke `#/verify`.
  3. Untuk user `verified`, frontend memanggil `POST /api/oauth/authorize` → backend memvalidasi `client_id` & `redirect_uri` (allowlist ketat) dan menerbitkan one-time `code` (TTL 2 menit).
  4. Browser diarahkan kembali ke callback `connector-basic` di Apache Answer.
  5. Apache Answer (server-to-server) menukar `code` via `POST http://backend:8080/api/oauth/token` (Basic Auth / form-urlencoded) menjadi JWT berdurasi 5 menit, lalu mengambil profil flat JSON via `GET http://backend:8080/api/oauth/userinfo`.

## Security Controls
- `POST /api/oauth/authorize` dilindungi `AuthMiddleware()` + `RequireVerified()`.
- Allowlist ketat `OAUTH_ALLOWED_REDIRECT_PREFIX` (mencegah open redirect).
- Authorization code `crypto/rand` sekali pakai dengan masa berlaku 2 menit.
- Perbandingan `client_id` & `client_secret` menggunakan `subtle.ConstantTimeCompare`.
- Semua rahasia hanya disimpan di `.env` (gitignored) dengan template placeholder pada `.env.example`.

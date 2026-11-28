# Alumni NTUST Indonesia

Aplikasi manajemen data, direktori terverifikasi, subgroup wilayah, dan Forum Q&A terintegrasi **Apache Answer (OAuth2 SSO)** untuk alumni National Taiwan University of Science and Technology (NTUST) di Indonesia.

## Fitur Utama

- **Landing Page & Ant Design v5 UI/UX** — antarmuka HTML5 + CSS3 tokens & animations tanpa hardcoded/inline style, dengan tema **Light / Dark / System** yang dapat dipilih pengguna
- **Pendaftaran Alumni** — siapa pun dapat mendaftar dengan email apa pun (status awal `pending`)
- **Verifikasi Multi-Channel** — verifikasi identitas melalui email `@mail.ntust.edu.tw` (tautan unik + OTP 6 digit dari `ntust-alumni-verification@cerdas.club`), SMS OTP, atau WhatsApp OTP via Brevo API
- **Direktori Alumni** — cari (`?q=`), filter (`wilayah`, `jurusan`), urutkan (`sort`), dan paginasi profil alumni terverifikasi
- **Subgroup Per Wilayah** — ikatan alumni NTUST per wilayah beserta struktur pengurus wilayah (misal: Jabodetabek)
- **Penunjukan Admin Wilayah** — tunjuk alumni terverifikasi sebagai `regional_admin` pada wilayah terkait
- **Forum Q&A (Apache Answer + OAuth2 SSO)** — platform tanya-jawab lengkap (pertanyaan, jawaban, accepted answer, voting, tag) terintegrasi lewat OAuth2 SSO untuk alumni terverifikasi

## Tech Stack

- **Backend**: Go 1.22+ + Gin + CouchDB REST client (`net/http`) + OAuth2 Provider
- **Frontend**: HTML5 + CSS3 Design Tokens (`css/tokens.css`, `css/style.css`) + Vanilla JS SPA (`js/theme.js`, `js/app.js`) + Nginx reverse proxy (`Content-Security-Policy: style-src 'self'`)
- **Forum Q&A**: Apache Answer (`connector-basic` OAuth2 SSO) + SQLite
- **Database**: CouchDB 3.3
- **Email / SMS / WA**: Brevo API v3

## Mulai dengan Container (`docker-compose` / `podman-compose`)

```bash
# Salin template .env dan isi seluruh kredensial/secret (file .env di-gitignore)
cp .env.example .env
chmod 600 .env

# Build dan jalankan seluruh service (CouchDB & Backend hanya di-bind ke 127.0.0.1)
podman-compose --env-file .env up -d --build
# atau dengan Docker Compose:
docker compose --env-file .env up -d --build

# Aktifkan & konfigurasikan SSO OAuth2 pada Apache Answer + seed welcome Q&A
python3 scripts/configure_answer_sso.py

# Cek health endpoint via Nginx frontend
curl http://localhost:8081/health
```

- **Portal Alumni**: [http://localhost:8081](http://localhost:8081)
- **Forum Q&A (Apache Answer)**: [http://localhost:9080](http://localhost:9080)

### Lingkungan (Environment Variables)

Semua variabel sensitif disimpan di file `.env` (jangan pernah di-commit) dan dilewatkan per service sesuai prinsip *least privilege*. Lihat [`.env.example`](.env.example).

| Variable | Deskripsi | Default |
|----------|-----------|---------|
| `COUCHDB_USER` | Username CouchDB (**wajib** di `.env`) | (wajib diisi) |
| `COUCHDB_PASSWORD` | Password CouchDB (**wajib** di `.env`) | (wajib diisi) |
| `JWT_SECRET` | Secret untuk signing JWT sesi & token OAuth (**wajib**, min 32 karakter) | (wajib diisi) |
| `PORT` | Port HTTP backend | `8080` |
| `FRONTEND_URL` | URL base frontend untuk tautan verifikasi, SSO & CORS allowlist | `http://localhost:8081` |
| `CORS_ALLOWED_ORIGINS` | Origin tambahan untuk CORS (dipisah koma) | (kosong) |
| `VERIFICATION_ALLOWED_DOMAINS` | Domain email verifikasi yang diizinkan (dipisah koma) | `@mail.ntust.edu.tw` |
| `BREVO_API_KEY` | API key Brevo (`xkeysib-...` atau token MCP base64) | (kosong) |
| `BREVO_FROM_EMAIL` | Email pengirim verifikasi | `ntust-alumni-verification@cerdas.club` |
| `BREVO_FROM_NAME` | Nama pengirim | `Alumni NTUST Indonesia` |
| `BREVO_SMS_SENDER` | Sender ID SMS Brevo | `NTUSTAlumni` |
| `BREVO_WHATSAPP_SENDER` | Nomor pengirim WhatsApp Brevo | (kosong) |
| `BREVO_WHATSAPP_TEMPLATE_ID` | ID template WhatsApp Brevo | (kosong) |
| `ANSWER_SITE_URL` | URL publik Apache Answer | `http://localhost:9080` |
| `ANSWER_CONTACT_EMAIL` | Email kontak Apache Answer | `admin@ntust-alumni.id` |
| `ANSWER_ADMIN_NAME` | Username admin Apache Answer (tanpa spasi) | `admin` |
| `ANSWER_ADMIN_EMAIL` | Email admin Apache Answer | `admin@ntust-alumni.id` |
| `ANSWER_ADMIN_PASSWORD` | Password admin Apache Answer (**wajib**) | (wajib diisi) |
| `OAUTH_CLIENT_ID` | Client ID OAuth2 portal → Apache Answer | `ntust-alumni-portal` |
| `OAUTH_CLIENT_SECRET` | Client Secret OAuth2 (**wajib**) | (wajib diisi) |
| `OAUTH_ALLOWED_REDIRECT_PREFIX` | Prefix URL callback OAuth2 yang diizinkan | `http://localhost:9080/answer/api/v1/connector/` |
| `SEED_*_PASSWORD` | Password akun seed (`cmd/seed`) — hanya dipakai saat seeding | (wajib saat seed) |

## Development (tanpa Container)

```bash
cd backend
set -a; source ../.env; set +a
export COUCHDB_URL="http://${COUCHDB_USER}:${COUCHDB_PASSWORD}@localhost:5984"
go build ./...
go test ./...
go run ./cmd/server
```

## Struktur Project

```
ntust-alumni/
├── answer/                  # Custom Dockerfile Apache Answer + connector-basic
├── backend/
│   ├── cmd/seed/            # CLI seed akun & subgroup wilayah
│   ├── cmd/server/          # Entry point, HTTP server timeouts, CORS, security headers
│   ├── internal/db/         # CouchDB REST client (Basic Auth + ID validation)
│   ├── internal/models/     # Struct data
│   ├── internal/middleware/ # JWT HS256, verification, role middleware
│   ├── internal/handlers/   # Auth, alumni, subgroup, oauth, verification, rate limiter
│   ├── internal/email/      # Brevo API v3 client
│   └── Dockerfile           # Multi-stage non-root runtime (alpine:3.21)
├── frontend/
│   ├── css/tokens.css       # Ant Design v5 design tokens (Light / Dark / System)
│   ├── css/style.css        # Ant Design components & CSS3 animations
│   ├── js/theme.js          # Early Light/Dark/System theme switcher
│   ├── js/app.js            # SPA views + Landing Page + OAuth2 consent
│   ├── index.html           # Semantic HTML5 shell
│   └── nginx.conf           # Reverse proxy + strict CSP (style-src 'self')
├── scripts/
│   └── configure_answer_sso.py
└── docker-compose.yml
```

## API Endpoint

- `GET /health` — Status backend & konektivitas/latensi CouchDB (disanitasi)
- `POST /api/register` — Daftar alumni baru
- `POST /api/login` — Login (rate limited + `Cache-Control: no-store`)
- `GET /api/me` — Profil user login
- `POST /api/request-verification` — Minta tautan/OTP verifikasi (email NTUST / SMS / WhatsApp)
- `GET /api/verify/status` — Cek status & masa berlaku token verifikasi
- `POST /api/verify` — Verifikasi OTP + NIM
- `GET /api/alumni` — Daftar alumni terverifikasi (`?q=`, `wilayah=`, `jurusan=`, `sort=`, `page=`, `limit=`)
- `GET /api/alumni/:id` — Detail alumni terverifikasi
- `PUT /api/alumni/:id` / `POST /api/alumni/:id` — Edit profil & upload foto profil (PNG/JPEG/WEBP/GIF, maks 2 MB)
- `GET /api/subgroups` — Daftar subgroup (login)
- `POST /api/subgroups` — Buat subgroup (`super_admin`)
- `PUT /api/subgroups/:id` — Edit subgroup (`super_admin`)
- `DELETE /api/subgroups/:id` — Hapus subgroup (`super_admin`)
- `POST /api/subgroups/:id/admin` — Pilih admin wilayah (`super_admin` atau `regional_admin` subgroup tersebut)
- `POST /api/oauth/authorize` — Terbitkan kode otorisasi OAuth2 SSO (khusus alumni `verified`)
- `POST /api/oauth/token` — Tukar kode otorisasi dengan scoped `oauth_access` JWT
- `GET /api/oauth/userinfo` — Profil alumni terverifikasi untuk Apache Answer `connector-basic`
- `GET /api/forum`, `POST /api/forum` — Endpoint legacy forum internal (terverifikasi; UI dialihkan penuh ke Apache Answer)

## Testing

```bash
cd backend && go test -v -count=1 ./...
node --check frontend/js/theme.js && node --check frontend/js/app.js
```

## License

MIT

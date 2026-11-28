# Alumni NTUST Indonesia

Aplikasi manajemen data dan komunikasi untuk alumni National Taiwan University of Science and Technology (NTUST) di Indonesia.

## Fitur Utama

- **Pendaftaran Alumni** — siapa pun boleh mendaftar dengan email apa pun (status awal `pending`)
- **Verifikasi Email NTUST** — verifikasi menggunakan email `@mail.ntust.edu.tw` (OTP 6 digit via Brevo API) + NIM
- **Direktori Alumni** — cari (`?q=`), filter (`wilayah`, `jurusan`), urutkan (`sort`), dan paginasi profil alumni terverifikasi
- **Subgroup Per Wilayah** — ikatan alumni NTUST per wilayah (misal: Jakarta, Bandung)
- **Penunjukan Admin Wilayah** — tunjuk alumni di wilayah tertentu sebagai `regional_admin`
- **Forum Q&A (Apache Answer + SSO)** — platform Q&A lengkap (pertanyaan, jawaban, accepted answer, voting, tag) terintegrasi lewat OAuth2 SSO untuk alumni terverifikasi

## Tech Stack

- **Backend**: Go 1.22+ + Gin + CouchDB REST client (`net/http`) + OAuth2 Provider
- **Frontend**: Vanilla JS SPA (no build step) + Nginx reverse proxy
- **Forum Q&A**: Apache Answer (`connector-basic` OAuth2 SSO) + SQLite
- **Database**: CouchDB 3.3
- **Email / SMS / WA**: Brevo API v3

## Mulai dengan Docker

```bash
# Salin template .env dan isi seluruh kredensial/secret (file .env di-gitignore)
cp .env.example .env
chmod 600 .env

# Build dan jalankan semua service
docker-compose up -d --build

# Aktifkan & konfigurasikan SSO OAuth2 pada Apache Answer (sekali saat inisialisasi)
python3 scripts/configure_answer_sso.py

# Cek health endpoint
curl http://localhost:8080/health
```

- **Portal Alumni**: [http://localhost:8081](http://localhost:8081)
- **Forum Q&A (Apache Answer)**: [http://localhost:9080](http://localhost:9080)

### Lingkungan (Environment Variables)

Semua variabel sensitif disimpan di file `.env` (jangan pernah di-commit). Lihat [`.env.example`](.env.example).

| Variable | Deskripsi | Default |
|----------|-----------|---------|
| `COUCHDB_USER` | Username CouchDB (**wajib** di `.env`) | (wajib diisi) |
| `COUCHDB_PASSWORD` | Password CouchDB (**wajib** di `.env`) | (wajib diisi) |
| `COUCHDB_URL` | URL koneksi CouchDB (**wajib**) | (wajib diisi) |
| `JWT_SECRET` | Secret untuk signing JWT token (**wajib**, server menolak start jika kosong) | (wajib diisi) |
| `PORT` | Port HTTP backend | `8080` |
| `FRONTEND_URL` | URL base frontend untuk tautan verifikasi email | `http://localhost:8081` |
| `BREVO_API_KEY` | API key Brevo (`xkeysib-...` atau token MCP base64) | (kosong) |
| `BREVO_FROM_EMAIL` | Email pengirim verifikasi | `ntust-alumni-verification@cerdas.club` |
| `BREVO_FROM_NAME` | Nama pengirim | `Alumni NTUST Indonesia` |
| `VERIFICATION_ALLOWED_DOMAINS` | Domain email verifikasi yang diizinkan (dipisah koma) | `@mail.ntust.edu.tw` |

## Development (tanpa Docker)

```bash
cd backend
set -a; source ../.env; set +a
go build ./...
go test ./...
go run ./cmd/server
```

Untuk frontend statis secara lokal, jalankan dari root repo:

```bash
python3 -m http.server 8081 --directory frontend
```

Lalu buka `http://localhost:8081`.

## Struktur Project

```
ntust-alumni/
├── backend/
│   ├── cmd/server/          # Entry point & router
│   ├── internal/db/         # CouchDB REST client
│   ├── internal/models/     # Struct data
│   ├── internal/middleware/ # JWT, verification, role middleware
│   ├── internal/handlers/   # Auth, alumni, subgroup, forum, verification, rate limiter
│   ├── internal/email/      # Brevo API client
│   └── Dockerfile
├── frontend/                # Vanilla JS SPA + nginx.conf
└── docker-compose.yml
```

## API Endpoint

- `GET /health` — Status backend & konektivitas/latensi CouchDB
- `POST /api/register` — Daftar alumni baru
- `POST /api/login` — Login (dengan rate limiting)
- `GET /api/me` — Profil user login
- `POST /api/request-verification` — Minta OTP verifikasi email `@mail.ntust.edu.tw` (login dulu)
- `POST /api/verify` — Verifikasi OTP + NIM (publik, mengembalikan JWT baru jika berhasil)
- `GET /api/alumni` — Daftar alumni terverifikasi (`?q=`, `wilayah=`, `jurusan=`, `sort=`, `page=`, `limit=`)
- `GET /api/alumni/:id` — Detail alumni (terverifikasi)
- `PUT /api/alumni/:id` / `POST /api/alumni/:id` — Edit profil & upload foto profil (maks 2 MB)
- `GET /api/subgroups` — Daftar subgroup (login)
- `POST /api/subgroups` — Buat subgroup (`super_admin`)
- `PUT /api/subgroups/:id` — Edit subgroup (`super_admin`)
- `DELETE /api/subgroups/:id` — Hapus subgroup (`super_admin`)
- `POST /api/subgroups/:id/admin` — Pilih admin wilayah (`super_admin`/`regional_admin`)
- `GET /api/forum` — Daftar postingan forum (publik)
- `POST /api/forum` — Buat postingan forum (terverifikasi)

## Testing

```bash
cd backend && go test ./... -v
node --check frontend/js/app.js
```

## License

MIT

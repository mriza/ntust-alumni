# Alumni NTUST Indonesia

Aplikasi manajemen data dan komunikasi untuk alumni PT Taman Siswa Universitas Negeri Tangerang Selatan (NTUST).

## Fitur Utama

- **Pendaftaran Alumni** — siapa pun boleh mendaftar dengan email apa pun
- **Verifikasi Email** — untuk bergabung di aplikasi, sertakan email `@mail.ntust.edu.tw` + NIM
- **Direktori Alumni** — lihat profil alumni terverifikasi (nama, jurusan, pekerjaan, sosmed, dll.)
- **Subgroup Per Wilayah** — ikatan alumni NTUST per wilayah (misal: Jakarta, Bandung)
- **Penunjukan Admin Wilayah** — tunjuk alumni di wilayah tertentu sebagai regional admin
- **Forum Diskusi** — channel komunikasi sederhana untuk alumni
- **Manajemen Profil** — lengkapi profil: nama, kontak, pekerjaan, pendidikan NTUST, dll.

## Tech Stack

- **Backend**: Go + Gin + CouchDB
- **Frontend**: Vanilla JS SPA (no build step)
- **Database**: CouchDB 3.3
- **Email**: Brevo API v3

## Mulai dengan Docker

```bash
# Pasang Brevo API Key (opsional, untuk verifikasi email)
export BREVO_API_KEY="your-brevo-api-key-here"

# Build dan jalankan semua service
docker-compose up -d

# Cek health endpoint
curl http://localhost:8080/health
```

Aplikasi frontend dibuka di: [http://localhost:80](http://localhost:80)

### Lingkungan (Environment Variables)

| Variable | Deskripsi | Default |
|----------|-----------|---------|
| `COUCHDB_URL` | URL CouchDB | `http://admin:<COUCHDB_PASSWORD>@localhost:5984` |
| `JWT_SECRET` | Secret untuk JWT token | `<JWT_SECRET>` |
| `BREVO_API_KEY` | API key Brevo | (kosong) |
| `BREVO_FROM_EMAIL` | Email pengirim verifikasi | `noreply@ntust-alumni.id` |
| `BREVO_FROM_NAME` | Nama pengirim | `Alumni NTUST Indonesia` |

## Development (tanpa Docker)

```bash
cd backend
go build ./...
go test ./...
./server  # atau: go run ./cmd/server
```

## Struktur Project

```
ntust-alumni/
├── backend/
│   ├── cmd/server/         # Entry point
│   ├── internal/db/        # CouchDB REST client
│   ├── internal/models/    # Struct data
│   ├── internal/middleware/ # JWT, verification, role middleware
│   ├── internal/handlers/  # Auth, alumni, subgroup, forum, verification
│   ├── internal/email/     # Brevo API client
│   └── Dockerfile
├── frontend/               # Vanilla JS SPA
└── docker-compose.yml
```

## API Endpoint

- `POST /api/register` — Daftar alumni baru
- `POST /api/login` — Login
- `GET /api/me` — Profil user login
- `POST /api/request-verification` — Minta verifikasi email (login dulu)
- `POST /api/verify` — Verifikasi OTP + NIM (publik)
- `GET /api/alumni` — Daftar alumni (terverifikasi)
- `GET /api/alumni/:id` — Detail alumni
- `PUT /api/alumni/:id` — Edit profil sendiri
- `GET /api/subgroups` — Daftar subgroup
- `POST /api/subgroups` — Buat subgroup (super_admin)
- `POST /api/subgroups/:id/admin` — Pilih admin wilayah (super_admin/regional_admin)
- `GET /api/forum` — Daftar postingan forum
- `POST /api/forum` — Buat postingan forum (terverifikasi)

## Testing

Semua unit tests: `cd backend && go test ./...`

## License

MIT

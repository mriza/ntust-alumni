package main

import (
	"context"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
	"ntust-alumni/backend/internal/db"
	"ntust-alumni/backend/internal/models"
)

type seedAccount struct {
	Email    string
	Password string
	Profile  models.Alumni
}

func upsertAlumni(ctx context.Context, database db.Store, acc seedAccount) (*models.Alumni, error) {
	existing, err := database.FindAlumniByEmail(ctx, acc.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		log.Printf("Seed alumni sudah ada: %s (id=%s, role=%s)", existing.Email, existing.ID, existing.Role)
		return existing, nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(acc.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	a := acc.Profile
	a.Email = acc.Email
	a.PasswordHash = string(hash)
	if err := database.SaveAlumni(ctx, &a); err != nil {
		return nil, err
	}
	log.Printf("Berhasil seed alumni: %s (id=%s, role=%s)", a.Email, a.ID, a.Role)
	return &a, nil
}

func requireEnv(name string) string {
	val := os.Getenv(name)
	if val == "" {
		log.Fatalf("Environment variable %s wajib diisi sebelum menjalankan seed", name)
	}
	return val
}

func main() {
	couchURL := requireEnv("COUCHDB_URL")
	adminPassword := requireEnv("SEED_ADMIN_PASSWORD")
	regionalPassword := requireEnv("SEED_REGIONAL_ADMIN_PASSWORD")
	pengurusPassword := requireEnv("SEED_PENGURUS_PASSWORD")
	memberPassword := requireEnv("SEED_MEMBER_PASSWORD")

	database, err := db.NewCouchDB(couchURL)
	if err != nil {
		log.Fatalf("Gagal koneksi CouchDB: %v", err)
	}

	ctx := context.Background()
	if err := database.InitDatabase(ctx); err != nil {
		log.Fatalf("Gagal inisialisasi database: %v", err)
	}

	accounts := []seedAccount{
		{
			Email:    "admin@ntust-alumni.id",
			Password: adminPassword,
			Profile: models.Alumni{
				Nama:             "Admin Pusat NTUST",
				Nim:              "M10500001",
				Jurusan:          "Computer Science & Information Engineering",
				Status:           "verified",
				EmailNtust:       "admin@mail.ntust.edu.tw",
				Wilayah:          "Jabodetabek",
				Role:             "super_admin",
				NoTelp:           "08110000001",
				NoWA:             "08110000001",
				BidangKeahianan:  []string{"Architecture", "Governance", "Cloud"},
				PreferensiKontak: "Email",
				StatusKeaktifan:  "aktif",
				Pekerjaan: &models.Pekerjaan{
					Institusi:     "NTUST Alumni Indonesia",
					JenisIndustri: "Organisasi Alumni",
					Jabatan:       "Lead Architect",
				},
				PendidikanNTUST: &models.PendidikanNTUST{
					TahunMasuk: 2016,
					TahunLulus: 2018,
					Fakultas:   "EECS",
					Jurusan:    "CSIE",
				},
			},
		},
		{
			Email:    "admin.jabodetabek@ntust-alumni.id",
			Password: regionalPassword,
			Profile: models.Alumni{
				Nama:             "Budi Pratama",
				Nim:              "M10715001",
				Jurusan:          "Electrical Engineering",
				Status:           "verified",
				EmailNtust:       "budi.pratama@mail.ntust.edu.tw",
				Wilayah:          "Jabodetabek",
				Role:             "regional_admin",
				NoTelp:           "08120000002",
				NoWA:             "08120000002",
				BidangKeahianan:  []string{"Power Systems", "IoT", "Management"},
				PreferensiKontak: "WhatsApp",
				StatusKeaktifan:  "aktif",
				Pekerjaan: &models.Pekerjaan{
					Institusi:     "PT Teknologi Nusantara",
					JenisIndustri: "Teknologi & Energi",
					Jabatan:       "Engineering Manager",
				},
				PendidikanNTUST: &models.PendidikanNTUST{
					TahunMasuk: 2018,
					TahunLulus: 2020,
					Fakultas:   "EECS",
					Jurusan:    "EE",
				},
			},
		},
		{
			Email:    "wakil.jabodetabek@ntust-alumni.id",
			Password: pengurusPassword,
			Profile: models.Alumni{
				Nama:             "Siti Rahmawati",
				Nim:              "M10809002",
				Jurusan:          "Industrial Management",
				Status:           "verified",
				EmailNtust:       "siti.rahmawati@mail.ntust.edu.tw",
				Wilayah:          "Jabodetabek",
				Role:             "member",
				NoTelp:           "08120000003",
				NoWA:             "08120000003",
				BidangKeahianan:  []string{"Supply Chain", "Operations"},
				PreferensiKontak: "WhatsApp",
				StatusKeaktifan:  "aktif",
				Pekerjaan: &models.Pekerjaan{
					Institusi:     "PT Logistik Mandiri",
					JenisIndustri: "Logistik",
					Jabatan:       "Operations Lead",
				},
				PendidikanNTUST: &models.PendidikanNTUST{
					TahunMasuk: 2019,
					TahunLulus: 2021,
					Fakultas:   "Management",
					Jurusan:    "Industrial Management",
				},
			},
		},
		{
			Email:    "sekretaris.jabodetabek@ntust-alumni.id",
			Password: pengurusPassword,
			Profile: models.Alumni{
				Nama:             "Andi Wijaya",
				Nim:              "M10915003",
				Jurusan:          "Computer Science & Information Engineering",
				Status:           "verified",
				EmailNtust:       "andi.wijaya@mail.ntust.edu.tw",
				Wilayah:          "Jabodetabek",
				Role:             "member",
				NoTelp:           "08120000004",
				NoWA:             "08120000004",
				BidangKeahianan:  []string{"Data Engineering", "Python"},
				PreferensiKontak: "Telegram",
				StatusKeaktifan:  "aktif",
				Pekerjaan: &models.Pekerjaan{
					Institusi:     "PT Analitika Data",
					JenisIndustri: "Teknologi Informasi",
					Jabatan:       "Senior Data Engineer",
				},
				PendidikanNTUST: &models.PendidikanNTUST{
					TahunMasuk: 2020,
					TahunLulus: 2022,
					Fakultas:   "EECS",
					Jurusan:    "CSIE",
				},
			},
		},
		{
			Email:    "bendahara.jabodetabek@ntust-alumni.id",
			Password: pengurusPassword,
			Profile: models.Alumni{
				Nama:             "Dewi Lestari",
				Nim:              "M10908004",
				Jurusan:          "Business Administration",
				Status:           "verified",
				EmailNtust:       "dewi.lestari@mail.ntust.edu.tw",
				Wilayah:          "Jabodetabek",
				Role:             "member",
				NoTelp:           "08120000005",
				NoWA:             "08120000005",
				BidangKeahianan:  []string{"Finance", "Corporate Strategy"},
				PreferensiKontak: "Email",
				StatusKeaktifan:  "aktif",
				Pekerjaan: &models.Pekerjaan{
					Institusi:     "PT Finansial Sejahtera",
					JenisIndustri: "Keuangan",
					Jabatan:       "Finance Controller",
				},
				PendidikanNTUST: &models.PendidikanNTUST{
					TahunMasuk: 2020,
					TahunLulus: 2022,
					Fakultas:   "Management",
					Jurusan:    "Business Administration",
				},
			},
		},
		{
			Email:    "humas.jabodetabek@ntust-alumni.id",
			Password: pengurusPassword,
			Profile: models.Alumni{
				Nama:             "Rizky Ramadhan",
				Nim:              "M11005005",
				Jurusan:          "Mechanical Engineering",
				Status:           "verified",
				EmailNtust:       "rizky.ramadhan@mail.ntust.edu.tw",
				Wilayah:          "Jabodetabek",
				Role:             "member",
				NoTelp:           "08120000006",
				NoWA:             "08120000006",
				BidangKeahianan:  []string{"Robotics", "Community Building"},
				PreferensiKontak: "WhatsApp",
				StatusKeaktifan:  "aktif",
				Pekerjaan: &models.Pekerjaan{
					Institusi:     "PT Otomasi Industri",
					JenisIndustri: "Manufaktur",
					Jabatan:       "R&D Specialist",
				},
				PendidikanNTUST: &models.PendidikanNTUST{
					TahunMasuk: 2021,
					TahunLulus: 2023,
					Fakultas:   "Engineering",
					Jurusan:    "Mechanical Engineering",
				},
			},
		},
		{
			Email:    "member@ntust-alumni.id",
			Password: memberPassword,
			Profile: models.Alumni{
				Nama:             "Rina Kusuma",
				Nim:              "M11115010",
				Jurusan:          "Computer Science & Information Engineering",
				Status:           "verified",
				EmailNtust:       "rina.kusuma@mail.ntust.edu.tw",
				Wilayah:          "Jabodetabek",
				Role:             "member",
				NoTelp:           "08120000010",
				NoWA:             "08120000010",
				BidangKeahianan:  []string{"Go", "Backend", "Cloud"},
				PreferensiKontak: "WhatsApp",
				StatusKeaktifan:  "aktif",
				Pekerjaan: &models.Pekerjaan{
					Institusi:     "PT Solusi Digital Indonesia",
					JenisIndustri: "Teknologi Informasi",
					Jabatan:       "Software Engineer",
				},
				PendidikanNTUST: &models.PendidikanNTUST{
					TahunMasuk: 2022,
					TahunLulus: 2024,
					Fakultas:   "EECS",
					Jurusan:    "CSIE",
				},
			},
		},
	}

	seededByEmail := make(map[string]*models.Alumni, len(accounts))
	for _, acc := range accounts {
		saved, err := upsertAlumni(ctx, database, acc)
		if err != nil {
			log.Fatalf("Gagal seed akun %s: %v", acc.Email, err)
		}
		seededByEmail[acc.Email] = saved
	}

	// Seed Subgroup Jabodetabek beserta struktur pengurus wilayah
	subgroups, err := database.ListSubgroups(ctx)
	if err != nil {
		log.Fatalf("Gagal mengambil daftar subgroup: %v", err)
	}
	var jaboSubgroup *models.Subgroup
	for i := range subgroups {
		if subgroups[i].Wilayah == "Jabodetabek" {
			jaboSubgroup = &subgroups[i]
			break
		}
	}

	ketua := seededByEmail["admin.jabodetabek@ntust-alumni.id"]
	wakil := seededByEmail["wakil.jabodetabek@ntust-alumni.id"]
	sekretaris := seededByEmail["sekretaris.jabodetabek@ntust-alumni.id"]
	bendahara := seededByEmail["bendahara.jabodetabek@ntust-alumni.id"]
	humas := seededByEmail["humas.jabodetabek@ntust-alumni.id"]

	pengurus := []models.PengurusWilayah{
		{Jabatan: "Ketua Wilayah (Admin Wilayah)", Nama: ketua.Nama, AlumniID: ketua.ID},
		{Jabatan: "Wakil Ketua", Nama: wakil.Nama, AlumniID: wakil.ID},
		{Jabatan: "Sekretaris", Nama: sekretaris.Nama, AlumniID: sekretaris.ID},
		{Jabatan: "Bendahara", Nama: bendahara.Nama, AlumniID: bendahara.ID},
		{Jabatan: "Koordinator Acara & Humas", Nama: humas.Nama, AlumniID: humas.ID},
	}

	if jaboSubgroup == nil {
		jaboSubgroup = &models.Subgroup{
			Nama:      "Ikatan Alumni NTUST Wilayah Jabodetabek",
			Wilayah:   "Jabodetabek",
			Deskripsi: "Wadah komunikasi, kolaborasi, dan kegiatan alumni NTUST di wilayah Jakarta, Bogor, Depok, Tangerang, dan Bekasi.",
			AdminID:   ketua.ID,
			Pengurus:  pengurus,
		}
		if err := database.SaveSubgroup(ctx, jaboSubgroup); err != nil {
			log.Fatalf("Gagal membuat subgroup Jabodetabek: %v", err)
		}
		log.Printf("Berhasil seed subgroup Jabodetabek (id=%s)", jaboSubgroup.ID)
	} else {
		log.Printf("Subgroup Jabodetabek sudah ada (id=%s)", jaboSubgroup.ID)
	}

	// Tautkan seluruh alumni wilayah Jabodetabek ke subgroup_id jika belum terisi
	for _, a := range seededByEmail {
		if a.Wilayah == "Jabodetabek" && a.SubgroupID != jaboSubgroup.ID {
			a.SubgroupID = jaboSubgroup.ID
			if err := database.UpdateAlumni(ctx, a); err != nil {
				log.Fatalf("Gagal menautkan subgroup_id untuk %s: %v", a.Email, err)
			}
		}
	}

	// Seed contoh postingan forum jika kosong
	posts, err := database.ListForumPosts(ctx)
	if err != nil {
		log.Fatalf("Gagal memeriksa postingan forum: %v", err)
	}
	if len(posts) == 0 {
		admin := seededByEmail["admin@ntust-alumni.id"]
		post := &models.ForumPost{
			Judul:       "Selamat Datang di Portal Alumni NTUST Indonesia",
			Isi:         "Halo rekan-rekan alumni NTUST! Silakan lengkapi profil, perbarui data pekerjaan, dan gunakan direktori serta forum ini untuk berkolaborasi lintas angkatan dan wilayah.",
			PenulisID:   admin.ID,
			PenulisNama: admin.Nama,
			Wilayah:     admin.Wilayah,
		}
		if err := database.SaveForumPost(ctx, post); err != nil {
			log.Fatalf("Gagal seed forum post: %v", err)
		}
		log.Printf("Berhasil seed contoh forum post (id=%s)", post.ID)
	} else {
		log.Printf("Forum posts sudah terisi (%d post), skip seed forum", len(posts))
	}

	log.Println("Seed selesai.")
}

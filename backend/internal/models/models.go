package models

import "time"

// Sosmed struct untuk akun media sosial alumni
type Sosmed struct {
	Facebook string `json:"facebook,omitempty"`
	Instagram string `json:"instagram,omitempty"`
	Twitter  string `json:"twitter,omitempty"`
	Threads  string `json:"threads,omitempty"`
}

// Pekerjaan struct untuk data pekerjaan alumni
type Pekerjaan struct {
	Institusi     string `json:"institusi,omitempty"`
	JenisIndustri string `json:"jenis_industri,omitempty"`
	Jabatan       string `json:"jabatan,omitempty"`
	AlamatKantor  string `json:"alamat_kantor,omitempty"`
	TelpKantor    string `json:"telp_kantor,omitempty"`
	EmailKerja    string `json:"email_kerja,omitempty"`
}

// PendidikanNTUST struct untuk data pendidikan di NTUST
type PendidikanNTUST struct {
	TahunMasuk int    `json:"tahun_masuk,omitempty"`
	TahunLulus int    `json:"tahun_lulus_studi,omitempty"`
	Supervisor string `json:"supervisor,omitempty"`
	Fakultas   string `json:"fakultas,omitempty"`
}

// Alumni mewakili data anggota alumni NTUST
type Alumni struct {
	ID              string       `json:"id,omitempty"`
	Rev             string       `json:"rev,omitempty"`
	Nama            string       `json:"nama"`
	Email           string       `json:"email"`
	PasswordHash    string       `json:"-"`
	Nim             string       `json:"nim"`
	Jurusan         string       `json:"jurusan,omitempty"`
	Status          string       `json:"status"` // pending, verified
	EmailNtust      string       `json:"email_ntust,omitempty"`

	// Kontak pribadi (dapat diedit oleh alumni)
	NoTelp          string       `json:"no_telp,omitempty"`
	NoWA            string       `json:"no_wa,omitempty"`
	Telegram        string       `json:"telegram,omitempty"`
	Sosmed          *Sosmed      `json:"sosmed,omitempty"`
	FotoProfil      string       `json:"foto_profil,omitempty"`
	LinkedIn        string       `json:"linkedin,omitempty"`
	BidangKeahianan []string     `json:"bidang_keahlian,omitempty"`
	PreferensiKontak string      `json:"preferensi_kontak,omitempty"`
	StatusKeaktifan  string      `json:"status_keaktifan,omitempty"` // aktif, tidak aktif

	// Pekerjaan (dapat diedit oleh alumni)
	Pekerjaan *Pekerjaan `json:"pekerjaan,omitempty"`

	// Pendidikan di NTUST (dapat diedit oleh alumni)
	PendidikanNTUST *PendidikanNTUST `json:"pendidikan_ntust,omitempty"`

	// Wilayah & Role
	Wilayah     string `json:"wilayah,omitempty"`
	SubgroupID  string `json:"subgroup_id,omitempty"`
	Role        string `json:"role"` // member, regional_admin, super_admin

	// OTP & verifikasi (private)
	OtpHash          string    `json:"-"`
	OtpExpiresAt     time.Time `json:"-"`
	VerificationToken string   `json:"-"`
	TokenExpiresAt   time.Time `json:"-"`

	DibuatPada  time.Time `json:"dibuat_pada,omitempty"`
	Diperbarui  time.Time `json:"diperbarui,omitempty"`
}

// Subgroup mewakili ikatan alumni per wilayah
type Subgroup struct {
	ID          string    `json:"id,omitempty" db:"_id,omitempty"`
	Rev         string    `json:"rev,omitempty" db:"_rev,omitempty"`
	Nama        string    `json:"nama"`        // misal: IKATAN ALUMNI NTUST WILAYAH JAKARTA
	Wilayah     string    `json:"wilayah"`     // misal: Jakarta, Bandung, Surabaya
	Deskripsi   string    `json:"deskripsi"`
	AdminID     string    `json:"admin_id"`    // ID alumni yang menjadi admin wilayah
	DibuatPada  time.Time `json:"dibuat_pada,omitempty"`
	Diperbarui  time.Time `json:"diperbarui,omitempty"`
}

// ForumPost mewakili postingan di channel forum
type ForumPost struct {
	ID          string    `json:"id,omitempty" db:"_id,omitempty"`
	Rev         string    `json:"rev,omitempty" db:"_rev,omitempty"`
	Judul       string    `json:"judul"`
	Isi         string    `json:"isi"`
	PenulisID   string    `json:"penulis_id"`
	PenulisNama string    `json:"penulis_nama"`
	Wilayah     string    `json:"wilayah,omitempty"`
	DibuatPada  time.Time `json:"dibuat_pada,omitempty"`
	Diperbarui  time.Time `json:"diperbarui,omitempty"`
}

// AuthRequest untuk login/register
type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterRequest untuk pendaftaran alumni baru
type RegisterRequest struct {
	Nama       string `json:"nama"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	Nim        string `json:"nim"`
	Jurusan    string `json:"jurusan"`
	TahunLulus int    `json:"tahun_lulus"`
	Wilayah    string `json:"wilayah"`
	Telepon    string `json:"telepon"`
}

// SubgroupRequest untuk membuat/mengedit subgroup
type SubgroupRequest struct {
	Nama       string `json:"nama"`
	Wilayah    string `json:"wilayah"`
	Deskripsi  string `json:"deskripsi"`
	AdminID    string `json:"admin_id"`
}

// ForumPostRequest untuk membuat postingan forum
type ForumPostRequest struct {
	Judul   string `json:"judul"`
	Isi     string `json:"isi"`
	Wilayah string `json:"wilayah,omitempty"`
}

// TokenResponse untuk response login
type TokenResponse struct {
	Token string `json:"token"`
	Role  string `json:"role"`
	ID    string `json:"id"`
	Nama  string `json:"nama"`
}

// ErrorResponse untuk response error
type ErrorResponse struct {
	Error string `json:"error"`
}

// SuccessResponse untuk response sukses
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

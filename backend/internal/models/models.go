package models

import "time"

// Alumni mewakili data anggota alumni NTUST
type Alumni struct {
	ID          string    `json:"id,omitempty" db:"_id,omitempty"`
	Rev         string    `json:"rev,omitempty" db:"_rev,omitempty"`
	Nama        string    `json:"nama"`
	Email       string    `json:"email"`
	Nim         string    `json:"nim"`
	Jurusan     string    `json:"jurusan"`
	TahunLulus  int       `json:"tahun_lulus"`
	Wilayah     string    `json:"wilayah"`
	Telepon     string    `json:"telepon"`
	Role        string    `json:"role"` // member, regional_admin, super_admin
	SubgroupID  string    `json:"subgroup_id,omitempty"`
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

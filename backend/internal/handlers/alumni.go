package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"ntust-alumni/backend/internal/db"
	"ntust-alumni/backend/internal/models"
)

type AlumniHandler struct {
	database db.Store
}

func NewAlumniHandler(database db.Store) *AlumniHandler {
	return &AlumniHandler{database: database}
}

// ListAlumni mengembalikan daftar alumni VERIFIED (direktori) dengan search & filter
func (h *AlumniHandler) ListAlumni(c *gin.Context) {
	// Parsing query params: q=search, wilayah, jurusan, page, limit
	query := c.Query("q")
	wilayah := c.Query("wilayah")
	jurusan := c.Query("jurusan")
	pageStr := c.Query("page")
	limitStr := c.Query("limit")

	page := 1
	if pageStr != "" {
		page, _ = strconv.Atoi(pageStr)
		if page < 1 {
			page = 1
		}
	}
	limit := 50
	if limitStr != "" {
		limit, _ = strconv.Atoi(limitStr)
		if limit <= 0 || limit > 200 {
			limit = 50
		}
	}
	offset := (page - 1) * limit

	alumni, err := h.database.ListAlumni(c.Request.Context(), query, wilayah, jurusan, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal mengambil data alumni"})
		return
	}

	var result []AlumniPublic
	for _, a := range alumni {
		if a.Status != "verified" {
			continue
		}
		result = append(result, AlumniPublicFromAlumni(&a))
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Data alumni berhasil diambil",
		Data: gin.H{
			"alumni": result,
			"page":   page,
			"limit":  limit,
		},
	})
}

// AlumniPublic adalah representasi public dari alumni yang ditampilkan ke publik
type AlumniPublic struct {
	ID              string         `json:"id"`
	Nama            string         `json:"nama"`
	Email           string         `json:"email"`
	Nim             string         `json:"nim,omitempty"`
	Status          string         `json:"status"`
	Wilayah         string         `json:"wilayah,omitempty"`
	SubgroupID      string         `json:"subgroup_id,omitempty"`
	Role            string         `json:"role"`
	NoTelp          string         `json:"no_telp,omitempty"`
	NoWA            string         `json:"no_wa,omitempty"`
	Telegram        string         `json:"telegram,omitempty"`
	Sosmed          *models.Sosmed `json:"sosmed,omitempty"`
	FotoProfil      string         `json:"foto_profil,omitempty"`
	LinkedIn        string         `json:"linkedin,omitempty"`
	BidangKeahianan []string       `json:"bidang_keahlian,omitempty"`
	PreferensiKontak string        `json:"preferensi_kontak,omitempty"`
	StatusKeaktifan  string        `json:"status_keaktifan,omitempty"`
	Pekerjaan       *models.Pekerjaan `json:"pekerjaan,omitempty"`
	PendidikanNTUST *models.PendidikanNTUST `json:"pendidikan_ntust,omitempty"`
	DibuatPada      time.Time      `json:"dibuat_pada,omitempty"`
}

func AlumniPublicFromAlumni(a *models.Alumni) AlumniPublic {
	return AlumniPublic{
		ID:               a.ID,
		Nama:             a.Nama,
		Email:            a.Email,
		Nim:              a.Nim,
		Status:           a.Status,
		Wilayah:          a.Wilayah,
		SubgroupID:       a.SubgroupID,
		Role:             a.Role,
		NoTelp:           a.NoTelp,
		NoWA:             a.NoWA,
		Telegram:         a.Telegram,
		Sosmed:           a.Sosmed,
		FotoProfil:       a.FotoProfil,
		LinkedIn:         a.LinkedIn,
		BidangKeahianan:  a.BidangKeahianan,
		PreferensiKontak: a.PreferensiKontak,
		StatusKeaktifan:  a.StatusKeaktifan,
		Pekerjaan:        a.Pekerjaan,
		PendidikanNTUST:  a.PendidikanNTUST,
		DibuatPada:       a.DibuatPada,
	}
}

// GetAlumniByID mengembalikan detail satu alumni
func (h *AlumniHandler) GetAlumniByID(c *gin.Context) {
	id := c.Param("id")
	alumni, err := h.database.GetAlumni(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Alumni tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Data alumni ditemukan",
		Data:    AlumniPublicFromAlumni(alumni),
	})
}

// UpdateAlumniRequest adalah body request untuk PUT /alumni/:id
type UpdateAlumniRequest struct {
	Nama             string                    `json:"nama,omitempty"`
	NoTelp           string                    `json:"no_telp,omitempty"`
	NoWA             string                    `json:"no_wa,omitempty"`
	Telegram         string                    `json:"telegram,omitempty"`
	Sosmed           *models.Sosmed            `json:"sosmed,omitempty"`
	FotoProfil       string                    `json:"foto_profil,omitempty"`
	LinkedIn         string                    `json:"linkedin,omitempty"`
	BidangKeahianan  []string                  `json:"bidang_keahlian,omitempty"`
	PreferensiKontak string                    `json:"preferensi_kontak,omitempty"`
	StatusKeaktifan  string                    `json:"status_keaktifan,omitempty"`
	Pekerjaan        *models.Pekerjaan         `json:"pekerjaan,omitempty"`
	PendidikanNTUST  *models.PendidikanNTUST   `json:"pendidikan_ntust,omitempty"`
	Wilayah          string                    `json:"wilayah,omitempty"`
}

// UpdateAlumni memperbarui profil alumni (sendiri atau admin)
func (h *AlumniHandler) UpdateAlumni(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")

	// Hanya sendiri atau super_admin/regional_admin yang boleh edit
	if userID.(string) != id && userRole.(string) != "super_admin" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Tidak boleh mengedit data orang lain"})
		return
	}

	var req UpdateAlumniRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Data tidak valid: " + err.Error()})
		return
	}

	alumni, err := h.database.GetAlumni(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Alumni tidak ditemukan"})
		return
	}

	// Update semua field yang di-request
	if req.Nama != "" {
		alumni.Nama = req.Nama
	}
	if req.NoTelp != "" {
		alumni.NoTelp = req.NoTelp
	}
	if req.NoWA != "" {
		alumni.NoWA = req.NoWA
	}
	if req.Telegram != "" {
		alumni.Telegram = req.Telegram
	}
	if req.Sosmed != nil {
		alumni.Sosmed = req.Sosmed
	}
	if req.FotoProfil != "" {
		alumni.FotoProfil = req.FotoProfil
	}
	if req.LinkedIn != "" {
		alumni.LinkedIn = req.LinkedIn
	}
	if req.BidangKeahianan != nil {
		alumni.BidangKeahianan = req.BidangKeahianan
	}
	if req.PreferensiKontak != "" {
		alumni.PreferensiKontak = req.PreferensiKontak
	}
	if req.StatusKeaktifan != "" {
		alumni.StatusKeaktifan = req.StatusKeaktifan
	}
	if req.Pekerjaan != nil {
		alumni.Pekerjaan = req.Pekerjaan
	}
	if req.PendidikanNTUST != nil {
		alumni.PendidikanNTUST = req.PendidikanNTUST
	}
	if req.Wilayah != "" {
		alumni.Wilayah = req.Wilayah
	}

	if err := h.database.UpdateAlumni(c.Request.Context(), alumni); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal memperbarui data"})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Profil alumni berhasil diperbarui",
		Data:    AlumniPublicFromAlumni(alumni),
	})
}

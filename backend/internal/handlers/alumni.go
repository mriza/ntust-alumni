package handlers

import (
	"net/http"
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

// ListAlumni mengembalikan daftar semua alumni
func (h *AlumniHandler) ListAlumni(c *gin.Context) {
	alumni, err := h.database.ListAlumni(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal mengambil data alumni"})
		return
	}

	// Hilangkan password dari response
	type AlumniPublic struct {
		ID          string    `json:"id"`
		Nama        string    `json:"nama"`
		Email       string    `json:"email"`
		Nim         string    `json:"nim"`
		Jurusan     string    `json:"jurusan"`
		TahunLulus  int       `json:"tahun_lulus"`
		Wilayah     string    `json:"wilayah"`
		Telepon     string    `json:"telepon"`
		Role        string    `json:"role"`
		SubgroupID  string    `json:"subgroup_id"`
		DibuatPada  time.Time `json:"dibuat_pada"`
	}

	var result []AlumniPublic
	for _, a := range alumni {
		result = append(result, AlumniPublic{
			ID:          a.ID,
			Nama:        a.Nama,
			Email:       a.Email,
			Nim:         a.Nim,
			Jurusan:     a.Jurusan,
			TahunLulus:  a.TahunLulus,
			Wilayah:     a.Wilayah,
			Telepon:     a.Telepon,
			Role:        a.Role,
			SubgroupID:  a.SubgroupID,
			DibuatPada:  a.DibuatPada,
		})
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Data alumni berhasil diambil",
		Data:    result,
	})
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
		Data: gin.H{
			"id":           alumni.ID,
			"nama":         alumni.Nama,
			"email":        alumni.Email,
			"nim":          alumni.Nim,
			"jurusan":      alumni.Jurusan,
			"tahun_lulus":  alumni.TahunLulus,
			"wilayah":      alumni.Wilayah,
			"telepon":      alumni.Telepon,
			"role":         alumni.Role,
			"subgroup_id":  alumni.SubgroupID,
		},
	})
}

// UpdateAlumni memperbarui data alumni (self atau admin)
func (h *AlumniHandler) UpdateAlumni(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")

	// Hanya sendiri atau super_admin/regional_admin yang boleh edit
	if userID.(string) != id && userRole.(string) != "super_admin" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Tidak boleh mengedit data orang lain"})
		return
	}

	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Data tidak valid"})
		return
	}

	alumni, err := h.database.GetAlumni(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Alumni tidak ditemukan"})
		return
	}

	alumni.Nama = req.Nama
	alumni.Nim = req.Nim
	alumni.Jurusan = req.Jurusan
	alumni.TahunLulus = req.TahunLulus
	alumni.Wilayah = req.Wilayah
	alumni.Telepon = req.Telepon

	if err := h.database.UpdateAlumni(c.Request.Context(), alumni); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal memperbarui data"})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Data alumni berhasil diperbarui"})
}

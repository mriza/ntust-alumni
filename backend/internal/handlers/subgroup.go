package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"ntust-alumni/backend/internal/db"
	"ntust-alumni/backend/internal/models"
)

type SubgroupHandler struct {
	database db.Store
}

func NewSubgroupHandler(database db.Store) *SubgroupHandler {
	return &SubgroupHandler{database: database}
}

// ListSubgroups mengembalikan daftar semua subgroup (ikatan alumni per wilayah)
func (h *SubgroupHandler) ListSubgroups(c *gin.Context) {
	subgroups, err := h.database.ListSubgroups(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal mengambil data subgroup"})
		return
	}
	if subgroups == nil {
		subgroups = []models.Subgroup{}
	}
	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Data subgroup berhasil diambil",
		Data:    subgroups,
	})
}

// CreateSubgroup membuat subgroup baru (hanya super_admin)
func (h *SubgroupHandler) CreateSubgroup(c *gin.Context) {
	var req models.SubgroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Data tidak valid"})
		return
	}

	req.Nama = strings.TrimSpace(req.Nama)
	req.Wilayah = strings.TrimSpace(req.Wilayah)
	if req.Nama == "" || req.Wilayah == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Nama subgroup dan wilayah wajib diisi"})
		return
	}

	subgroup := &models.Subgroup{
		Nama:      req.Nama,
		Wilayah:   req.Wilayah,
		Deskripsi: req.Deskripsi,
		AdminID:   req.AdminID,
		Pengurus:  req.Pengurus,
	}

	if err := h.database.SaveSubgroup(c.Request.Context(), subgroup); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal membuat subgroup"})
		return
	}

	c.JSON(http.StatusCreated, models.SuccessResponse{
		Message: "Subgroup berhasil dibuat",
		Data:    subgroup,
	})
}

// UpdateSubgroup memperbarui subgroup (hanya super_admin)
func (h *SubgroupHandler) UpdateSubgroup(c *gin.Context) {
	id := c.Param("id")
	var req models.SubgroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Data tidak valid"})
		return
	}

	subgroup, err := h.database.GetSubgroup(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Subgroup tidak ditemukan"})
		return
	}

	subgroup.Nama = req.Nama
	subgroup.Wilayah = req.Wilayah
	subgroup.Deskripsi = req.Deskripsi
	subgroup.AdminID = req.AdminID
	if req.Pengurus != nil {
		subgroup.Pengurus = req.Pengurus
	}

	if err := h.database.UpdateSubgroup(c.Request.Context(), subgroup); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal memperbarui subgroup"})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Subgroup berhasil diperbarui",
		Data:    subgroup,
	})
}

// DeleteSubgroup menghapus subgroup (hanya super_admin)
func (h *SubgroupHandler) DeleteSubgroup(c *gin.Context) {
	id := c.Param("id")
	subgroup, err := h.database.GetSubgroup(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Subgroup tidak ditemukan"})
		return
	}

	if err := h.database.DeleteSubgroup(c.Request.Context(), id, subgroup.Rev); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal menghapus subgroup"})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Subgroup berhasil dihapus"})
}

// AppointAdmin menunjuk admin di wilayah tertentu
func (h *SubgroupHandler) AppointAdmin(c *gin.Context) {
	subgroupID := c.Param("id")
	var req struct {
		AlumniID string `json:"alumni_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Data tidak valid"})
		return
	}

	subgroup, err := h.database.GetSubgroup(c.Request.Context(), subgroupID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Subgroup tidak ditemukan"})
		return
	}

	alumni, err := h.database.GetAlumni(c.Request.Context(), req.AlumniID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Alumni tidak ditemukan"})
		return
	}

	// Set alumni sebagai regional_admin dan hubungkan ke subgroup
	alumni.Role = "regional_admin"
	alumni.SubgroupID = subgroupID
	if err := h.database.UpdateAlumni(c.Request.Context(), alumni); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal mengupdate alumni"})
		return
	}

	subgroup.AdminID = req.AlumniID
	if err := h.database.UpdateSubgroup(c.Request.Context(), subgroup); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal mengupdate subgroup"})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Admin wilayah berhasil ditunjuk",
		Data: gin.H{
			"subgroup_id": subgroup.ID,
			"admin_id":    alumni.ID,
			"admin_nama":  alumni.Nama,
			"wilayah":     subgroup.Wilayah,
		},
	})
}

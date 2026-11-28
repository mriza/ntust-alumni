package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"ntust-alumni/backend/internal/db"
	"ntust-alumni/backend/internal/models"
)

type ForumHandler struct {
	database db.Store
}

func NewForumHandler(database db.Store) *ForumHandler {
	return &ForumHandler{database: database}
}

// ListForumPosts mengembalikan daftar postingan forum (terbaru dulu)
func (h *ForumHandler) ListForumPosts(c *gin.Context) {
	posts, err := h.database.ListForumPosts(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal mengambil postingan forum"})
		return
	}
	if posts == nil {
		posts = []models.ForumPost{}
	}
	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Data forum berhasil diambil",
		Data:    posts,
	})
}

// CreateForumPost membuat postingan forum baru (hanya alumni terverifikasi)
func (h *ForumHandler) CreateForumPost(c *gin.Context) {
	userIDVal, _ := c.Get("user_id")
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Sesi tidak valid"})
		return
	}

	var req models.ForumPostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Data tidak valid"})
		return
	}

	req.Judul = strings.TrimSpace(req.Judul)
	req.Isi = strings.TrimSpace(req.Isi)
	if req.Judul == "" || req.Isi == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Judul dan isi postingan wajib diisi"})
		return
	}

	alumni, err := h.database.GetAlumni(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Alumni tidak ditemukan"})
		return
	}

	if alumni.Status != "verified" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Akun belum terverifikasi. Silakan verifikasi email dan NIM terlebih dahulu"})
		return
	}

	post := &models.ForumPost{
		Judul:       req.Judul,
		Isi:         req.Isi,
		PenulisID:   alumni.ID,
		PenulisNama: alumni.Nama,
		Wilayah:     alumni.Wilayah,
	}

	if err := h.database.SaveForumPost(c.Request.Context(), post); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Gagal membuat postingan"})
		return
	}

	c.JSON(http.StatusCreated, models.SuccessResponse{
		Message: "Postingan forum berhasil dibuat",
		Data:    post,
	})
}

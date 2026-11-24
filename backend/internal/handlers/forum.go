package handlers

import (
	"net/http"

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
	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Data forum berhasil diambil",
		Data:    posts,
	})
}

// CreateForumPost membuat postingan forum baru
func (h *ForumHandler) CreateForumPost(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userNama, _ := c.Get("user_nama")
	userWilayah, _ := c.Get("user_wilayah")

	var req models.ForumPostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Data tidak valid"})
		return
	}

	post := &models.ForumPost{
		Judul:       req.Judul,
		Isi:         req.Isi,
		PenulisID:   userID.(string),
		PenulisNama: userNama.(string),
		Wilayah:     userWilayah.(string),
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

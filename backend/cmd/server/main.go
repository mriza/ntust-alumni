package main

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"ntust-alumni/backend/internal/db"
	"ntust-alumni/backend/internal/email"
	"ntust-alumni/backend/internal/handlers"
	"ntust-alumni/backend/internal/middleware"
)

// validateConfig validates environment variables before starting the server.
func validateConfig() error {
	if os.Getenv("JWT_SECRET") == "" {
		return errors.New("JWT_SECRET environment variable is required; refusing to start with insecure default")
	}
	return nil
}

func SetupRouter() *gin.Engine {
	router := gin.Default()

	// Security headers
	router.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'")
		c.Next()
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	return router
}

func main() {
	gin.SetMode(gin.DebugMode)

	// Validasi konfigurasi sebelum mulai
	if err := validateConfig(); err != nil {
		log.Fatal(err)
	}

	couchURL := os.Getenv("COUCHDB_URL")
	if couchURL == "" {
		couchURL = "http://admin:<COUCHDB_PASSWORD>@localhost:5984"
	}

	jwtSecret := os.Getenv("JWT_SECRET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Koneksi ke CouchDB
	database, err := db.NewCouchDB(couchURL)
	if err != nil {
		log.Fatalf("Gagal koneksi CouchDB: %v", err)
	}

	if err := database.InitDatabase(context.Background()); err != nil {
		log.Fatalf("Gagal inisialisasi database: %v", err)
	}

	// Setup JWT
	middleware.InitJWT(jwtSecret)
	handlers.SetJWTKey([]byte(jwtSecret))

	// Inisialisasi handler
	authHandler := handlers.NewAuthHandler(database)
	alumniHandler := handlers.NewAlumniHandler(database)
	subgroupHandler := handlers.NewSubgroupHandler(database)
	forumHandler := handlers.NewForumHandler(database)

	// Email client (Brevo)
	brevoClient := email.NewBrevoClient(
		os.Getenv("BREVO_API_KEY"),
		os.Getenv("BREVO_FROM_EMAIL"),
		os.Getenv("BREVO_FROM_NAME"),
	)
	verificationHandler := handlers.NewVerificationHandler(database, brevoClient)

	router := SetupRouter()

	// Public routes
	public := router.Group("/api")
	{
		public.POST("/register", authHandler.Register)
		public.POST("/login", authHandler.Login)
		public.POST("/verify", verificationHandler.Verify)
		public.GET("/forum", forumHandler.ListForumPosts)
	}

	// Verified alumni routes (hanya alumni terverifikasi)
	verified := router.Group("/api")
	verified.Use(middleware.AuthMiddleware())
	verified.Use(middleware.RequireVerified())
	{
		verified.GET("/alumni", alumniHandler.ListAlumni)
		verified.GET("/alumni/:id", alumniHandler.GetAlumniByID)
	}

	// Protected routes (perlu login)
	protected := router.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/me", authHandler.Me)
		protected.PUT("/alumni/:id", alumniHandler.UpdateAlumni)
		protected.POST("/forum", forumHandler.CreateForumPost)
		protected.POST("/request-verification", verificationHandler.RequestVerification)
	}

	// Admin routes (super_admin dan regional_admin)
	admin := router.Group("/api")
	admin.Use(middleware.AuthMiddleware())
	admin.Use(middleware.RequireRole("super_admin", "regional_admin"))
	{
		admin.GET("/subgroups", subgroupHandler.ListSubgroups)
		admin.POST("/subgroups", subgroupHandler.CreateSubgroup)
		admin.PUT("/subgroups/:id", subgroupHandler.UpdateSubgroup)
		admin.DELETE("/subgroups/:id", subgroupHandler.DeleteSubgroup)
		admin.POST("/subgroups/:id/admin", subgroupHandler.AppointAdmin)
	}

	// Super admin only
	superAdmin := router.Group("/api")
	superAdmin.Use(middleware.AuthMiddleware())
	superAdmin.Use(middleware.RequireRole("super_admin"))
	{
		// Super admin bisa managing semua
	}

	log.Printf("Server berjalan di port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Gagal menjalankan server: %v", err)
	}
}

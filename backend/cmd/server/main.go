package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

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
	if os.Getenv("COUCHDB_URL") == "" {
		return errors.New("COUCHDB_URL environment variable is required; refusing to start with hardcoded credentials")
	}
	return nil
}

// Package-level store untuk health check
var dbStore db.Store

func SetupRouter() *gin.Engine {
	router := gin.Default()

	// CORS for local static/file frontend access
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

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
		start := time.Now()
		if dbStore != nil {
			if err := dbStore.Ping(); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"status":  "error",
					"couchdb": gin.H{"status": "error", "detail": err.Error()},
				})
				return
			}
		}
		c.JSON(200, gin.H{
			"status":  "ok",
			"couchdb": gin.H{"status": "ok", "latency_ms": time.Since(start).Milliseconds()},
		})
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
	dbStore = database

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

	// Email, SMS & WhatsApp client (Brevo)
	brevoClient := email.NewBrevoClient(
		os.Getenv("BREVO_API_KEY"),
		os.Getenv("BREVO_FROM_EMAIL"),
		os.Getenv("BREVO_FROM_NAME"),
	)
	var waTemplateID int64
	if rawTID := os.Getenv("BREVO_WHATSAPP_TEMPLATE_ID"); rawTID != "" {
		waTemplateID, _ = strconv.ParseInt(rawTID, 10, 64)
	}
	brevoClient.ConfigureMessaging(
		os.Getenv("BREVO_SMS_SENDER"),
		os.Getenv("BREVO_WHATSAPP_SENDER"),
		waTemplateID,
	)
	verificationHandler := handlers.NewVerificationHandler(database, brevoClient)

	router := SetupRouter()

	// Public routes
	public := router.Group("/api")
	{
		public.POST("/register", authHandler.Register)
		public.POST("/login", authHandler.Login)
		public.GET("/verify/status", verificationHandler.VerificationStatus)
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
		protected.POST("/alumni/:id", alumniHandler.UpdateAlumni)
		protected.POST("/forum", forumHandler.CreateForumPost)
		protected.POST("/request-verification", verificationHandler.RequestVerification)
		protected.GET("/subgroups", subgroupHandler.ListSubgroups)
	}

	// Admin routes (super_admin dan regional_admin)
	admin := router.Group("/api")
	admin.Use(middleware.AuthMiddleware())
	admin.Use(middleware.RequireRole("super_admin", "regional_admin"))
	{
		admin.POST("/subgroups/:id/admin", subgroupHandler.AppointAdmin)
	}

	// Super admin only
	superAdmin := router.Group("/api")
	superAdmin.Use(middleware.AuthMiddleware())
	superAdmin.Use(middleware.RequireRole("super_admin"))
	{
		superAdmin.POST("/subgroups", subgroupHandler.CreateSubgroup)
		superAdmin.PUT("/subgroups/:id", subgroupHandler.UpdateSubgroup)
		superAdmin.DELETE("/subgroups/:id", subgroupHandler.DeleteSubgroup)
	}

	log.Printf("Server berjalan di port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Gagal menjalankan server: %v", err)
	}
}

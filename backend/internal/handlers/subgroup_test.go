package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"ntust-alumni/backend/internal/middleware"
	"ntust-alumni/backend/internal/models"
)

func TestCreateSubgroupRejectsNonAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	middleware.InitJWT("test-secret")
	SetJWTKey([]byte("test-secret"))

	h := NewSubgroupHandler(database)

	router := gin.New()
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	protected.Use(middleware.RequireRole("super_admin", "regional_admin"))
	protected.POST("/subgroups", h.CreateSubgroup)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "u1",
		"role":    "member",
		"status":  "verified",
	})
	tokenString, _ := token.SignedString([]byte("test-secret"))

	body := `{"nama":"IKATAN JAKARTA","wilayah":"Jakarta","deskripsi":"Test","admin_id":""}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/subgroups", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateSubgroupAsSuperAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	middleware.InitJWT("test-secret")
	SetJWTKey([]byte("test-secret"))

	// Create a super_admin alumni
	database.SaveAlumni(context.Background(), &models.Alumni{
		ID:           "admin1",
		Nama:         "Super Admin",
		Email:        "admin@example.com",
		Role:         "super_admin",
		Status:       "verified",
		PasswordHash: "hash",
	})

	h := NewSubgroupHandler(database)

	router := gin.New()
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	protected.Use(middleware.RequireRole("super_admin", "regional_admin"))
	protected.POST("/subgroups", h.CreateSubgroup)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "admin1",
		"role":    "super_admin",
		"status":  "verified",
	})
	tokenString, _ := token.SignedString([]byte("test-secret"))

	body := `{"nama":"IKATAN ALUMNI NTUST WILAYAH JAKARTA","wilayah":"Jakarta","deskripsi":"Ikatan alumni di Jakarta","admin_id":"admin1"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/subgroups", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify subgroup was saved
	subgroups, _ := database.ListSubgroups(context.Background())
	if len(subgroups) != 1 {
		t.Fatalf("expected 1 subgroup, got %d", len(subgroups))
	}
	if subgroups[0].Nama != "IKATAN ALUMNI NTUST WILAYAH JAKARTA" {
		t.Errorf("expected nama IKATAN ALUMNI NTUST WILAYAH JAKARTA, got %s", subgroups[0].Nama)
	}
	if subgroups[0].Wilayah != "Jakarta" {
		t.Errorf("expected wilayah Jakarta, got %s", subgroups[0].Wilayah)
	}
}

func TestAppointAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	middleware.InitJWT("test-secret")
	SetJWTKey([]byte("test-secret"))

	// Create subgroup
	database.SaveSubgroup(context.Background(), &models.Subgroup{
		Nama:    "IKATAN JAKARTA",
		Wilayah: "Jakarta",
		AdminID: "",
	})

	// Create a member alumni to be appointed
	database.SaveAlumni(context.Background(), &models.Alumni{
		ID:           "regional1",
		Nama:         "Regional Admin",
		Email:        "regional@example.com",
		Role:         "member",
		Status:       "verified",
		PasswordHash: "hash",
	})

	// Create super admin
	database.SaveAlumni(context.Background(), &models.Alumni{
		ID:           "superadmin",
		Nama:         "Super Admin",
		Email:        "superadmin@example.com",
		Role:         "super_admin",
		Status:       "verified",
		PasswordHash: "hash",
	})

	h := NewSubgroupHandler(database)

	router := gin.New()
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	protected.Use(middleware.RequireRole("super_admin", "regional_admin"))
	protected.POST("/subgroups/:id/admin", h.AppointAdmin)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "superadmin",
		"role":    "super_admin",
		"status":  "verified",
	})
	tokenString, _ := token.SignedString([]byte("test-secret"))

	body := `{"alumni_id":"regional1"}`
	w := httptest.NewRecorder()
	subs, _ := database.ListSubgroups(context.Background())
	subID := subs[0].ID
	req, _ := http.NewRequest(http.MethodPost, "/subgroups/"+subID+"/admin", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify admin appointed
	sub, err := database.GetSubgroup(context.Background(), subID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sub.AdminID != "regional1" {
		t.Errorf("expected AdminID regional1, got %s", sub.AdminID)
	}

	alumni, _ := database.GetAlumni(context.Background(), "regional1")
	if alumni.Role != "regional_admin" {
		t.Errorf("expected role regional_admin, got %s", alumni.Role)
	}
	if alumni.SubgroupID != subID {
		t.Errorf("expected subgroup_id %s, got %s", subID, alumni.SubgroupID)
	}
}

func TestDeleteSubgroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newFakeDB()
	middleware.InitJWT("test-secret")
	SetJWTKey([]byte("test-secret"))

	database.SaveSubgroup(context.Background(), &models.Subgroup{
		Nama:    "IKATAN BANDUNG",
		Wilayah: "Bandung",
	})

	h := NewSubgroupHandler(database)

	router := gin.New()
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	protected.Use(middleware.RequireRole("super_admin", "regional_admin"))
	protected.DELETE("/subgroups/:id", h.DeleteSubgroup)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "superadmin",
		"role":    "super_admin",
		"status":  "verified",
	})
	tokenString, _ := token.SignedString([]byte("test-secret"))

	subs, _ := database.ListSubgroups(context.Background())
	subID := subs[0].ID

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/subgroups/"+subID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"health-checkup/backend/internal/config"
	"health-checkup/backend/internal/models"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLoginWithEmailCodeDoesNotRequirePassword(t *testing.T) {
	handler, router, redisClient := newPasswordlessAuthFixture(t)
	user := models.User{Name: "管理员", Email: "admin@example.com", Phone: "A1001", Role: "admin", Status: "active"}
	if err := handler.db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := redisClient.Set(context.Background(), authEmailCodeKey(user.Email), "123456", time.Minute).Err(); err != nil {
		t.Fatalf("set code: %v", err)
	}

	response := performPasswordlessAuthRequest(t, router, "/auth/login", map[string]any{"email": user.Email, "code": "123456"})

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if payload["accessToken"] == "" {
		t.Fatalf("expected access token, got %#v", payload)
	}
}

func TestRegisterUserCalculatesAgeFromIDCard(t *testing.T) {
	handler, router, redisClient := newPasswordlessAuthFixture(t)
	email := "user@example.com"
	if err := redisClient.Set(context.Background(), authEmailCodeKey(email), "123456", time.Minute).Err(); err != nil {
		t.Fatalf("set code: %v", err)
	}

	response := performPasswordlessAuthRequest(t, router, "/auth/register/user", map[string]any{
		"name":   "用户",
		"email":  email,
		"code":   "123456",
		"gender": "男",
		"idCard": testIDCard("19900101"),
	})

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	var user models.User
	if err := handler.db.Where("email = ?", email).First(&user).Error; err != nil {
		t.Fatalf("find user: %v", err)
	}
	if user.Age <= 0 || user.PasswordHash != "" {
		t.Fatalf("expected calculated age and empty legacy credential field, got age=%d value=%q", user.Age, user.PasswordHash)
	}
}

func TestRegisterUserRejectsInvalidIDCard(t *testing.T) {
	_, router, _ := newPasswordlessAuthFixture(t)

	response := performPasswordlessAuthRequest(t, router, "/auth/register/user", map[string]any{
		"name":   "用户",
		"email":  "bad@example.com",
		"code":   "123456",
		"idCard": "110105199001010000",
	})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
}

func newPasswordlessAuthFixture(t *testing.T) (*Handler, *gin.Engine, *redis.Client) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.LoginLog{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	handler := &Handler{db: db, redis: redisClient, config: config.Config{JWTSecret: "test-secret", TokenHours: 24}}
	router := gin.New()
	router.POST("/auth/login", handler.login)
	router.POST("/auth/register/user", handler.registerUser)
	return handler, router, redisClient
}

func performPasswordlessAuthRequest(t *testing.T, router *gin.Engine, path string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	return response
}

func testIDCard(birth string) string {
	prefix := "110105" + birth + "001"
	weights := []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	checks := "10X98765432"
	sum := 0
	for i, weight := range weights {
		sum += int(prefix[i]-'0') * weight
	}
	return prefix + string(checks[sum%11])
}

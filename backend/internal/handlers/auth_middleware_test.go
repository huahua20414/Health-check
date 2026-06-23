package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"health-checkup/backend/internal/auth"
	"health-checkup/backend/internal/config"
	"health-checkup/backend/internal/middleware"
	"health-checkup/backend/internal/models"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAuthRequiredReturns401WithoutBearerToken(t *testing.T) {
	router, _ := newAuthMiddlewareTestRouter(t, nil)
	response := performAuthMiddlewareRequest(t, router, "")

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", response.Code, response.Body.String())
	}
	assertUnifiedError(t, response.Body.Bytes(), http.StatusUnauthorized, "missing bearer token")
}

func TestAuthRequiredReturns401WhenSessionIsMissing(t *testing.T) {
	user := models.User{ID: 1, Name: "用户", Email: "user@example.com", Role: "user", Status: "active"}
	router, redisClient := newAuthMiddlewareTestRouter(t, []models.User{user})
	token := issueTestTokenWithoutSession(t, redisClient, user)

	response := performAuthMiddlewareRequest(t, router, token)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", response.Code, response.Body.String())
	}
	assertUnifiedError(t, response.Body.Bytes(), http.StatusUnauthorized, "session expired")
}

func TestAuthRequiredRejectsTokenForDifferentSessionUser(t *testing.T) {
	user := models.User{ID: 1, Name: "管理员", Email: "admin@example.com", Role: "admin", Status: "active"}
	router, redisClient := newAuthMiddlewareTestRouter(t, []models.User{user})
	token := issueTestToken(t, redisClient, user)
	claims, err := auth.ParseToken(token, "test-secret")
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if err := redisClient.Set(context.Background(), auth.SessionKey(claims.SessionID), strconv.Itoa(int(user.ID)+1), time.Hour).Err(); err != nil {
		t.Fatalf("overwrite session: %v", err)
	}

	response := performAuthMiddlewareRequest(t, router, token)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", response.Code, response.Body.String())
	}
	assertUnifiedError(t, response.Body.Bytes(), http.StatusUnauthorized, "session expired")
}

func TestAuthRequiredReturns401ForInactiveUser(t *testing.T) {
	user := models.User{ID: 1, Name: "停用用户", Email: "disabled@example.com", Role: "admin", Status: "disabled"}
	router, redisClient := newAuthMiddlewareTestRouter(t, []models.User{user})
	token := issueTestToken(t, redisClient, user)

	response := performAuthMiddlewareRequest(t, router, token)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", response.Code, response.Body.String())
	}
	assertUnifiedError(t, response.Body.Bytes(), http.StatusUnauthorized, "invalid user")
}

func TestRequireRoleReturns403ForWrongRole(t *testing.T) {
	user := models.User{ID: 1, Name: "普通用户", Email: "user@example.com", Role: "user", Status: "active"}
	router, redisClient := newAuthMiddlewareTestRouter(t, []models.User{user})
	token := issueTestToken(t, redisClient, user)

	response := performAuthMiddlewareRequest(t, router, token)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", response.Code, response.Body.String())
	}
	assertUnifiedError(t, response.Body.Bytes(), http.StatusForbidden, "permission denied")
}

func TestRequireRoleAllowsMatchingRole(t *testing.T) {
	user := models.User{ID: 1, Name: "管理员", Email: "admin@example.com", Role: "admin", Status: "active"}
	router, redisClient := newAuthMiddlewareTestRouter(t, []models.User{user})
	token := issueTestToken(t, redisClient, user)

	response := performAuthMiddlewareRequest(t, router, token)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["code"].(float64) != 0 {
		t.Fatalf("expected success code 0, got %#v", payload)
	}
	data := payload["data"].(map[string]any)
	if data["role"] != "admin" {
		t.Fatalf("expected admin role in response, got %#v", data)
	}
}

func TestRequirePermissionReturns403WhenPermissionDisabled(t *testing.T) {
	user := models.User{ID: 1, Name: "管理员", Email: "admin@example.com", Role: "admin", Status: "active"}
	router, redisClient := newAuthMiddlewareTestRouter(t, []models.User{user}, models.RolePermission{Role: "admin", Permission: "admin:system:manage", Enabled: false})
	token := issueTestToken(t, redisClient, user)

	response := performAuthMiddlewareRequest(t, router, token)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", response.Code, response.Body.String())
	}
	assertUnifiedError(t, response.Body.Bytes(), http.StatusForbidden, "permission denied")
}

func TestRequirePermissionAllowsEnabledPermission(t *testing.T) {
	user := models.User{ID: 1, Name: "管理员", Email: "admin@example.com", Role: "admin", Status: "active"}
	router, redisClient := newAuthMiddlewareTestRouter(t, []models.User{user}, models.RolePermission{Role: "admin", Permission: "admin:system:manage", Enabled: true})
	token := issueTestToken(t, redisClient, user)

	response := performAuthMiddlewareRequest(t, router, token)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
}

func TestRequirePermissionUsesFallbackWhenPermissionTableEmpty(t *testing.T) {
	user := models.User{ID: 1, Name: "管理员", Email: "admin@example.com", Role: "admin", Status: "active"}
	router, redisClient := newAuthMiddlewareTestRouter(t, []models.User{user})
	token := issueTestToken(t, redisClient, user)

	response := performAuthMiddlewareRequest(t, router, token)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
}

func newAuthMiddlewareTestRouter(t *testing.T, users []models.User, permissions ...models.RolePermission) (*gin.Engine, *redis.Client) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.RolePermission{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	for i := range users {
		if users[i].PasswordHash == "" {
			users[i].PasswordHash = "test-hash"
		}
		if err := db.Create(&users[i]).Error; err != nil {
			t.Fatalf("create user: %v", err)
		}
	}
	for i := range permissions {
		row := map[string]any{
			"role":        permissions[i].Role,
			"permission":  permissions[i].Permission,
			"description": permissions[i].Description,
			"enabled":     permissions[i].Enabled,
		}
		if err := db.Model(&models.RolePermission{}).Create(row).Error; err != nil {
			t.Fatalf("create permission: %v", err)
		}
	}
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	handler := &Handler{
		db:     db,
		redis:  redisClient,
		config: config.Config{JWTSecret: "test-secret", TokenHours: 24},
	}
	router := gin.New()
	router.Use(middleware.RequestID(), middleware.UnifiedJSONResponse(), middleware.Recovery())
	router.GET("/admin-only", handler.authRequired(), handler.requireRoleAndPermission("admin:system:manage", "admin"), func(c *gin.Context) {
		current := currentUser(c)
		c.JSON(http.StatusOK, gin.H{"id": current.ID, "role": current.Role})
	})
	return router, redisClient
}

func issueTestToken(t *testing.T, redisClient *redis.Client, user models.User) string {
	t.Helper()
	token, err := auth.IssueToken(context.Background(), redisClient, "test-secret", time.Hour, user)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return token
}

func issueTestTokenWithoutSession(t *testing.T, redisClient *redis.Client, user models.User) string {
	t.Helper()
	token := issueTestToken(t, redisClient, user)
	claims, err := auth.ParseToken(token, "test-secret")
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if err := redisClient.Del(context.Background(), auth.SessionKey(claims.SessionID)).Err(); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	return token
}

func performAuthMiddlewareRequest(t *testing.T, router *gin.Engine, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/admin-only", nil)
	req.Header.Set("X-Request-Id", "auth-test")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func assertUnifiedError(t *testing.T, body []byte, code int, message string) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode response %q: %v", string(body), err)
	}
	if int(payload["code"].(float64)) != code {
		t.Fatalf("expected code %d, got %#v", code, payload["code"])
	}
	if payload["message"] != message {
		t.Fatalf("expected message %q, got %#v", message, payload["message"])
	}
	if payload["requestId"] != "auth-test" {
		t.Fatalf("expected request id auth-test, got %#v", payload["requestId"])
	}
	if payload["data"] != nil {
		t.Fatalf("expected nil data, got %#v", payload["data"])
	}
}

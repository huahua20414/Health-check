package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"health-checkup/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAdminCannotDisableOwnAccount(t *testing.T) {
	handler, db, fixture := newUserStatusFixture(t)
	router := newUserStatusRouter(handler, fixture.admin)

	response := performUserStatusPatch(t, router, fixture.admin.ID, statusRequest{Status: "disabled"})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertUserStatus(t, db, fixture.admin.ID, "active")
}

func TestCannotDisableLastActiveAdmin(t *testing.T) {
	handler, db, fixture := newUserStatusFixture(t)
	if err := db.Model(&models.User{}).Where("id = ?", fixture.otherAdmin.ID).Update("status", "disabled").Error; err != nil {
		t.Fatalf("disable other admin: %v", err)
	}
	router := newUserStatusRouter(handler, fixture.otherAdmin)

	response := performUserStatusPatch(t, router, fixture.admin.ID, statusRequest{Status: "disabled"})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertUserStatus(t, db, fixture.admin.ID, "active")
}

func TestAdminCanDisableAnotherAdminWhenAnotherActiveAdminRemains(t *testing.T) {
	handler, db, fixture := newUserStatusFixture(t)
	router := newUserStatusRouter(handler, fixture.admin)

	response := performUserStatusPatch(t, router, fixture.otherAdmin.ID, statusRequest{Status: "disabled"})

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	assertUserStatus(t, db, fixture.otherAdmin.ID, "disabled")
	assertOperationCount(t, db, "update_status", "user", 1)
}

func TestAdminCanReactivateDisabledUser(t *testing.T) {
	handler, db, fixture := newUserStatusFixture(t)
	router := newUserStatusRouter(handler, fixture.admin)

	response := performUserStatusPatch(t, router, fixture.disabledUser.ID, statusRequest{Status: "active"})

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	assertUserStatus(t, db, fixture.disabledUser.ID, "active")
}

type userStatusFixture struct {
	admin        models.User
	otherAdmin   models.User
	disabledUser models.User
}

func newUserStatusFixture(t *testing.T) (*Handler, *gorm.DB, userStatusFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.OperationLog{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	fixture := userStatusFixture{
		admin:        models.User{ID: 1, Name: "管理员甲", Email: "admin@example.com", Phone: "13800000001", Role: "admin", Status: "active", PasswordHash: "hash"},
		otherAdmin:   models.User{ID: 2, Name: "管理员乙", Email: "admin2@example.com", Phone: "13800000002", Role: "admin", Status: "active", PasswordHash: "hash"},
		disabledUser: models.User{ID: 3, Name: "停用用户", Email: "disabled@example.com", Phone: "13800000003", Role: "user", Status: "disabled", PasswordHash: "hash"},
	}
	for _, row := range []any{&fixture.admin, &fixture.otherAdmin, &fixture.disabledUser} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}, db, fixture
}

func newUserStatusRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.PATCH("/users/:id/status", handler.updateUserStatus)
	return router
}

func performUserStatusPatch(t *testing.T, router *gin.Engine, id uint, body statusRequest) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/users/"+strconv.Itoa(int(id))+"/status", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func assertUserStatus(t *testing.T, db *gorm.DB, id uint, want string) {
	t.Helper()
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		t.Fatalf("load user: %v", err)
	}
	if user.Status != want {
		t.Fatalf("expected user %d status %s, got %s", id, want, user.Status)
	}
}

package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"health-checkup/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRolePermissionsSupportFiltersAndPagination(t *testing.T) {
	handler := newRolePermissionFixture(t)
	router := newRolePermissionRouter(handler)

	response := performRolePermissionGet(t, router, "/role-permissions?role=admin&enabled=true&keyword=系统&page=1&pageSize=1")
	page := decodeRolePermissionPage(t, response)

	if page.Total != 1 || page.Page != 1 || page.PageSize != 1 {
		t.Fatalf("unexpected pagination metadata: %#v", page)
	}
	if len(page.Items) != 1 || page.Items[0].Permission != "admin:system:manage" {
		t.Fatalf("unexpected permission page: %#v", page.Items)
	}
}

func TestRolePermissionsRejectInvalidEnabledFilter(t *testing.T) {
	handler := newRolePermissionFixture(t)
	router := newRolePermissionRouter(handler)

	response := performRolePermissionGet(t, router, "/role-permissions?enabled=maybe")

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
}

func newRolePermissionFixture(t *testing.T) *Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.RolePermission{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	rows := []models.RolePermission{
		{Role: "admin", Permission: "admin:system:manage", Description: "系统设置管理", Enabled: true},
		{Role: "admin", Permission: "admin:data:exchange", Description: "数据导入导出", Enabled: false},
		{Role: "doctor", Permission: "doctor:appointment:update", Description: "医生预约处理", Enabled: true},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("create permissions: %v", err)
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}
}

func newRolePermissionRouter(handler *Handler) *gin.Engine {
	router := gin.New()
	router.GET("/role-permissions", handler.rolePermissions)
	return router
}

type rolePermissionPage struct {
	Items    []models.RolePermission `json:"items"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"pageSize"`
}

func performRolePermissionGet(t *testing.T, router *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeRolePermissionPage(t *testing.T, response *httptest.ResponseRecorder) rolePermissionPage {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var page rolePermissionPage
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode role permission page: %v", err)
	}
	return page
}

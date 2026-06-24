package handlers

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestExportRolePermissionsUsesFiltersAndAudits(t *testing.T) {
	handler := newRolePermissionFixture(t)
	admin := models.User{ID: 99, Name: "管理员", Role: "admin", Status: "active"}
	router := newRolePermissionRouterWithUser(handler, admin)

	response := performRolePermissionGet(t, router, "/role-permissions/export?role=admin&enabled=false&keyword=数据")

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	rows, err := csv.NewReader(strings.NewReader(response.Body.String())).ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected header plus one permission, got %d rows: %#v", len(rows), rows)
	}
	if rows[1][1] != "admin" || rows[1][2] != "admin:data:exchange" || rows[1][4] != "false" {
		t.Fatalf("unexpected role permission csv row: %#v", rows[1])
	}
	assertRolePermissionOperationLogCount(t, handler.db, admin.ID, "export", 1)
}

func newRolePermissionFixture(t *testing.T) *Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.RolePermission{}, &models.OperationLog{}); err != nil {
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
	if err := db.Model(&models.RolePermission{}).Where("permission = ?", "admin:data:exchange").Update("enabled", false).Error; err != nil {
		t.Fatalf("disable data exchange permission: %v", err)
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}
}

func newRolePermissionRouter(handler *Handler) *gin.Engine {
	router := gin.New()
	router.GET("/role-permissions", handler.rolePermissions)
	router.GET("/role-permissions/export", handler.exportRolePermissions)
	return router
}

func newRolePermissionRouterWithUser(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.GET("/role-permissions", handler.rolePermissions)
	router.GET("/role-permissions/export", handler.exportRolePermissions)
	return router
}

func assertRolePermissionOperationLogCount(t *testing.T, db *gorm.DB, userID uint, action string, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&models.OperationLog{}).Where("user_id = ? AND action = ? AND resource = ?", userID, action, "role_permission").Count(&count).Error; err != nil {
		t.Fatalf("count operation logs: %v", err)
	}
	if count != want {
		t.Fatalf("expected %d operation logs, got %d", want, count)
	}
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

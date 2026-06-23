package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"health-checkup/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPackagesPublicCatalogSupportsSearchSortAndPagination(t *testing.T) {
	handler, _ := newPackageCatalogFixture(t)
	router := newPackageCatalogRouter(handler)

	response := performPackageCatalogRequest(t, router, "/packages?page=1&pageSize=2&keyword=筛查&category=慢病筛查&sort=price_desc")

	payload := decodePackageCatalogPage(t, response)
	if payload.Total != 2 || len(payload.Items) != 2 {
		t.Fatalf("expected two matching active packages, got %#v", payload)
	}
	if payload.Items[0].Name != "慢病深度筛查" || payload.Items[1].Name != "基础慢病筛查" {
		t.Fatalf("expected price desc order, got %#v", packageNames(payload.Items))
	}
	for _, pkg := range payload.Items {
		if pkg.Status != "active" || pkg.Category != "慢病筛查" {
			t.Fatalf("public catalog leaked wrong package: %#v", pkg)
		}
	}
}

func TestPackagesPublicCatalogDoesNotExposeInactivePackages(t *testing.T) {
	handler, _ := newPackageCatalogFixture(t)
	router := newPackageCatalogRouter(handler)

	response := performPackageCatalogRequest(t, router, "/packages?keyword=停用")

	payload := decodePackageCatalogList(t, response)
	if len(payload) != 0 {
		t.Fatalf("inactive package should not be exposed publicly, got %#v", payload)
	}
}

func TestPackagesAuthenticatedCatalogCanFilterStatus(t *testing.T) {
	handler, _ := newPackageCatalogFixture(t)
	router := newPackageCatalogRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/packages?status=disabled&sort=created_desc", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	packages := decodePackageCatalogList(t, rec)
	if len(packages) != 1 || packages[0].Status != "disabled" || packages[0].Name != "停用套餐" {
		t.Fatalf("authenticated status filter returned wrong packages: %#v", packages)
	}
}

type packageCatalogPage struct {
	Items    []models.CheckupPackage `json:"items"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"pageSize"`
}

func newPackageCatalogFixture(t *testing.T) (*Handler, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.CheckupPackage{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	base := time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC)
	packages := []models.CheckupPackage{
		{ID: 10, Name: "基础慢病筛查", Category: "慢病筛查", Description: "血糖血脂基础风险筛查", Price: 199, Items: "血糖,血脂", Status: "active", CreatedAt: base},
		{ID: 11, Name: "慢病深度筛查", Category: "慢病筛查", Description: "心脑血管风险筛查", Price: 599, Items: "血糖,血脂,心电图", Status: "active", CreatedAt: base.Add(time.Hour)},
		{ID: 12, Name: "年度综合体检", Category: "年度综合", Description: "企业员工年度体检", Price: 399, Items: "血常规,肝功能", Status: "active", CreatedAt: base.Add(2 * time.Hour)},
		{ID: 13, Name: "停用套餐", Category: "慢病筛查", Description: "停用筛查套餐", Price: 99, Items: "血糖", Status: "disabled", CreatedAt: base.Add(3 * time.Hour)},
	}
	for i := range packages {
		if err := db.Create(&packages[i]).Error; err != nil {
			t.Fatalf("create package: %v", err)
		}
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}, db
}

func newPackageCatalogRouter(handler *Handler) *gin.Engine {
	router := gin.New()
	router.GET("/packages", handler.packages)
	return router
}

func performPackageCatalogRequest(t *testing.T, router *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodePackageCatalogPage(t *testing.T, response *httptest.ResponseRecorder) packageCatalogPage {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload packageCatalogPage
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode package page: %v", err)
	}
	return payload
}

func decodePackageCatalogList(t *testing.T, response *httptest.ResponseRecorder) []models.CheckupPackage {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var packages []models.CheckupPackage
	if err := json.Unmarshal(response.Body.Bytes(), &packages); err != nil {
		t.Fatalf("decode packages: %v", err)
	}
	return packages
}

func packageNames(packages []models.CheckupPackage) []string {
	names := make([]string, 0, len(packages))
	for _, pkg := range packages {
		names = append(names, pkg.Name)
	}
	return names
}

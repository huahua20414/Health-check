package handlers

import (
	"bytes"
	"encoding/csv"
	"io"
	"mime/multipart"
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

func TestExportPackageItemsSupportsPackageFilterAndAudits(t *testing.T) {
	handler, db, fixture := newPackageItemExchangeFixture(t)
	router := newPackageItemExchangeRouter(handler, fixture.admin)

	response := performPackageItemExchangeRequest(t, router, http.MethodGet, "/package-items/export?packageId=20", nil, "")

	records := decodePackageItemCSV(t, response)
	if len(records) != 2 {
		t.Fatalf("expected header plus one package item, got %#v", records)
	}
	if records[1][0] != "基础套餐" || records[1][1] != "血常规" || records[1][2] != "1" || records[1][3] != "true" {
		t.Fatalf("export returned wrong package item row: %#v", records[1])
	}
	assertPackageItemOperationLogCount(t, db, fixture.admin.ID, "export", 1)
}

func TestImportPackageItemsCreatesAndUpdatesByPackageAndItem(t *testing.T) {
	handler, db, fixture := newPackageItemExchangeFixture(t)
	router := newPackageItemExchangeRouter(handler, fixture.admin)
	csvText := strings.Join([]string{
		"package_name,item_name,sort_order,required",
		"基础套餐,血常规,3,false",
		"基础套餐,胸部CT,4,true",
		"",
	}, "\n")

	response := performPackageItemExchangeRequest(t, router, http.MethodPost, "/package-items/import", strings.NewReader(csvText), "package-items.csv")

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	assertPackageItemImportResult(t, response.Body.String(), `"created":1`, `"updated":1`)
	assertPackageItem(t, db, fixture.pkg.ID, fixture.bloodItem.ID, 3, false)
	assertPackageItem(t, db, fixture.pkg.ID, fixture.ctItem.ID, 4, true)
	assertPackageItemOperationLogCount(t, db, fixture.admin.ID, "import", 1)
}

func TestImportPackageItemsRejectsMissingItem(t *testing.T) {
	handler, db, fixture := newPackageItemExchangeFixture(t)
	router := newPackageItemExchangeRouter(handler, fixture.admin)
	csvText := strings.Join([]string{
		"package_name,item_name,sort_order,required",
		"基础套餐,不存在项目,1,true",
		"",
	}, "\n")

	response := performPackageItemExchangeRequest(t, router, http.MethodPost, "/package-items/import", strings.NewReader(csvText), "package-items.csv")

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "checkup item not found for package item: 不存在项目")
	assertPackageItemOperationLogCount(t, db, fixture.admin.ID, "import", 1)
}

type packageItemExchangeFixture struct {
	admin     models.User
	pkg       models.CheckupPackage
	bloodItem models.CheckupItem
	ctItem    models.CheckupItem
}

func newPackageItemExchangeFixture(t *testing.T) (*Handler, *gorm.DB, packageItemExchangeFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.CheckupPackage{}, &models.CheckupItem{}, &models.PackageItem{}, &models.OperationLog{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	fixture := packageItemExchangeFixture{
		admin:     models.User{ID: 1, Name: "管理员", Email: "admin-package-item@example.com", Phone: "13800004001", Role: "admin", Status: "active", PasswordHash: "hash"},
		pkg:       models.CheckupPackage{ID: 20, Name: "基础套餐", Category: "年度综合", Price: 399, Items: "血常规", Status: "active"},
		bloodItem: models.CheckupItem{ID: 30, Name: "血常规", Category: "检验", Department: "检验科", Price: 30, DurationMin: 10, Status: "active"},
		ctItem:    models.CheckupItem{ID: 31, Name: "胸部CT", Category: "影像", Department: "影像科", Price: 180, DurationMin: 20, Status: "active"},
	}
	rows := []any{
		&fixture.admin,
		&fixture.pkg,
		&fixture.bloodItem,
		&fixture.ctItem,
		&models.PackageItem{ID: 40, PackageID: fixture.pkg.ID, ItemID: fixture.bloodItem.ID, SortOrder: 1, Required: true},
	}
	for _, row := range rows {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}, db, fixture
}

func newPackageItemExchangeRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.GET("/package-items/export", handler.exportPackageItems)
	router.POST("/package-items/import", handler.importPackageItems)
	return router
}

func performPackageItemExchangeRequest(t *testing.T, router *gin.Engine, method, path string, body io.Reader, filename string) *httptest.ResponseRecorder {
	t.Helper()
	var requestBody io.Reader
	contentType := ""
	if body != nil {
		var multipartBody bytes.Buffer
		writer := multipart.NewWriter(&multipartBody)
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := io.Copy(part, body); err != nil {
			t.Fatalf("copy csv body: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("close multipart writer: %v", err)
		}
		requestBody = &multipartBody
		contentType = writer.FormDataContentType()
	}
	req := httptest.NewRequest(method, path, requestBody)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodePackageItemCSV(t *testing.T, response *httptest.ResponseRecorder) [][]string {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	records, err := csv.NewReader(strings.NewReader(response.Body.String())).ReadAll()
	if err != nil {
		t.Fatalf("decode package item csv: %v", err)
	}
	return records
}

func assertPackageItemImportResult(t *testing.T, body string, parts ...string) {
	t.Helper()
	for _, part := range parts {
		if !strings.Contains(body, part) {
			t.Fatalf("expected import response to contain %s, got %s", part, body)
		}
	}
}

func assertPackageItem(t *testing.T, db *gorm.DB, packageID, itemID uint, sortOrder int, required bool) {
	t.Helper()
	var link models.PackageItem
	if err := db.Where("package_id = ? AND item_id = ?", packageID, itemID).First(&link).Error; err != nil {
		t.Fatalf("load package item package=%d item=%d: %v", packageID, itemID, err)
	}
	if link.SortOrder != sortOrder || link.Required != required {
		t.Fatalf("unexpected package item package=%d item=%d: %#v", packageID, itemID, link)
	}
}

func assertPackageItemOperationLogCount(t *testing.T, db *gorm.DB, userID uint, action string, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&models.OperationLog{}).Where("user_id = ? AND action = ? AND resource = ?", userID, action, "package_item").Count(&count).Error; err != nil {
		t.Fatalf("count operation logs: %v", err)
	}
	if count != want {
		t.Fatalf("expected %d %s package item operation logs, got %d", want, action, count)
	}
}

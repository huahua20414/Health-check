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

func TestExportCheckupItemsSupportsStatusFilterAndAudits(t *testing.T) {
	handler, db, fixture := newCheckupItemExchangeFixture(t)
	router := newCheckupItemExchangeRouter(handler, fixture.admin)

	response := performCheckupItemExchangeRequest(t, router, http.MethodGet, "/checkup-items/export?status=active", nil, "")

	records := decodeCheckupItemCSV(t, response)
	if len(records) != 2 {
		t.Fatalf("expected header plus one item, got %#v", records)
	}
	if records[1][0] != "血常规" || records[1][1] != "检验" || records[1][3] != "30.00" {
		t.Fatalf("export returned wrong item row: %#v", records[1])
	}
	assertCheckupItemOperationLogCount(t, db, fixture.admin.ID, "export", 1)
}

func TestImportCheckupItemsCreatesAndUpdatesByName(t *testing.T) {
	handler, db, fixture := newCheckupItemExchangeFixture(t)
	router := newCheckupItemExchangeRouter(handler, fixture.admin)
	csvText := strings.Join([]string{
		"name,category,department,price,duration_min,description,status",
		"血常规,检验,检验科,35,15,批量更新,active",
		"胸部CT,影像,影像科,180,20,批量新增,active",
		"",
	}, "\n")

	response := performCheckupItemExchangeRequest(t, router, http.MethodPost, "/checkup-items/import", strings.NewReader(csvText), "items.csv")

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	assertCheckupItemImportResult(t, response.Body.String(), `"created":1`, `"updated":1`)
	assertCheckupItem(t, db, "血常规", "检验", "检验科", 35, 15)
	assertCheckupItem(t, db, "胸部CT", "影像", "影像科", 180, 20)
	assertCheckupItemOperationLogCount(t, db, fixture.admin.ID, "import", 1)
}

func TestImportCheckupItemsRejectsInvalidDuration(t *testing.T) {
	handler, db, fixture := newCheckupItemExchangeFixture(t)
	router := newCheckupItemExchangeRouter(handler, fixture.admin)
	csvText := strings.Join([]string{
		"name,category,price,duration_min",
		"错误项目,检验,20,0",
		"",
	}, "\n")

	response := performCheckupItemExchangeRequest(t, router, http.MethodPost, "/checkup-items/import", strings.NewReader(csvText), "items.csv")

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "invalid checkup item duration for 错误项目")
	assertCheckupItemOperationLogCount(t, db, fixture.admin.ID, "import", 1)
}

type checkupItemExchangeFixture struct {
	admin models.User
}

func newCheckupItemExchangeFixture(t *testing.T) (*Handler, *gorm.DB, checkupItemExchangeFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.CheckupItem{}, &models.OperationLog{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	fixture := checkupItemExchangeFixture{
		admin: models.User{ID: 1, Name: "管理员", Email: "admin-item@example.com", Phone: "13800003001", Role: "admin", Status: "active", PasswordHash: "hash"},
	}
	rows := []any{
		&fixture.admin,
		&models.CheckupItem{ID: 10, Name: "血常规", Category: "检验", Department: "检验科", Price: 30, DurationMin: 10, Description: "基础检验", Status: "active"},
		&models.CheckupItem{ID: 11, Name: "归档项目", Category: "检验", Department: "检验科", Price: 10, DurationMin: 5, Status: "deleted"},
	}
	for _, row := range rows {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}, db, fixture
}

func newCheckupItemExchangeRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.GET("/checkup-items/export", handler.exportCheckupItems)
	router.POST("/checkup-items/import", handler.importCheckupItems)
	return router
}

func performCheckupItemExchangeRequest(t *testing.T, router *gin.Engine, method, path string, body io.Reader, filename string) *httptest.ResponseRecorder {
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

func decodeCheckupItemCSV(t *testing.T, response *httptest.ResponseRecorder) [][]string {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	records, err := csv.NewReader(strings.NewReader(response.Body.String())).ReadAll()
	if err != nil {
		t.Fatalf("decode checkup item csv: %v", err)
	}
	return records
}

func assertCheckupItemImportResult(t *testing.T, body string, parts ...string) {
	t.Helper()
	for _, part := range parts {
		if !strings.Contains(body, part) {
			t.Fatalf("expected import response to contain %s, got %s", part, body)
		}
	}
}

func assertCheckupItem(t *testing.T, db *gorm.DB, name, category, department string, price float64, duration int) {
	t.Helper()
	var item models.CheckupItem
	if err := db.Where("name = ?", name).First(&item).Error; err != nil {
		t.Fatalf("load checkup item %s: %v", name, err)
	}
	if item.Category != category || item.Department != department || item.Price != price || item.DurationMin != duration {
		t.Fatalf("unexpected checkup item %s: %#v", name, item)
	}
}

func assertCheckupItemOperationLogCount(t *testing.T, db *gorm.DB, userID uint, action string, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&models.OperationLog{}).Where("user_id = ? AND action = ? AND resource = ?", userID, action, "checkup_item").Count(&count).Error; err != nil {
		t.Fatalf("count operation logs: %v", err)
	}
	if count != want {
		t.Fatalf("expected %d %s checkup item operation logs, got %d", want, action, count)
	}
}

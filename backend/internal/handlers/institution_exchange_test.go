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

func TestExportInstitutionsSupportsStatusFilterAndAudits(t *testing.T) {
	handler, db, fixture := newInstitutionExchangeFixture(t)
	router := newInstitutionExchangeRouter(handler, fixture.admin)

	response := performInstitutionExchangeRequest(t, router, http.MethodGet, "/institutions/export?status=active", nil, "")

	records := decodeInstitutionCSV(t, response)
	if len(records) != 2 {
		t.Fatalf("expected header plus one institution, got %#v", records)
	}
	if records[1][0] != "主院区" || records[1][1] != "健康路 1 号" || records[1][3] != "08:00-17:00" {
		t.Fatalf("export returned wrong institution row: %#v", records[1])
	}
	assertInstitutionOperationLogCount(t, db, fixture.admin.ID, "export", 1)
}

func TestImportInstitutionsCreatesAndUpdatesByName(t *testing.T) {
	handler, db, fixture := newInstitutionExchangeFixture(t)
	router := newInstitutionExchangeRouter(handler, fixture.admin)
	csvText := strings.Join([]string{
		"name,address,phone,open_hours,status",
		"主院区,健康路 99 号,400-999-0000,08:30-17:30,disabled",
		"东区体检中心,东区大道 8 号,400-888-0000,09:00-18:00,active",
		"",
	}, "\n")

	response := performInstitutionExchangeRequest(t, router, http.MethodPost, "/institutions/import", strings.NewReader(csvText), "institutions.csv")

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	assertInstitutionImportResult(t, response.Body.String(), `"created":1`, `"updated":1`)
	assertInstitution(t, db, "主院区", "健康路 99 号", "400-999-0000", "08:30-17:30", "disabled")
	assertInstitution(t, db, "东区体检中心", "东区大道 8 号", "400-888-0000", "09:00-18:00", "active")
	assertInstitutionOperationLogCount(t, db, fixture.admin.ID, "import", 1)
}

func TestImportInstitutionsRejectsMissingAddress(t *testing.T) {
	handler, db, fixture := newInstitutionExchangeFixture(t)
	router := newInstitutionExchangeRouter(handler, fixture.admin)
	csvText := strings.Join([]string{
		"name,address,phone",
		"错误院区,,400-000-0000",
		"",
	}, "\n")

	response := performInstitutionExchangeRequest(t, router, http.MethodPost, "/institutions/import", strings.NewReader(csvText), "institutions.csv")

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "institution name and address are required")
	assertInstitutionOperationLogCount(t, db, fixture.admin.ID, "import", 1)
}

type institutionExchangeFixture struct {
	admin models.User
}

func newInstitutionExchangeFixture(t *testing.T) (*Handler, *gorm.DB, institutionExchangeFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.CheckupInstitution{}, &models.OperationLog{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	fixture := institutionExchangeFixture{
		admin: models.User{ID: 1, Name: "管理员", Email: "admin-institution@example.com", Phone: "13800005001", Role: "admin", Status: "active", PasswordHash: "hash"},
	}
	rows := []any{
		&fixture.admin,
		&models.CheckupInstitution{ID: 10, Name: "主院区", Address: "健康路 1 号", Phone: "400-100-1000", OpenHours: "08:00-17:00", Status: "active"},
		&models.CheckupInstitution{ID: 11, Name: "归档院区", Address: "旧址", Status: "deleted"},
	}
	for _, row := range rows {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}, db, fixture
}

func newInstitutionExchangeRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.GET("/institutions/export", handler.exportInstitutions)
	router.POST("/institutions/import", handler.importInstitutions)
	return router
}

func performInstitutionExchangeRequest(t *testing.T, router *gin.Engine, method, path string, body io.Reader, filename string) *httptest.ResponseRecorder {
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

func decodeInstitutionCSV(t *testing.T, response *httptest.ResponseRecorder) [][]string {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	records, err := csv.NewReader(strings.NewReader(response.Body.String())).ReadAll()
	if err != nil {
		t.Fatalf("decode institution csv: %v", err)
	}
	return records
}

func assertInstitutionImportResult(t *testing.T, body string, parts ...string) {
	t.Helper()
	for _, part := range parts {
		if !strings.Contains(body, part) {
			t.Fatalf("expected import response to contain %s, got %s", part, body)
		}
	}
}

func assertInstitution(t *testing.T, db *gorm.DB, name, address, phone, openHours, status string) {
	t.Helper()
	var institution models.CheckupInstitution
	if err := db.Where("name = ?", name).First(&institution).Error; err != nil {
		t.Fatalf("load institution %s: %v", name, err)
	}
	if institution.Address != address || institution.Phone != phone || institution.OpenHours != openHours || institution.Status != status {
		t.Fatalf("unexpected institution %s: %#v", name, institution)
	}
}

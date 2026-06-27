package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"health-checkup/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPublicInstitutionsOnlyReturnsActiveRows(t *testing.T) {
	handler, _ := newInstitutionTestFixture(t)
	router := newInstitutionListRouter(handler, false)

	response := performInstitutionRequest(t, router, http.MethodGet, "/institutions", nil)
	rows := decodeInstitutionRows(t, response)

	if len(rows) != 1 || rows[0].Name != "主院区" {
		t.Fatalf("expected only active public institution, got %#v", rows)
	}
}

func TestAdminInstitutionsHideDeletedByDefaultAndCanFilterStatus(t *testing.T) {
	handler, _ := newInstitutionTestFixture(t)
	router := newInstitutionListRouter(handler, true)

	defaultResponse := performInstitutionRequest(t, router, http.MethodGet, "/institutions", nil)
	defaultRows := decodeInstitutionRows(t, defaultResponse)
	if len(defaultRows) != 2 {
		t.Fatalf("expected active and disabled rows by default, got %#v", defaultRows)
	}
	for _, row := range defaultRows {
		if row.Status == "deleted" {
			t.Fatalf("default admin list leaked deleted row: %#v", defaultRows)
		}
	}

	deletedResponse := performInstitutionRequest(t, router, http.MethodGet, "/institutions?status=deleted", nil)
	deletedRows := decodeInstitutionRows(t, deletedResponse)
	if len(deletedRows) != 1 || deletedRows[0].Name != "归档院区" {
		t.Fatalf("expected only deleted row, got %#v", deletedRows)
	}
}

func TestAdminInstitutionsSupportKeywordAndPagination(t *testing.T) {
	handler, _ := newInstitutionTestFixture(t)
	router := newInstitutionListRouter(handler, true)

	response := performInstitutionRequest(t, router, http.MethodGet, "/institutions?keyword=健康路&page=1&pageSize=1", nil)
	page := decodeInstitutionPage(t, response)

	if page.Total != 2 || page.Page != 1 || page.PageSize != 1 {
		t.Fatalf("unexpected pagination metadata: %#v", page)
	}
	if len(page.Items) != 1 || page.Items[0].Name != "主院区" {
		t.Fatalf("unexpected paginated items: %#v", page.Items)
	}
}

func TestAdminCreatesAndUpdatesInstitutionWithAudit(t *testing.T) {
	handler, db := newInstitutionTestFixture(t)
	router := newInstitutionAdminRouter(handler, models.User{ID: 99, Name: "管理员", Role: "admin", Status: "active"})

	createBody := `{"name":"新体检中心","address":"健康大道 99 号","phone":"400-800-1000","openHours":"08:00-17:00","status":"active"}`
	createResponse := performInstitutionRequest(t, router, http.MethodPost, "/institutions", strings.NewReader(createBody))
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", createResponse.Code, createResponse.Body.String())
	}
	created := decodeInstitutionRow(t, createResponse)
	if created.Name != "新体检中心" || created.Address != "健康大道 99 号" {
		t.Fatalf("created wrong institution: %#v", created)
	}

	updateBody := `{"name":"新体检中心东区","address":"健康大道 100 号","phone":"400-800-2000","openHours":"08:30-17:30","status":"disabled"}`
	updateResponse := performInstitutionRequest(t, router, http.MethodPatch, "/institutions/"+strconv.Itoa(int(created.ID)), strings.NewReader(updateBody))
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", updateResponse.Code, updateResponse.Body.String())
	}
	updated := decodeInstitutionRow(t, updateResponse)
	if updated.Name != "新体检中心东区" || updated.Status != "disabled" || updated.OpenHours != "08:30-17:30" {
		t.Fatalf("updated wrong institution: %#v", updated)
	}
	assertInstitutionOperationLogCount(t, db, 99, "create", 1)
	assertInstitutionOperationLogCount(t, db, 99, "update", 1)
}

func TestArchiveInstitutionArchivesUnusedScheduleSlots(t *testing.T) {
	handler, db := newInstitutionTestFixture(t)
	router := newInstitutionAdminRouter(handler, models.User{ID: 99, Name: "管理员", Role: "admin", Status: "active"})
	institution := models.CheckupInstitution{Name: "占用院区", Address: "占用路", Status: "active"}
	doctor := models.User{Name: "医生", Role: "doctor", Status: "active"}
	if err := db.Create(&institution).Error; err != nil {
		t.Fatalf("create institution: %v", err)
	}
	if err := db.Create(&doctor).Error; err != nil {
		t.Fatalf("create doctor: %v", err)
	}
	slot := models.ScheduleSlot{
		DoctorID:      doctor.ID,
		InstitutionID: institution.ID,
		Date:          "2026-07-03",
		Period:        "上午",
		Category:      "年度综合",
		StartTime:     "09:00",
		EndTime:       "09:30",
		Capacity:      1,
		Status:        "available",
	}
	if err := db.Create(&slot).Error; err != nil {
		t.Fatalf("create schedule slot: %v", err)
	}

	response := performInstitutionRequest(t, router, http.MethodDelete, "/institutions/"+strconv.Itoa(int(institution.ID)), nil)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var archivedSlot models.ScheduleSlot
	if err := db.First(&archivedSlot, slot.ID).Error; err != nil {
		t.Fatalf("load archived slot: %v", err)
	}
	if archivedSlot.Status != "deleted" {
		t.Fatalf("expected related slot deleted, got %q", archivedSlot.Status)
	}
	assertInstitutionOperationLogCount(t, db, 99, "archive", 1)
}

func TestArchiveInstitutionRejectsBookedScheduleSlots(t *testing.T) {
	handler, db := newInstitutionTestFixture(t)
	router := newInstitutionAdminRouter(handler, models.User{ID: 99, Name: "管理员", Role: "admin", Status: "active"})
	institution := models.CheckupInstitution{Name: "已预约院区", Address: "预约路", Status: "active"}
	doctor := models.User{Name: "医生", Role: "doctor", Status: "active"}
	if err := db.Create(&institution).Error; err != nil {
		t.Fatalf("create institution: %v", err)
	}
	if err := db.Create(&doctor).Error; err != nil {
		t.Fatalf("create doctor: %v", err)
	}
	slot := models.ScheduleSlot{
		DoctorID:      doctor.ID,
		InstitutionID: institution.ID,
		Date:          "2026-07-03",
		Period:        "上午",
		Category:      "年度综合",
		StartTime:     "09:00",
		EndTime:       "09:30",
		Capacity:      1,
		BookedCount:   1,
		Status:        "available",
	}
	if err := db.Create(&slot).Error; err != nil {
		t.Fatalf("create schedule slot: %v", err)
	}

	response := performInstitutionRequest(t, router, http.MethodDelete, "/institutions/"+strconv.Itoa(int(institution.ID)), nil)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "institution has booked schedule slots")
}

func TestArchiveInstitutionAllowsUnusedInstitution(t *testing.T) {
	handler, db := newInstitutionTestFixture(t)
	router := newInstitutionAdminRouter(handler, models.User{ID: 99, Name: "管理员", Role: "admin", Status: "active"})
	institution := models.CheckupInstitution{Name: "可归档院区", Address: "可归档路", Status: "active"}
	if err := db.Create(&institution).Error; err != nil {
		t.Fatalf("create institution: %v", err)
	}

	response := performInstitutionRequest(t, router, http.MethodDelete, "/institutions/"+strconv.Itoa(int(institution.ID)), nil)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var archived models.CheckupInstitution
	if err := db.First(&archived, institution.ID).Error; err != nil {
		t.Fatalf("load archived institution: %v", err)
	}
	if archived.Status != "deleted" {
		t.Fatalf("expected deleted status, got %q", archived.Status)
	}
	assertInstitutionOperationLogCount(t, db, 99, "archive", 1)
}

func newInstitutionTestFixture(t *testing.T) (*Handler, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.CheckupInstitution{}, &models.ScheduleSlot{}, &models.OperationLog{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	rows := []models.CheckupInstitution{
		{Name: "主院区", Address: "健康路 1 号", Phone: "400-100-1000", OpenHours: "08:00-17:00", Status: "active"},
		{Name: "停用院区", Address: "健康路 2 号", Status: "disabled"},
		{Name: "归档院区", Address: "健康路 3 号", Status: "deleted"},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("create institutions: %v", err)
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}, db
}

func newInstitutionListRouter(handler *Handler, withAuthHeader bool) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if withAuthHeader {
			c.Request.Header.Set("Authorization", "Bearer test")
		}
		c.Next()
	})
	router.GET("/institutions", handler.institutions)
	return router
}

func newInstitutionAdminRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.POST("/institutions", handler.createInstitution)
	router.PATCH("/institutions/:id", handler.updateInstitution)
	router.DELETE("/institutions/:id", handler.archiveInstitution)
	return router
}

func performInstitutionRequest(t *testing.T, router *gin.Engine, method, path string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	requestBody := body
	if requestBody == nil {
		requestBody = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, requestBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeInstitutionRows(t *testing.T, response *httptest.ResponseRecorder) []models.CheckupInstitution {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var rows []models.CheckupInstitution
	if err := json.Unmarshal(response.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode institutions: %v", err)
	}
	return rows
}

func decodeInstitutionRow(t *testing.T, response *httptest.ResponseRecorder) models.CheckupInstitution {
	t.Helper()
	var row models.CheckupInstitution
	if err := json.Unmarshal(response.Body.Bytes(), &row); err != nil {
		t.Fatalf("decode institution: %v", err)
	}
	return row
}

type institutionPageResponse struct {
	Items    []models.CheckupInstitution `json:"items"`
	Total    int64                       `json:"total"`
	Page     int                         `json:"page"`
	PageSize int                         `json:"pageSize"`
}

func decodeInstitutionPage(t *testing.T, response *httptest.ResponseRecorder) institutionPageResponse {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var page institutionPageResponse
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode institution page: %v", err)
	}
	return page
}

func assertInstitutionOperationLogCount(t *testing.T, db *gorm.DB, userID uint, action string, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&models.OperationLog{}).Where("user_id = ? AND action = ? AND resource = ?", userID, action, "institution").Count(&count).Error; err != nil {
		t.Fatalf("count operation logs: %v", err)
	}
	if count != want {
		t.Fatalf("expected %d %s institution operation logs, got %d", want, action, count)
	}
}

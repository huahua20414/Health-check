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

func TestExportScheduleSlotsUsesFiltersAndAudits(t *testing.T) {
	handler, db, fixture := newScheduleSlotExchangeFixture(t)
	router := newScheduleSlotExchangeRouter(handler, fixture.admin)

	response := performScheduleSlotExchangeRequest(t, router, http.MethodGet, "/schedule/slots/export?date=2026-07-01&category=年度综合", nil, "")

	records := decodeScheduleSlotCSV(t, response)
	if len(records) != 2 {
		t.Fatalf("expected header plus one slot, got %#v", records)
	}
	if records[1][0] != fixture.doctor.Email || records[1][2] != fixture.institution.Name || records[1][6] != "09:00" || records[1][8] != "2" {
		t.Fatalf("export returned wrong schedule slot row: %#v", records[1])
	}
	assertScheduleSlotExchangeOperationLogCount(t, db, fixture.admin.ID, "export", 1)
}

func TestImportScheduleSlotsCreatesAndUpdatesByDoctorInstitutionDateStartTime(t *testing.T) {
	handler, db, fixture := newScheduleSlotExchangeFixture(t)
	router := newScheduleSlotExchangeRouter(handler, fixture.admin)
	csvText := strings.Join([]string{
		"doctor_email,institution_name,date,period,category,start_time,end_time,capacity,status",
		fixture.doctor.Email + "," + fixture.institution.Name + ",2026-07-01,上午,年度综合,09:00,09:30,3,available",
		fixture.doctor.Email + "," + fixture.institution.Name + ",2026-07-01,上午,年度综合,09:30,10:00,1,available",
		"",
	}, "\n")

	response := performScheduleSlotExchangeRequest(t, router, http.MethodPost, "/schedule/slots/import", strings.NewReader(csvText), "slots.csv")

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	assertScheduleSlotImportResult(t, response.Body.String(), `"created":1`, `"updated":1`)
	assertScheduleSlot(t, db, fixture.doctor.ID, "2026-07-01", "09:00", 3, 1)
	assertScheduleSlot(t, db, fixture.doctor.ID, "2026-07-01", "09:30", 1, 0)
	assertScheduleSlotExchangeOperationLogCount(t, db, fixture.admin.ID, "import", 1)
}

func TestImportScheduleSlotsRejectsOverlappingSlot(t *testing.T) {
	handler, db, fixture := newScheduleSlotExchangeFixture(t)
	router := newScheduleSlotExchangeRouter(handler, fixture.admin)
	csvText := strings.Join([]string{
		"doctor_email,institution_name,date,period,category,start_time,end_time,capacity,status",
		fixture.doctor.Email + "," + fixture.institution.Name + ",2026-07-01,上午,年度综合,09:15,09:45,1,available",
		"",
	}, "\n")

	response := performScheduleSlotExchangeRequest(t, router, http.MethodPost, "/schedule/slots/import", strings.NewReader(csvText), "slots.csv")

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "schedule slot overlaps with existing slot")
	assertScheduleSlotExchangeOperationLogCount(t, db, fixture.admin.ID, "import", 1)
}

type scheduleSlotExchangeFixture struct {
	admin       models.User
	doctor      models.User
	institution models.CheckupInstitution
}

func newScheduleSlotExchangeFixture(t *testing.T) (*Handler, *gorm.DB, scheduleSlotExchangeFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.CheckupInstitution{}, &models.ScheduleSlot{}, &models.OperationLog{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	fixture := scheduleSlotExchangeFixture{
		admin:       models.User{ID: 1, Name: "管理员", Email: "admin-slot@example.com", Phone: "13800005001", Role: "admin", Status: "active", PasswordHash: "hash"},
		doctor:      models.User{ID: 2, Name: "医生", Email: "doctor-slot@example.com", Phone: "13800005002", Role: "doctor", Status: "active", PasswordHash: "hash"},
		institution: models.CheckupInstitution{ID: 10, Name: "主院区", Address: "健康路 1 号", Status: "active"},
	}
	rows := []any{
		&fixture.admin,
		&fixture.doctor,
		&fixture.institution,
		&models.ScheduleSlot{ID: 20, DoctorID: fixture.doctor.ID, InstitutionID: fixture.institution.ID, Date: "2026-07-01", Period: "上午", Category: "年度综合", StartTime: "09:00", EndTime: "09:30", Capacity: 2, BookedCount: 1, Status: "available"},
		&models.ScheduleSlot{ID: 21, DoctorID: fixture.doctor.ID, InstitutionID: fixture.institution.ID, Date: "2026-07-02", Period: "上午", Category: "入职体检", StartTime: "10:00", EndTime: "10:30", Capacity: 1, BookedCount: 0, Status: "available"},
	}
	for _, row := range rows {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}, db, fixture
}

func newScheduleSlotExchangeRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.GET("/schedule/slots/export", handler.exportScheduleSlots)
	router.POST("/schedule/slots/import", handler.importScheduleSlots)
	return router
}

func performScheduleSlotExchangeRequest(t *testing.T, router *gin.Engine, method, path string, body io.Reader, filename string) *httptest.ResponseRecorder {
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

func decodeScheduleSlotCSV(t *testing.T, response *httptest.ResponseRecorder) [][]string {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	records, err := csv.NewReader(strings.NewReader(response.Body.String())).ReadAll()
	if err != nil {
		t.Fatalf("decode schedule slot csv: %v", err)
	}
	return records
}

func assertScheduleSlotImportResult(t *testing.T, body string, parts ...string) {
	t.Helper()
	for _, part := range parts {
		if !strings.Contains(body, part) {
			t.Fatalf("expected import response to contain %s, got %s", part, body)
		}
	}
}

func assertScheduleSlot(t *testing.T, db *gorm.DB, doctorID uint, date, startTime string, capacity, bookedCount int) {
	t.Helper()
	var slot models.ScheduleSlot
	if err := db.Where("doctor_id = ? AND date = ? AND start_time = ?", doctorID, date, startTime).First(&slot).Error; err != nil {
		t.Fatalf("load schedule slot %s %s: %v", date, startTime, err)
	}
	if slot.Capacity != capacity || slot.BookedCount != bookedCount {
		t.Fatalf("unexpected schedule slot %s %s: %#v", date, startTime, slot)
	}
}

func assertScheduleSlotExchangeOperationLogCount(t *testing.T, db *gorm.DB, userID uint, action string, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&models.OperationLog{}).Where("user_id = ? AND action = ? AND resource = ?", userID, action, "schedule_slot").Count(&count).Error; err != nil {
		t.Fatalf("count operation logs: %v", err)
	}
	if count != want {
		t.Fatalf("expected %d %s schedule slot operation logs, got %d", want, action, count)
	}
}

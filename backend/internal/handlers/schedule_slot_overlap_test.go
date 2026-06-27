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

func TestCreateScheduleSlotRejectsOverlappingSlot(t *testing.T) {
	handler, _, fixture := newScheduleSlotOverlapFixture(t)
	router := newScheduleSlotOverlapRouter(handler, fixture.admin)

	req := scheduleSlotRequest{DoctorID: fixture.doctor.ID, InstitutionID: fixture.institution.ID, Date: "2026-07-01", Category: "年度综合", StartTime: "09:15", EndTime: "09:45", Capacity: 1}
	response := performCreateScheduleSlotRequest(t, router, req)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "schedule slot overlaps with existing slot")
}

func TestCreateScheduleSlotAllowsAdjacentSlot(t *testing.T) {
	handler, db, fixture := newScheduleSlotOverlapFixture(t)
	router := newScheduleSlotOverlapRouter(handler, fixture.admin)

	req := scheduleSlotRequest{DoctorID: fixture.doctor.ID, InstitutionID: fixture.institution.ID, Date: "2026-07-01", Category: "年度综合", StartTime: "09:30", EndTime: "10:00", Capacity: 1}
	response := performCreateScheduleSlotRequest(t, router, req)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	assertScheduleSlotCount(t, db, fixture.doctor.ID, "2026-07-01", 3)
}

func TestCreateScheduleSlotAllowsDifferentDoctorAtSameTime(t *testing.T) {
	handler, db, fixture := newScheduleSlotOverlapFixture(t)
	router := newScheduleSlotOverlapRouter(handler, fixture.admin)

	req := scheduleSlotRequest{DoctorID: fixture.otherDoctor.ID, InstitutionID: fixture.institution.ID, Date: "2026-07-01", Category: "年度综合", StartTime: "09:15", EndTime: "09:45", Capacity: 1}
	response := performCreateScheduleSlotRequest(t, router, req)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	assertScheduleSlotCount(t, db, fixture.otherDoctor.ID, "2026-07-01", 1)
}

func TestCreateScheduleSlotRejectsUnsupportedInstitutionCategory(t *testing.T) {
	handler, _, fixture := newScheduleSlotOverlapFixture(t)
	router := newScheduleSlotOverlapRouter(handler, fixture.admin)

	req := scheduleSlotRequest{DoctorID: fixture.doctor.ID, InstitutionID: fixture.institution.ID, Date: "2026-07-01", Category: "影像专项", StartTime: "11:00", EndTime: "11:30", Capacity: 1}
	response := performCreateScheduleSlotRequest(t, router, req)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "institution does not support selected category")
}

func TestUpdateScheduleSlotAllowsUpdatingItself(t *testing.T) {
	handler, db, fixture := newScheduleSlotOverlapFixture(t)
	router := newScheduleSlotOverlapRouter(handler, fixture.admin)

	req := scheduleSlotRequest{DoctorID: fixture.doctor.ID, InstitutionID: fixture.institution.ID, Date: "2026-07-01", Category: "入职体检", StartTime: "09:00", EndTime: "09:30", Capacity: 2}
	response := performUpdateScheduleSlotRequest(t, router, fixture.existingSlot.ID, req)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var slot models.ScheduleSlot
	if err := db.First(&slot, fixture.existingSlot.ID).Error; err != nil {
		t.Fatalf("load slot: %v", err)
	}
	if slot.Category != "入职体检" || slot.Capacity != 2 {
		t.Fatalf("expected category/capacity to update, got %s/%d", slot.Category, slot.Capacity)
	}
}

func TestUpdateScheduleSlotRejectsOverlapWithOtherSlot(t *testing.T) {
	handler, _, fixture := newScheduleSlotOverlapFixture(t)
	router := newScheduleSlotOverlapRouter(handler, fixture.admin)

	req := scheduleSlotRequest{DoctorID: fixture.doctor.ID, InstitutionID: fixture.institution.ID, Date: "2026-07-01", Category: "年度综合", StartTime: "09:15", EndTime: "09:45", Capacity: 1}
	response := performUpdateScheduleSlotRequest(t, router, fixture.laterSlot.ID, req)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "schedule slot overlaps with existing slot")
}

func TestUpdateBookedScheduleSlotRejectsAssignmentChange(t *testing.T) {
	handler, db, fixture := newScheduleSlotOverlapFixture(t)
	router := newScheduleSlotOverlapRouter(handler, fixture.admin)
	if err := db.Model(&models.ScheduleSlot{}).Where("id = ?", fixture.existingSlot.ID).Update("booked_count", 1).Error; err != nil {
		t.Fatalf("mark slot booked: %v", err)
	}

	req := scheduleSlotRequest{DoctorID: fixture.otherDoctor.ID, InstitutionID: fixture.institution.ID, Date: "2026-07-01", Category: "年度综合", StartTime: "09:00", EndTime: "09:30", Capacity: 2}
	response := performUpdateScheduleSlotRequest(t, router, fixture.existingSlot.ID, req)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "booked schedule slot cannot change doctor, institution, date or time")
}

func TestUpdateBookedScheduleSlotAllowsCapacityIncrease(t *testing.T) {
	handler, db, fixture := newScheduleSlotOverlapFixture(t)
	router := newScheduleSlotOverlapRouter(handler, fixture.admin)
	if err := db.Model(&models.ScheduleSlot{}).Where("id = ?", fixture.existingSlot.ID).Update("booked_count", 1).Error; err != nil {
		t.Fatalf("mark slot booked: %v", err)
	}

	req := scheduleSlotRequest{DoctorID: fixture.doctor.ID, InstitutionID: fixture.institution.ID, Date: "2026-07-01", Period: "上午", Category: "年度综合", StartTime: "09:00", EndTime: "09:30", Capacity: 3}
	response := performUpdateScheduleSlotRequest(t, router, fixture.existingSlot.ID, req)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var slot models.ScheduleSlot
	if err := db.First(&slot, fixture.existingSlot.ID).Error; err != nil {
		t.Fatalf("load slot: %v", err)
	}
	if slot.Capacity != 3 || slot.BookedCount != 1 {
		t.Fatalf("expected capacity/booked count to be 3/1, got %d/%d", slot.Capacity, slot.BookedCount)
	}
}

func TestUpdateBookedScheduleSlotAllowsCategoryChange(t *testing.T) {
	handler, db, fixture := newScheduleSlotOverlapFixture(t)
	router := newScheduleSlotOverlapRouter(handler, fixture.admin)
	if err := db.Model(&models.ScheduleSlot{}).Where("id = ?", fixture.existingSlot.ID).Update("booked_count", 1).Error; err != nil {
		t.Fatalf("mark slot booked: %v", err)
	}

	req := scheduleSlotRequest{DoctorID: fixture.doctor.ID, InstitutionID: fixture.institution.ID, Date: "2026-07-01", Period: "上午", Category: "入职体检", StartTime: "09:00", EndTime: "09:30", Capacity: 2}
	response := performUpdateScheduleSlotRequest(t, router, fixture.existingSlot.ID, req)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var slot models.ScheduleSlot
	if err := db.First(&slot, fixture.existingSlot.ID).Error; err != nil {
		t.Fatalf("load slot: %v", err)
	}
	if slot.Category != "入职体检" || slot.BookedCount != 1 {
		t.Fatalf("expected category change with booked count preserved, got %#v", slot)
	}
}

func TestCreateScheduleSlotRejectsInvalidTimeRange(t *testing.T) {
	handler, _, fixture := newScheduleSlotOverlapFixture(t)
	router := newScheduleSlotOverlapRouter(handler, fixture.admin)

	req := scheduleSlotRequest{DoctorID: fixture.doctor.ID, InstitutionID: fixture.institution.ID, Date: "2026-07-01", Category: "年度综合", StartTime: "10:00", EndTime: "10:00", Capacity: 1}
	response := performCreateScheduleSlotRequest(t, router, req)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "end time must be after start time")
}

type scheduleSlotOverlapFixture struct {
	admin        models.User
	doctor       models.User
	otherDoctor  models.User
	institution  models.CheckupInstitution
	existingSlot models.ScheduleSlot
	laterSlot    models.ScheduleSlot
}

func newScheduleSlotOverlapFixture(t *testing.T) (*Handler, *gorm.DB, scheduleSlotOverlapFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.CheckupInstitution{}, &models.CheckupPackage{}, &models.InstitutionPackage{}, &models.ScheduleSlot{}, &models.OperationLog{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	annualPackage := models.CheckupPackage{ID: 30, Name: "年度套餐", Category: "年度综合", Price: 399, Status: "active"}
	entryPackage := models.CheckupPackage{ID: 31, Name: "入职套餐", Category: "入职体检", Price: 199, Status: "active"}
	fixture := scheduleSlotOverlapFixture{
		admin:        models.User{ID: 1, Name: "管理员", Email: "admin-overlap@example.com", Phone: "13800001001", Role: "admin", Status: "active", PasswordHash: "hash"},
		doctor:       models.User{ID: 2, Name: "医生甲", Email: "doctor-overlap@example.com", Phone: "13800001002", Role: "doctor", Status: "active", PasswordHash: "hash"},
		otherDoctor:  models.User{ID: 3, Name: "医生乙", Email: "doctor-other@example.com", Phone: "13800001003", Role: "doctor", Status: "active", PasswordHash: "hash"},
		institution:  models.CheckupInstitution{ID: 10, Name: "主院区", Address: "健康路 1 号", Status: "active"},
		existingSlot: models.ScheduleSlot{ID: 20, DoctorID: 2, InstitutionID: 10, Date: "2026-07-01", Period: "上午", Category: "年度综合", StartTime: "09:00", EndTime: "09:30", Capacity: 1, BookedCount: 0, Status: "available"},
		laterSlot:    models.ScheduleSlot{ID: 21, DoctorID: 2, InstitutionID: 10, Date: "2026-07-01", Period: "上午", Category: "年度综合", StartTime: "10:00", EndTime: "10:30", Capacity: 1, BookedCount: 0, Status: "available"},
	}
	annualLink := models.InstitutionPackage{InstitutionID: fixture.institution.ID, PackageID: annualPackage.ID}
	entryLink := models.InstitutionPackage{InstitutionID: fixture.institution.ID, PackageID: entryPackage.ID}
	for _, row := range []any{&fixture.admin, &fixture.doctor, &fixture.otherDoctor, &fixture.institution, &annualPackage, &entryPackage, &annualLink, &entryLink, &fixture.existingSlot, &fixture.laterSlot} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}, db, fixture
}

func newScheduleSlotOverlapRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.POST("/schedule/slots", handler.createScheduleSlot)
	router.PUT("/schedule/slots/:id", handler.updateScheduleSlot)
	return router
}

func performCreateScheduleSlotRequest(t *testing.T, router *gin.Engine, body scheduleSlotRequest) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/schedule/slots", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func performUpdateScheduleSlotRequest(t *testing.T, router *gin.Engine, slotID uint, body scheduleSlotRequest) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/schedule/slots/"+strconv.Itoa(int(slotID)), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func assertScheduleSlotCount(t *testing.T, db *gorm.DB, doctorID uint, date string, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&models.ScheduleSlot{}).Where("doctor_id = ? AND date = ?", doctorID, date).Count(&count).Error; err != nil {
		t.Fatalf("count schedule slots: %v", err)
	}
	if count != want {
		t.Fatalf("expected %d schedule slots, got %d", want, count)
	}
}

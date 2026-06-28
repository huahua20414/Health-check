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

func TestScheduleSlotsFiltersByDoctorID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.CheckupInstitution{}, &models.ScheduleSlot{}, &models.WaitlistEntry{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	doctorA := models.User{ID: 1, Name: "医生甲", Phone: "13800000011", Email: "a@example.com", Role: "doctor", Status: "active"}
	doctorB := models.User{ID: 2, Name: "医生乙", Phone: "13800000012", Email: "b@example.com", Role: "doctor", Status: "active"}
	institution := models.CheckupInstitution{ID: 10, Name: "主院区", Status: "active"}
	slotA := models.ScheduleSlot{ID: 20, DoctorID: doctorA.ID, InstitutionID: institution.ID, Date: "2026-07-01", Period: "上午", Category: "年度综合", StartTime: "09:00", EndTime: "09:30", Capacity: 1, Status: "available"}
	slotB := models.ScheduleSlot{ID: 21, DoctorID: doctorB.ID, InstitutionID: institution.ID, Date: "2026-07-01", Period: "上午", Category: "年度综合", StartTime: "09:30", EndTime: "10:00", Capacity: 1, Status: "available"}
	for _, row := range []any{&doctorA, &doctorB, &institution, &slotA, &slotB} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create row %#v: %v", row, err)
		}
	}

	handler := &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}
	router := gin.New()
	router.GET("/schedule/slots", handler.scheduleSlots)

	req := httptest.NewRequest(http.MethodGet, "/schedule/slots?doctorId=1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var slots []models.ScheduleSlot
	if err := json.Unmarshal(rec.Body.Bytes(), &slots); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(slots) != 1 || slots[0].DoctorID != doctorA.ID {
		t.Fatalf("expected only doctor A slots, got %#v", slots)
	}
}

func TestScheduleSlotsOrdersNewestFirst(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.CheckupInstitution{}, &models.ScheduleSlot{}, &models.WaitlistEntry{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	doctor := models.User{ID: 1, Name: "医生甲", Phone: "13800000011", Email: "a@example.com", Role: "doctor", Status: "active"}
	institution := models.CheckupInstitution{ID: 10, Name: "主院区", Status: "active"}
	olderSlot := models.ScheduleSlot{ID: 20, DoctorID: doctor.ID, InstitutionID: institution.ID, Date: "2026-07-01", Period: "上午", Category: "年度综合", StartTime: "09:00", EndTime: "09:30", Capacity: 1, Status: "available"}
	newerSlot := models.ScheduleSlot{ID: 21, DoctorID: doctor.ID, InstitutionID: institution.ID, Date: "2026-06-01", Period: "上午", Category: "年度综合", StartTime: "08:00", EndTime: "08:30", Capacity: 1, Status: "available"}
	for _, row := range []any{&doctor, &institution, &olderSlot, &newerSlot} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create row %#v: %v", row, err)
		}
	}

	handler := &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}
	router := gin.New()
	router.GET("/schedule/slots", handler.scheduleSlots)

	req := httptest.NewRequest(http.MethodGet, "/schedule/slots", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var slots []models.ScheduleSlot
	if err := json.Unmarshal(rec.Body.Bytes(), &slots); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(slots) != 2 || slots[0].ID != newerSlot.ID || slots[1].ID != olderSlot.ID {
		t.Fatalf("expected newest slot first, got %#v", slots)
	}
}

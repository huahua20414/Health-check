package handlers

import (
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

func TestArchiveScheduleSlotRejectsBookedSlot(t *testing.T) {
	handler, db, fixture := newScheduleSlotArchiveFixture(t)
	router := newScheduleSlotArchiveRouter(handler, fixture.admin)

	response := performArchiveScheduleSlotRequest(t, router, fixture.bookedSlot.ID)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "schedule slot has booked appointments")
	assertScheduleSlotStatus(t, db, fixture.bookedSlot.ID, "available")
	assertScheduleSlotOperationLogCount(t, db, fixture.admin.ID, "archive", 0)
}

func TestArchiveScheduleSlotAllowsEmptySlot(t *testing.T) {
	handler, db, fixture := newScheduleSlotArchiveFixture(t)
	router := newScheduleSlotArchiveRouter(handler, fixture.admin)

	response := performArchiveScheduleSlotRequest(t, router, fixture.emptySlot.ID)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	assertScheduleSlotStatus(t, db, fixture.emptySlot.ID, "deleted")
	assertScheduleSlotOperationLogCount(t, db, fixture.admin.ID, "archive", 1)
}

type scheduleSlotArchiveFixture struct {
	admin       models.User
	doctor      models.User
	institution models.CheckupInstitution
	bookedSlot  models.ScheduleSlot
	emptySlot   models.ScheduleSlot
}

func newScheduleSlotArchiveFixture(t *testing.T) (*Handler, *gorm.DB, scheduleSlotArchiveFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.CheckupInstitution{}, &models.ScheduleSlot{}, &models.OperationLog{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	fixture := scheduleSlotArchiveFixture{
		admin:       models.User{ID: 1, Name: "管理员", Email: "admin@example.com", Phone: "13800000001", Role: "admin", Status: "active", PasswordHash: "hash"},
		doctor:      models.User{ID: 2, Name: "医生", Email: "doctor@example.com", Phone: "13800000002", Role: "doctor", Status: "active", PasswordHash: "hash"},
		institution: models.CheckupInstitution{ID: 10, Name: "主院区", Address: "健康路 1 号", Status: "active"},
		bookedSlot:  models.ScheduleSlot{ID: 20, DoctorID: 2, InstitutionID: 10, Date: "2026-07-01", Period: "上午", Category: "年度综合", StartTime: "09:00", EndTime: "09:30", Capacity: 2, BookedCount: 1, Status: "available"},
		emptySlot:   models.ScheduleSlot{ID: 21, DoctorID: 2, InstitutionID: 10, Date: "2026-07-01", Period: "上午", Category: "年度综合", StartTime: "10:00", EndTime: "10:30", Capacity: 2, BookedCount: 0, Status: "available"},
	}
	for _, row := range []any{&fixture.admin, &fixture.doctor, &fixture.institution, &fixture.bookedSlot, &fixture.emptySlot} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}, db, fixture
}

func newScheduleSlotArchiveRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.DELETE("/schedule/slots/:id", handler.archiveScheduleSlot)
	return router
}

func performArchiveScheduleSlotRequest(t *testing.T, router *gin.Engine, slotID uint) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, "/schedule/slots/"+strconv.Itoa(int(slotID)), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func assertScheduleSlotStatus(t *testing.T, db *gorm.DB, slotID uint, want string) {
	t.Helper()
	var slot models.ScheduleSlot
	if err := db.First(&slot, slotID).Error; err != nil {
		t.Fatalf("load slot: %v", err)
	}
	if slot.Status != want {
		t.Fatalf("expected slot %d status %s, got %s", slotID, want, slot.Status)
	}
}

func assertScheduleSlotOperationLogCount(t *testing.T, db *gorm.DB, userID uint, action string, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&models.OperationLog{}).Where("user_id = ? AND action = ? AND resource = ?", userID, action, "schedule_slot").Count(&count).Error; err != nil {
		t.Fatalf("count operation logs: %v", err)
	}
	if count != want {
		t.Fatalf("expected %d %s schedule slot operation logs, got %d", want, action, count)
	}
}

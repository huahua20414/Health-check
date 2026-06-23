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

func TestRescheduleAppointmentMovesBookingBetweenSlots(t *testing.T) {
	handler, db, fixture := newRescheduleFixture(t)
	router := newRescheduleTestRouter(handler, fixture.user)

	response := performRescheduleRequest(t, router, fixture.appointment.ID, rescheduleRequest{
		InstitutionID: fixture.institution.ID,
		SlotID:        fixture.newSlot.ID,
		Date:          fixture.newSlot.Date,
		Period:        fixture.newSlot.Period,
		Note:          "改到周三上午",
	})

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var updated models.Appointment
	if err := db.First(&updated, fixture.appointment.ID).Error; err != nil {
		t.Fatalf("load updated appointment: %v", err)
	}
	if updated.SlotID != fixture.newSlot.ID || updated.Date != fixture.newSlot.Date || updated.DoctorID != fixture.doctor.ID {
		t.Fatalf("appointment was not moved to new slot: %#v", updated)
	}
	if updated.Note != "改到周三上午" {
		t.Fatalf("expected note to be updated, got %q", updated.Note)
	}
	assertSlotBookedCount(t, db, fixture.oldSlot.ID, 0)
	assertSlotBookedCount(t, db, fixture.newSlot.ID, 1)
	assertNotificationCount(t, db, fixture.user.ID, 3)
}

func TestRescheduleAppointmentRejectsOtherUsersAppointment(t *testing.T) {
	handler, db, fixture := newRescheduleFixture(t)
	otherUser := models.User{ID: 200, Name: "其他用户", Role: "user", Status: "active", PasswordHash: "hash"}
	if err := db.Create(&otherUser).Error; err != nil {
		t.Fatalf("create other user: %v", err)
	}
	router := newRescheduleTestRouter(handler, otherUser)

	response := performRescheduleRequest(t, router, fixture.appointment.ID, rescheduleRequest{
		InstitutionID: fixture.institution.ID,
		SlotID:        fixture.newSlot.ID,
		Date:          fixture.newSlot.Date,
		Period:        fixture.newSlot.Period,
	})

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "appointment not found")
	assertSlotBookedCount(t, db, fixture.oldSlot.ID, 1)
	assertSlotBookedCount(t, db, fixture.newSlot.ID, 0)
}

func TestRescheduleAppointmentRejectsNonBookedAppointment(t *testing.T) {
	handler, db, fixture := newRescheduleFixture(t)
	if err := db.Model(&models.Appointment{}).Where("id = ?", fixture.appointment.ID).Update("status", "checked").Error; err != nil {
		t.Fatalf("mark appointment checked: %v", err)
	}
	router := newRescheduleTestRouter(handler, fixture.user)

	response := performRescheduleRequest(t, router, fixture.appointment.ID, rescheduleRequest{
		InstitutionID: fixture.institution.ID,
		SlotID:        fixture.newSlot.ID,
		Date:          fixture.newSlot.Date,
		Period:        fixture.newSlot.Period,
	})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "only booked appointments can be rescheduled")
	assertSlotBookedCount(t, db, fixture.oldSlot.ID, 1)
	assertSlotBookedCount(t, db, fixture.newSlot.ID, 0)
}

func TestRescheduleAppointmentRejectsFullTargetSlot(t *testing.T) {
	handler, db, fixture := newRescheduleFixture(t)
	if err := db.Model(&models.ScheduleSlot{}).Where("id = ?", fixture.newSlot.ID).Update("booked_count", 1).Error; err != nil {
		t.Fatalf("fill target slot: %v", err)
	}
	router := newRescheduleTestRouter(handler, fixture.user)

	response := performRescheduleRequest(t, router, fixture.appointment.ID, rescheduleRequest{
		InstitutionID: fixture.institution.ID,
		SlotID:        fixture.newSlot.ID,
		Date:          fixture.newSlot.Date,
		Period:        fixture.newSlot.Period,
	})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "no available slot for reschedule")
	assertSlotBookedCount(t, db, fixture.oldSlot.ID, 1)
	assertSlotBookedCount(t, db, fixture.newSlot.ID, 1)
}

type rescheduleFixture struct {
	user        models.User
	doctor      models.User
	institution models.CheckupInstitution
	pkg         models.CheckupPackage
	oldSlot     models.ScheduleSlot
	newSlot     models.ScheduleSlot
	appointment models.Appointment
}

func newRescheduleFixture(t *testing.T) (*Handler, *gorm.DB, rescheduleFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.CheckupInstitution{},
		&models.CheckupPackage{},
		&models.ScheduleSlot{},
		&models.Appointment{},
		&models.Notification{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	fixture := rescheduleFixture{
		user:        models.User{ID: 100, Name: "用户", Phone: "13800000100", Email: "user@example.com", Role: "user", Status: "active", PasswordHash: "hash"},
		doctor:      models.User{ID: 300, Name: "医生", Phone: "13800000300", Email: "doctor@example.com", Role: "doctor", Status: "active", PasswordHash: "hash"},
		institution: models.CheckupInstitution{ID: 10, Name: "主院区", Address: "健康路 1 号", Status: "active"},
		pkg:         models.CheckupPackage{ID: 20, Name: "年度体检", Category: "年度综合", Price: 399, Items: "血常规", Status: "active"},
		oldSlot:     models.ScheduleSlot{ID: 30, DoctorID: 300, InstitutionID: 10, Date: "2026-07-01", Period: "上午", Category: "年度综合", StartTime: "09:00", EndTime: "09:30", Capacity: 1, BookedCount: 1, Status: "available"},
		newSlot:     models.ScheduleSlot{ID: 31, DoctorID: 300, InstitutionID: 10, Date: "2026-07-03", Period: "上午", Category: "年度综合", StartTime: "10:00", EndTime: "10:30", Capacity: 1, BookedCount: 0, Status: "available"},
		appointment: models.Appointment{ID: 40, OrderNo: "HC202607010001", UserID: 100, DoctorID: 300, InstitutionID: 10, SlotID: 30, PackageID: 20, AppointmentType: "个人体检", Category: "年度综合", Date: "2026-07-01", Period: "上午", StartTime: "09:00", EndTime: "09:30", Status: "booked", PaymentStatus: "unpaid"},
	}
	for _, row := range []any{&fixture.user, &fixture.doctor, &fixture.institution, &fixture.pkg, &fixture.oldSlot, &fixture.newSlot, &fixture.appointment} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}
	handler := &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}
	return handler, db, fixture
}

func newRescheduleTestRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.PATCH("/appointments/:id/reschedule", func(c *gin.Context) {
		c.Set("user", current)
		handler.rescheduleAppointment(c)
	})
	return router
}

func performRescheduleRequest(t *testing.T, router *gin.Engine, appointmentID uint, req rescheduleRequest) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	httpReq := httptest.NewRequest(http.MethodPatch, "/appointments/"+strconv.Itoa(int(appointmentID))+"/reschedule", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httpReq)
	return rec
}

func assertSlotBookedCount(t *testing.T, db *gorm.DB, slotID uint, want int) {
	t.Helper()
	var slot models.ScheduleSlot
	if err := db.First(&slot, slotID).Error; err != nil {
		t.Fatalf("load slot %d: %v", slotID, err)
	}
	if slot.BookedCount != want {
		t.Fatalf("expected slot %d booked count %d, got %d", slotID, want, slot.BookedCount)
	}
}

func assertNotificationCount(t *testing.T, db *gorm.DB, userID uint, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&models.Notification{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	if count != want {
		t.Fatalf("expected %d notifications, got %d", want, count)
	}
}

func assertErrorMessage(t *testing.T, body []byte, want string) {
	t.Helper()
	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode error response %q: %v", string(body), err)
	}
	if response["error"] != want {
		t.Fatalf("expected error %q, got %#v", want, response)
	}
}

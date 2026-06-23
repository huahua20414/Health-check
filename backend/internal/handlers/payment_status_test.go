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

func TestUpdateAppointmentPaymentMarksOwnBookedAppointmentPaid(t *testing.T) {
	handler, db, fixture := newPaymentStatusFixture(t)
	router := newPaymentStatusRouter(handler, fixture.user)

	response := performPaymentStatusRequest(t, router, fixture.bookedAppointment.ID, paymentStatusRequest{PaymentStatus: "paid"})

	appointment := decodePaymentAppointment(t, response)
	if appointment.PaymentStatus != "paid" || appointment.ID != fixture.bookedAppointment.ID {
		t.Fatalf("unexpected payment appointment: %#v", appointment)
	}
	assertAppointmentPaymentStatus(t, db, fixture.bookedAppointment.ID, "paid")
	assertPaymentNotificationCount(t, db, fixture.user.ID, 1)
	assertAppointmentOperationLog(t, db, fixture.user.ID, fixture.bookedAppointment.ID, "update_payment", "paid")
}

func TestUpdateAppointmentPaymentRejectsOtherUsersAppointment(t *testing.T) {
	handler, db, fixture := newPaymentStatusFixture(t)
	router := newPaymentStatusRouter(handler, fixture.user)

	response := performPaymentStatusRequest(t, router, fixture.otherAppointment.ID, paymentStatusRequest{PaymentStatus: "paid"})

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
	assertAppointmentPaymentStatus(t, db, fixture.otherAppointment.ID, "unpaid")
}

func TestUpdateAppointmentPaymentRejectsFinishedAppointment(t *testing.T) {
	handler, db, fixture := newPaymentStatusFixture(t)
	router := newPaymentStatusRouter(handler, fixture.user)

	response := performPaymentStatusRequest(t, router, fixture.reportedAppointment.ID, paymentStatusRequest{PaymentStatus: "paid"})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "payment can only be updated for booked appointments")
	assertAppointmentPaymentStatus(t, db, fixture.reportedAppointment.ID, "unpaid")
}

func TestUpdateAppointmentPaymentRejectsInvalidPaymentStatus(t *testing.T) {
	handler, _, fixture := newPaymentStatusFixture(t)
	router := newPaymentStatusRouter(handler, fixture.user)

	response := performPaymentStatusRequest(t, router, fixture.bookedAppointment.ID, paymentStatusRequest{PaymentStatus: "refunded"})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "invalid payment status")
}

func TestUpdateAppointmentPaymentRespectsInAppToggle(t *testing.T) {
	handler, db, fixture := newPaymentStatusFixture(t)
	if err := db.Model(&models.SystemSetting{}).Where("key = ?", "notification.in_app_enabled").Update("value", "false").Error; err != nil {
		t.Fatalf("disable in app setting: %v", err)
	}
	router := newPaymentStatusRouter(handler, fixture.user)

	response := performPaymentStatusRequest(t, router, fixture.bookedAppointment.ID, paymentStatusRequest{PaymentStatus: "paid"})

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	assertPaymentNotificationCount(t, db, fixture.user.ID, 0)
}

type paymentStatusFixture struct {
	user                models.User
	otherUser           models.User
	doctor              models.User
	institution         models.CheckupInstitution
	pkg                 models.CheckupPackage
	slot                models.ScheduleSlot
	bookedAppointment   models.Appointment
	otherAppointment    models.Appointment
	reportedAppointment models.Appointment
}

func newPaymentStatusFixture(t *testing.T) (*Handler, *gorm.DB, paymentStatusFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.CheckupInstitution{}, &models.CheckupPackage{}, &models.ScheduleSlot{}, &models.Appointment{}, &models.Notification{}, &models.SystemSetting{}, &models.OperationLog{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	fixture := paymentStatusFixture{
		user:                models.User{ID: 100, Name: "用户", Phone: "13800000100", Email: "user@example.com", Role: "user", Status: "active", PasswordHash: "hash"},
		otherUser:           models.User{ID: 101, Name: "其他用户", Phone: "13800000101", Email: "other@example.com", Role: "user", Status: "active", PasswordHash: "hash"},
		doctor:              models.User{ID: 200, Name: "医生", Phone: "13800000200", Email: "doctor@example.com", Role: "doctor", Status: "active", PasswordHash: "hash"},
		institution:         models.CheckupInstitution{ID: 10, Name: "主院区", Address: "健康路 1 号", Status: "active"},
		pkg:                 models.CheckupPackage{ID: 20, Name: "年度体检", Category: "年度综合", Price: 399, Items: "血常规", Status: "active"},
		slot:                models.ScheduleSlot{ID: 30, DoctorID: 200, InstitutionID: 10, Date: "2026-07-01", Period: "上午", Category: "年度综合", StartTime: "09:00", EndTime: "09:30", Capacity: 3, BookedCount: 3, Status: "available"},
		bookedAppointment:   models.Appointment{ID: 40, OrderNo: "HC202607010001", UserID: 100, DoctorID: 200, InstitutionID: 10, SlotID: 30, PackageID: 20, AppointmentType: "个人体检", Category: "年度综合", Date: "2026-07-01", Period: "上午", StartTime: "09:00", EndTime: "09:30", Status: "booked", PaymentStatus: "unpaid"},
		otherAppointment:    models.Appointment{ID: 41, OrderNo: "HC202607010002", UserID: 101, DoctorID: 200, InstitutionID: 10, SlotID: 30, PackageID: 20, AppointmentType: "个人体检", Category: "年度综合", Date: "2026-07-01", Period: "上午", StartTime: "09:30", EndTime: "10:00", Status: "booked", PaymentStatus: "unpaid"},
		reportedAppointment: models.Appointment{ID: 42, OrderNo: "HC202607010003", UserID: 100, DoctorID: 200, InstitutionID: 10, SlotID: 30, PackageID: 20, AppointmentType: "个人体检", Category: "年度综合", Date: "2026-07-01", Period: "上午", StartTime: "10:00", EndTime: "10:30", Status: "reported", PaymentStatus: "unpaid"},
	}
	inAppSetting := models.SystemSetting{Key: "notification.in_app_enabled", Value: "true", ValueType: "boolean", Group: "notification", Label: "站内信通知", Status: "active"}
	for _, row := range []any{&fixture.user, &fixture.otherUser, &fixture.doctor, &fixture.institution, &fixture.pkg, &fixture.slot, &fixture.bookedAppointment, &fixture.otherAppointment, &fixture.reportedAppointment, &inAppSetting} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}, db, fixture
}

func newPaymentStatusRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.PATCH("/appointments/:id/payment", handler.updateAppointmentPayment)
	return router
}

func performPaymentStatusRequest(t *testing.T, router *gin.Engine, appointmentID uint, req paymentStatusRequest) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	httpReq := httptest.NewRequest(http.MethodPatch, "/appointments/"+strconv.Itoa(int(appointmentID))+"/payment", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httpReq)
	return rec
}

func decodePaymentAppointment(t *testing.T, response *httptest.ResponseRecorder) models.Appointment {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var appointment models.Appointment
	if err := json.Unmarshal(response.Body.Bytes(), &appointment); err != nil {
		t.Fatalf("decode appointment: %v", err)
	}
	return appointment
}

func assertAppointmentPaymentStatus(t *testing.T, db *gorm.DB, appointmentID uint, want string) {
	t.Helper()
	var appointment models.Appointment
	if err := db.First(&appointment, appointmentID).Error; err != nil {
		t.Fatalf("load appointment: %v", err)
	}
	if appointment.PaymentStatus != want {
		t.Fatalf("expected payment status %s, got %s", want, appointment.PaymentStatus)
	}
}

func assertPaymentNotificationCount(t *testing.T, db *gorm.DB, userID uint, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&models.Notification{}).Where("user_id = ? AND type = ?", userID, "payment_status").Count(&count).Error; err != nil {
		t.Fatalf("count payment notifications: %v", err)
	}
	if count != want {
		t.Fatalf("expected payment notification count %d, got %d", want, count)
	}
}

func assertAppointmentOperationLog(t *testing.T, db *gorm.DB, userID, appointmentID uint, action, detail string) {
	t.Helper()
	var count int64
	if err := db.Model(&models.OperationLog{}).
		Where("user_id = ? AND action = ? AND resource = ? AND resource_id = ? AND detail = ?", userID, action, "appointment", strconv.Itoa(int(appointmentID)), detail).
		Count(&count).Error; err != nil {
		t.Fatalf("count appointment operation log: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one %s operation log with detail %q, got %d", action, detail, count)
	}
}

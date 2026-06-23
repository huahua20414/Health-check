package handlers

import (
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

func TestCancelAppointmentRefundsPaidAppointment(t *testing.T) {
	handler, db, fixture := newAppointmentCancelFixture(t)
	router := newAppointmentCancelRouter(handler, fixture.user)

	response := performCancelAppointmentRequest(t, router, fixture.paidAppointment.ID)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	assertCanceledAppointmentState(t, db, fixture.paidAppointment.ID, "refunded")
	assertCancelSlotBookedCount(t, db, fixture.slot.ID, 1)
	assertCancellationNotification(t, db, fixture.user.ID, "模拟退款")
	assertCancelOperationLog(t, db, fixture.user.ID, fixture.paidAppointment.ID, "payment=refunded")
}

func TestCancelAppointmentKeepsUnpaidAppointmentUnpaid(t *testing.T) {
	handler, db, fixture := newAppointmentCancelFixture(t)
	router := newAppointmentCancelRouter(handler, fixture.user)

	response := performCancelAppointmentRequest(t, router, fixture.unpaidAppointment.ID)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	assertCanceledAppointmentState(t, db, fixture.unpaidAppointment.ID, "unpaid")
	assertCancellationNotification(t, db, fixture.user.ID, "已取消")
	assertCancelOperationLog(t, db, fixture.user.ID, fixture.unpaidAppointment.ID, "payment=unpaid")
}

func TestCancelAppointmentRejectsOtherUsersAppointment(t *testing.T) {
	handler, db, fixture := newAppointmentCancelFixture(t)
	router := newAppointmentCancelRouter(handler, fixture.user)

	response := performCancelAppointmentRequest(t, router, fixture.otherAppointment.ID)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
	assertAppointmentStatusAndPayment(t, db, fixture.otherAppointment.ID, "booked", "paid")
}

type appointmentCancelFixture struct {
	user              models.User
	otherUser         models.User
	doctor            models.User
	institution       models.CheckupInstitution
	pkg               models.CheckupPackage
	slot              models.ScheduleSlot
	paidAppointment   models.Appointment
	unpaidAppointment models.Appointment
	otherAppointment  models.Appointment
}

func newAppointmentCancelFixture(t *testing.T) (*Handler, *gorm.DB, appointmentCancelFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.CheckupInstitution{}, &models.CheckupPackage{}, &models.ScheduleSlot{}, &models.Appointment{}, &models.WaitlistEntry{}, &models.Notification{}, &models.OperationLog{}, &models.SystemSetting{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	fixture := appointmentCancelFixture{
		user:              models.User{ID: 100, Name: "用户", Phone: "13800000100", Email: "user@example.com", Role: "user", Status: "active", PasswordHash: "hash"},
		otherUser:         models.User{ID: 101, Name: "其他用户", Phone: "13800000101", Email: "other@example.com", Role: "user", Status: "active", PasswordHash: "hash"},
		doctor:            models.User{ID: 200, Name: "医生", Phone: "13800000200", Email: "doctor@example.com", Role: "doctor", Status: "active", PasswordHash: "hash"},
		institution:       models.CheckupInstitution{ID: 10, Name: "主院区", Address: "健康路 1 号", Status: "active"},
		pkg:               models.CheckupPackage{ID: 20, Name: "年度体检", Category: "年度综合", Price: 399, Items: "血常规", Status: "active"},
		slot:              models.ScheduleSlot{ID: 30, DoctorID: 200, InstitutionID: 10, Date: "2026-07-01", Period: "上午", Category: "年度综合", StartTime: "09:00", EndTime: "09:30", Capacity: 3, BookedCount: 2, Status: "available"},
		paidAppointment:   models.Appointment{ID: 40, OrderNo: "HCCANCEL001", UserID: 100, DoctorID: 200, InstitutionID: 10, SlotID: 30, PackageID: 20, AppointmentType: "个人体检", Category: "年度综合", Date: "2026-07-01", Period: "上午", StartTime: "09:00", EndTime: "09:30", Status: "booked", PaymentStatus: "paid", OriginalAmount: 399, PayableAmount: 399},
		unpaidAppointment: models.Appointment{ID: 41, OrderNo: "HCCANCEL002", UserID: 100, DoctorID: 200, InstitutionID: 10, SlotID: 0, PackageID: 20, AppointmentType: "个人体检", Category: "年度综合", Date: "2026-07-02", Period: "上午", StartTime: "09:30", EndTime: "10:00", Status: "booked", PaymentStatus: "unpaid", OriginalAmount: 399, PayableAmount: 399},
		otherAppointment:  models.Appointment{ID: 42, OrderNo: "HCCANCEL003", UserID: 101, DoctorID: 200, InstitutionID: 10, SlotID: 0, PackageID: 20, AppointmentType: "个人体检", Category: "年度综合", Date: "2026-07-02", Period: "上午", StartTime: "10:00", EndTime: "10:30", Status: "booked", PaymentStatus: "paid", OriginalAmount: 399, PayableAmount: 399},
	}
	inAppSetting := models.SystemSetting{Key: "notification.in_app_enabled", Value: "true", ValueType: "boolean", Group: "notification", Label: "站内信通知", Status: "active"}
	for _, row := range []any{&fixture.user, &fixture.otherUser, &fixture.doctor, &fixture.institution, &fixture.pkg, &fixture.slot, &fixture.paidAppointment, &fixture.unpaidAppointment, &fixture.otherAppointment, &inAppSetting} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}, db, fixture
}

func newAppointmentCancelRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.PATCH("/appointments/:id/cancel", handler.cancelAppointment)
	return router
}

func performCancelAppointmentRequest(t *testing.T, router *gin.Engine, appointmentID uint) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, "/appointments/"+strconv.Itoa(int(appointmentID))+"/cancel", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func assertCanceledAppointmentState(t *testing.T, db *gorm.DB, appointmentID uint, paymentStatus string) {
	t.Helper()
	assertAppointmentStatusAndPayment(t, db, appointmentID, "canceled", paymentStatus)
}

func assertAppointmentStatusAndPayment(t *testing.T, db *gorm.DB, appointmentID uint, status, paymentStatus string) {
	t.Helper()
	var appointment models.Appointment
	if err := db.First(&appointment, appointmentID).Error; err != nil {
		t.Fatalf("load appointment: %v", err)
	}
	if appointment.Status != status || appointment.PaymentStatus != paymentStatus {
		t.Fatalf("expected appointment %d to be %s/%s, got %s/%s", appointmentID, status, paymentStatus, appointment.Status, appointment.PaymentStatus)
	}
}

func assertCancelSlotBookedCount(t *testing.T, db *gorm.DB, slotID uint, want int) {
	t.Helper()
	var slot models.ScheduleSlot
	if err := db.First(&slot, slotID).Error; err != nil {
		t.Fatalf("load slot: %v", err)
	}
	if slot.BookedCount != want {
		t.Fatalf("expected slot booked count %d, got %d", want, slot.BookedCount)
	}
}

func assertCancellationNotification(t *testing.T, db *gorm.DB, userID uint, contentContains string) {
	t.Helper()
	var notification models.Notification
	if err := db.Where("user_id = ? AND type = ?", userID, "appointment_canceled").Order("id desc").First(&notification).Error; err != nil {
		t.Fatalf("load cancellation notification: %v", err)
	}
	if !strings.Contains(notification.Content, contentContains) {
		t.Fatalf("expected notification content to contain %q, got %q", contentContains, notification.Content)
	}
}

func assertCancelOperationLog(t *testing.T, db *gorm.DB, userID, appointmentID uint, detail string) {
	t.Helper()
	var count int64
	if err := db.Model(&models.OperationLog{}).Where("user_id = ? AND action = ? AND resource = ? AND resource_id = ? AND detail = ?", userID, "cancel", "appointment", strconv.Itoa(int(appointmentID)), detail).Count(&count).Error; err != nil {
		t.Fatalf("count cancel operation log: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one cancel operation log with detail %q, got %d", detail, count)
	}
}

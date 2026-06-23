package handlers

import (
	"bytes"
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

func TestCreateAppointmentSnapshotsCouponPricing(t *testing.T) {
	handler, db, fixture := newAppointmentCreateFixture(t)
	router := newAppointmentCreateRouter(handler, fixture.user)

	response := performCreateAppointmentRequest(t, router, appointmentRequest{
		PackageID:       fixture.pkg.ID,
		InstitutionID:   fixture.institution.ID,
		SlotID:          fixture.slot.ID,
		CouponID:        fixture.amountCoupon.ID,
		AppointmentType: "个人体检",
		Date:            fixture.slot.Date,
		Period:          fixture.slot.Period,
		PaymentStatus:   "paid",
	})

	appointment := decodeCreateAppointmentResponse(t, response)
	if appointment.OriginalAmount != 399 || appointment.DiscountAmount != 50 || appointment.PayableAmount != 349 || appointment.CouponID != fixture.amountCoupon.ID {
		t.Fatalf("unexpected pricing snapshot: %#v", appointment)
	}
	assertCreatedAppointmentPricing(t, db, appointment.ID, 399, 50, 349)
}

func TestCreateAppointmentRejectsCouponBelowMinimumAmount(t *testing.T) {
	handler, _, fixture := newAppointmentCreateFixture(t)
	router := newAppointmentCreateRouter(handler, fixture.user)

	response := performCreateAppointmentRequest(t, router, appointmentRequest{
		PackageID:       fixture.pkg.ID,
		InstitutionID:   fixture.institution.ID,
		SlotID:          fixture.slot.ID,
		CouponID:        fixture.highMinimumCoupon.ID,
		AppointmentType: "个人体检",
		Date:            fixture.slot.Date,
		Period:          fixture.slot.Period,
	})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "coupon minimum amount not reached")
}

func TestCreateAppointmentRejectsExpiredCoupon(t *testing.T) {
	handler, _, fixture := newAppointmentCreateFixture(t)
	router := newAppointmentCreateRouter(handler, fixture.user)

	response := performCreateAppointmentRequest(t, router, appointmentRequest{
		PackageID:       fixture.pkg.ID,
		InstitutionID:   fixture.institution.ID,
		SlotID:          fixture.slot.ID,
		CouponID:        fixture.expiredCoupon.ID,
		AppointmentType: "个人体检",
		Date:            fixture.slot.Date,
		Period:          fixture.slot.Period,
	})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "coupon is expired")
}

func TestCreateAppointmentRejectsPackageSpecificCouponMismatch(t *testing.T) {
	handler, _, fixture := newAppointmentCreateFixture(t)
	router := newAppointmentCreateRouter(handler, fixture.user)

	response := performCreateAppointmentRequest(t, router, appointmentRequest{
		PackageID:       fixture.pkg.ID,
		InstitutionID:   fixture.institution.ID,
		SlotID:          fixture.slot.ID,
		CouponID:        fixture.otherPackageCoupon.ID,
		AppointmentType: "个人体检",
		Date:            fixture.slot.Date,
		Period:          fixture.slot.Period,
	})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "coupon does not apply to this package")
}

func TestCreateAppointmentWaitlistCreatesNotifications(t *testing.T) {
	handler, db, fixture := newAppointmentCreateFixture(t)
	if err := db.Model(&models.ScheduleSlot{}).Where("id = ?", fixture.slot.ID).Update("booked_count", fixture.slot.Capacity).Error; err != nil {
		t.Fatalf("fill slot: %v", err)
	}
	router := newAppointmentCreateRouter(handler, fixture.user)

	response := performCreateAppointmentRequest(t, router, appointmentRequest{
		PackageID:       fixture.pkg.ID,
		InstitutionID:   fixture.institution.ID,
		SlotID:          fixture.slot.ID,
		AppointmentType: "个人体检",
		Date:            fixture.slot.Date,
		Period:          fixture.slot.Period,
	})

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Type     string               `json:"type"`
		Waitlist models.WaitlistEntry `json:"waitlist"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode waitlist response: %v", err)
	}
	if payload.Type != "waitlist" || payload.Waitlist.Status != "waiting" {
		t.Fatalf("expected waiting waitlist response, got %#v", payload)
	}
	assertCreateNotificationChannelCount(t, db, fixture.user.ID, "waitlist_joined", "in_app", 1)
	assertCreateNotificationChannelCount(t, db, fixture.user.ID, "waitlist_joined", "sms_mock", 1)
}

type appointmentCreateFixture struct {
	user               models.User
	doctor             models.User
	institution        models.CheckupInstitution
	pkg                models.CheckupPackage
	otherPackage       models.CheckupPackage
	slot               models.ScheduleSlot
	amountCoupon       models.Coupon
	highMinimumCoupon  models.Coupon
	expiredCoupon      models.Coupon
	otherPackageCoupon models.Coupon
}

func newAppointmentCreateFixture(t *testing.T) (*Handler, *gorm.DB, appointmentCreateFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sqlite db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&models.User{}, &models.CheckupInstitution{}, &models.CheckupPackage{}, &models.ScheduleSlot{}, &models.Appointment{}, &models.WaitlistEntry{}, &models.Coupon{}, &models.Notification{}, &models.MailLog{}, &models.SystemSetting{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	fixture := appointmentCreateFixture{
		user:               models.User{ID: 100, Name: "用户", Phone: "13800000100", Email: "user@example.com", Role: "user", Status: "active", PasswordHash: "hash"},
		doctor:             models.User{ID: 200, Name: "医生", Phone: "13800000200", Email: "doctor@example.com", Role: "doctor", Status: "active", PasswordHash: "hash"},
		institution:        models.CheckupInstitution{ID: 10, Name: "主院区", Address: "健康路 1 号", Status: "active"},
		pkg:                models.CheckupPackage{ID: 20, Name: "年度体检", Category: "年度综合", Price: 399, Items: "血常规", Status: "active"},
		otherPackage:       models.CheckupPackage{ID: 21, Name: "专项体检", Category: "影像专项", Price: 199, Items: "CT", Status: "active"},
		slot:               models.ScheduleSlot{ID: 30, DoctorID: 200, InstitutionID: 10, Date: "2026-07-01", Period: "上午", Category: "年度综合", StartTime: "09:00", EndTime: "09:30", Capacity: 3, BookedCount: 0, Status: "available"},
		amountCoupon:       models.Coupon{ID: 40, Name: "立减券", Code: "SAVE50", Type: "amount", Value: 50, MinAmount: 100, Status: "active", StartDate: "2026-01-01", EndDate: "2026-12-31"},
		highMinimumCoupon:  models.Coupon{ID: 41, Name: "高门槛券", Code: "SAVE500", Type: "amount", Value: 100, MinAmount: 500, Status: "active", StartDate: "2026-01-01", EndDate: "2026-12-31"},
		expiredCoupon:      models.Coupon{ID: 42, Name: "过期券", Code: "OLD50", Type: "amount", Value: 50, MinAmount: 0, Status: "active", StartDate: "2025-01-01", EndDate: "2025-12-31"},
		otherPackageCoupon: models.Coupon{ID: 43, Name: "专项券", Code: "IMG20", Type: "amount", Value: 20, MinAmount: 0, PackageID: 21, Status: "active", StartDate: "2026-01-01", EndDate: "2026-12-31"},
	}
	inAppSetting := models.SystemSetting{Key: "notification.in_app_enabled", Value: "true", ValueType: "boolean", Group: "notification", Label: "站内信通知", Status: "active"}
	smsSetting := models.SystemSetting{Key: "notification.sms_mock_enabled", Value: "true", ValueType: "boolean", Group: "notification", Label: "短信模拟通知", Status: "active"}
	for _, row := range []any{&fixture.user, &fixture.doctor, &fixture.institution, &fixture.pkg, &fixture.otherPackage, &fixture.slot, &fixture.amountCoupon, &fixture.highMinimumCoupon, &fixture.expiredCoupon, &fixture.otherPackageCoupon, &inAppSetting, &smsSetting} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}, db, fixture
}

func newAppointmentCreateRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.POST("/appointments", handler.createAppointment)
	return router
}

func performCreateAppointmentRequest(t *testing.T, router *gin.Engine, req appointmentRequest) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	httpReq := httptest.NewRequest(http.MethodPost, "/appointments", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httpReq)
	return rec
}

func decodeCreateAppointmentResponse(t *testing.T, response *httptest.ResponseRecorder) models.Appointment {
	t.Helper()
	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Type        string             `json:"type"`
		Appointment models.Appointment `json:"appointment"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode appointment response: %v", err)
	}
	if payload.Type != "appointment" {
		t.Fatalf("expected appointment response, got %#v", payload)
	}
	return payload.Appointment
}

func assertCreatedAppointmentPricing(t *testing.T, db *gorm.DB, appointmentID uint, original, discount, payable float64) {
	t.Helper()
	var appointment models.Appointment
	if err := db.First(&appointment, appointmentID).Error; err != nil {
		t.Fatalf("load appointment: %v", err)
	}
	if appointment.OriginalAmount != original || appointment.DiscountAmount != discount || appointment.PayableAmount != payable {
		t.Fatalf("expected pricing %.2f/%.2f/%.2f, got %.2f/%.2f/%.2f", original, discount, payable, appointment.OriginalAmount, appointment.DiscountAmount, appointment.PayableAmount)
	}
}

func assertCreateNotificationChannelCount(t *testing.T, db *gorm.DB, userID uint, kind, channel string, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&models.Notification{}).Where("user_id = ? AND type = ? AND channel = ?", userID, kind, channel).Count(&count).Error; err != nil {
		t.Fatalf("count %s notifications: %v", kind, err)
	}
	if count != want {
		t.Fatalf("expected %d %s/%s notifications, got %d", want, kind, channel, count)
	}
}

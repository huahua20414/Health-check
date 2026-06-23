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

func TestDoctorCanMarkOwnBookedAppointmentChecked(t *testing.T) {
	handler, db, fixture := newAppointmentStatusFixture(t)
	router := newAppointmentStatusRouter(handler, fixture.doctor)

	response := performAppointmentStatusPatch(t, router, fixture.bookedAppointment.ID, statusRequest{Status: "checked"})

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	assertAppointmentStatus(t, db, fixture.bookedAppointment.ID, "checked")
	assertOperationCount(t, db, "update_status", "appointment", 1)
}

func TestDoctorCannotUpdateOtherDoctorsAppointment(t *testing.T) {
	handler, db, fixture := newAppointmentStatusFixture(t)
	router := newAppointmentStatusRouter(handler, fixture.otherDoctor)

	response := performAppointmentStatusPatch(t, router, fixture.bookedAppointment.ID, statusRequest{Status: "checked"})

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", response.Code, response.Body.String())
	}
	assertAppointmentStatus(t, db, fixture.bookedAppointment.ID, "booked")
}

func TestAppointmentStatusRejectsInvalidTransition(t *testing.T) {
	handler, db, fixture := newAppointmentStatusFixture(t)
	router := newAppointmentStatusRouter(handler, fixture.admin)

	response := performAppointmentStatusPatch(t, router, fixture.reportedAppointment.ID, statusRequest{Status: "checked"})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertAppointmentStatus(t, db, fixture.reportedAppointment.ID, "reported")
}

func TestAppointmentStatusRejectsCanceledAppointmentTransition(t *testing.T) {
	handler, db, fixture := newAppointmentStatusFixture(t)
	router := newAppointmentStatusRouter(handler, fixture.admin)

	response := performAppointmentStatusPatch(t, router, fixture.canceledAppointment.ID, statusRequest{Status: "checked"})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertAppointmentStatus(t, db, fixture.canceledAppointment.ID, "canceled")
}

type appointmentStatusFixture struct {
	admin               models.User
	user                models.User
	doctor              models.User
	otherDoctor         models.User
	institution         models.CheckupInstitution
	pkg                 models.CheckupPackage
	bookedAppointment   models.Appointment
	reportedAppointment models.Appointment
	canceledAppointment models.Appointment
}

func newAppointmentStatusFixture(t *testing.T) (*Handler, *gorm.DB, appointmentStatusFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.CheckupInstitution{}, &models.CheckupPackage{}, &models.Appointment{}, &models.OperationLog{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	fixture := appointmentStatusFixture{
		admin:               models.User{ID: 1, Name: "管理员", Email: "admin@example.com", Phone: "13800000001", Role: "admin", Status: "active", PasswordHash: "hash"},
		user:                models.User{ID: 2, Name: "用户", Email: "user@example.com", Phone: "13800000002", Role: "user", Status: "active", PasswordHash: "hash"},
		doctor:              models.User{ID: 3, Name: "医生甲", Email: "doctor@example.com", Phone: "13800000003", Role: "doctor", Status: "active", PasswordHash: "hash"},
		otherDoctor:         models.User{ID: 4, Name: "医生乙", Email: "doctor2@example.com", Phone: "13800000004", Role: "doctor", Status: "active", PasswordHash: "hash"},
		institution:         models.CheckupInstitution{ID: 10, Name: "主院区", Address: "健康路 1 号", Status: "active"},
		pkg:                 models.CheckupPackage{ID: 20, Name: "年度套餐", Category: "年度综合", Price: 399, Status: "active"},
		bookedAppointment:   models.Appointment{ID: 30, OrderNo: "HCSTATUS001", UserID: 2, DoctorID: 3, InstitutionID: 10, PackageID: 20, AppointmentType: "个人体检", Category: "年度综合", Date: "2026-07-01", Period: "上午", Status: "booked"},
		reportedAppointment: models.Appointment{ID: 31, OrderNo: "HCSTATUS002", UserID: 2, DoctorID: 3, InstitutionID: 10, PackageID: 20, AppointmentType: "个人体检", Category: "年度综合", Date: "2026-07-02", Period: "上午", Status: "reported"},
		canceledAppointment: models.Appointment{ID: 32, OrderNo: "HCSTATUS003", UserID: 2, DoctorID: 3, InstitutionID: 10, PackageID: 20, AppointmentType: "个人体检", Category: "年度综合", Date: "2026-07-03", Period: "上午", Status: "canceled"},
	}
	for _, row := range []any{&fixture.admin, &fixture.user, &fixture.doctor, &fixture.otherDoctor, &fixture.institution, &fixture.pkg, &fixture.bookedAppointment, &fixture.reportedAppointment, &fixture.canceledAppointment} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}, db, fixture
}

func newAppointmentStatusRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.PATCH("/appointments/:id/status", handler.updateAppointmentStatus)
	return router
}

func performAppointmentStatusPatch(t *testing.T, router *gin.Engine, id uint, body statusRequest) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/appointments/"+strconv.Itoa(int(id))+"/status", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func assertAppointmentStatus(t *testing.T, db *gorm.DB, id uint, want string) {
	t.Helper()
	var appointment models.Appointment
	if err := db.First(&appointment, id).Error; err != nil {
		t.Fatalf("load appointment: %v", err)
	}
	if appointment.Status != want {
		t.Fatalf("expected appointment %d status %s, got %s", id, want, appointment.Status)
	}
}

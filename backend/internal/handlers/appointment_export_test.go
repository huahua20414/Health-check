package handlers

import (
	"encoding/csv"
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

func TestExportAppointmentsScopesDoctorRows(t *testing.T) {
	handler, db, fixture := newAppointmentExportFixture(t)
	router := newAppointmentExportRouter(handler, fixture.doctor)

	response := performAppointmentExportRequest(t, router, "/appointments/export")

	records := decodeAppointmentExportCSV(t, response)
	if len(records) != 2 {
		t.Fatalf("expected header plus one doctor row, got %#v", records)
	}
	if records[1][0] != fixture.doctorAppointment.OrderNo || records[1][3] != fixture.doctor.Name {
		t.Fatalf("doctor export returned wrong row: %#v", records)
	}
	if records[1][14] != "399.00" || records[1][15] != "50.00" || records[1][16] != "349.00" {
		t.Fatalf("doctor export returned wrong pricing columns: %#v", records[1])
	}
	assertExportOperationLog(t, db, fixture.doctor.ID, "appointment", 1)
}

func TestExportAppointmentsAdminCanFilterStatusAndKeyword(t *testing.T) {
	handler, _, fixture := newAppointmentExportFixture(t)
	router := newAppointmentExportRouter(handler, fixture.admin)

	response := performAppointmentExportRequest(t, router, "/appointments/export?status=reported&keyword=其他套餐")

	records := decodeAppointmentExportCSV(t, response)
	if len(records) != 2 {
		t.Fatalf("expected one filtered appointment row, got %#v", records)
	}
	if records[1][0] != fixture.otherAppointment.OrderNo || records[1][5] != fixture.otherPackage.Name || records[1][12] != "reported" {
		t.Fatalf("admin export filter returned wrong row: %#v", records)
	}
}

type appointmentExportFixture struct {
	admin             models.User
	user              models.User
	otherUser         models.User
	doctor            models.User
	otherDoctor       models.User
	institution       models.CheckupInstitution
	pkg               models.CheckupPackage
	otherPackage      models.CheckupPackage
	doctorAppointment models.Appointment
	otherAppointment  models.Appointment
}

func newAppointmentExportFixture(t *testing.T) (*Handler, *gorm.DB, appointmentExportFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.CheckupInstitution{}, &models.CheckupPackage{}, &models.Appointment{}, &models.OperationLog{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	fixture := appointmentExportFixture{
		admin:       models.User{ID: 1, Name: "管理员", Email: "admin@example.com", Phone: "13800000001", Role: "admin", Status: "active", PasswordHash: "hash"},
		user:        models.User{ID: 2, Name: "用户甲", Email: "user@example.com", Phone: "13800000002", Role: "user", Status: "active", PasswordHash: "hash"},
		otherUser:   models.User{ID: 3, Name: "用户乙", Email: "other@example.com", Phone: "13800000003", Role: "user", Status: "active", PasswordHash: "hash"},
		doctor:      models.User{ID: 4, Name: "医生甲", Email: "doctor@example.com", Phone: "13800000004", Role: "doctor", Status: "active", PasswordHash: "hash"},
		otherDoctor: models.User{ID: 5, Name: "医生乙", Email: "doctor2@example.com", Phone: "13800000005", Role: "doctor", Status: "active", PasswordHash: "hash"},
		institution: models.CheckupInstitution{ID: 10, Name: "主院区", Address: "健康路 1 号", Status: "active"},
		pkg:         models.CheckupPackage{ID: 20, Name: "年度套餐", Category: "年度综合", Price: 399, Items: "血常规", Status: "active"},
		otherPackage: models.CheckupPackage{
			ID: 21, Name: "其他套餐", Category: "影像专项", Price: 299, Items: "胸片", Status: "active",
		},
		doctorAppointment: models.Appointment{ID: 30, OrderNo: "HCEXPORT001", UserID: 2, DoctorID: 4, InstitutionID: 10, PackageID: 20, AppointmentType: "个人体检", Category: "年度综合", Date: "2026-07-02", Period: "上午", StartTime: "09:00", EndTime: "09:30", Status: "booked", PaymentStatus: "paid", OriginalAmount: 399, DiscountAmount: 50, PayableAmount: 349, InvoiceTitle: "用户甲", InvoiceTaxNo: "TAX001", Note: "导出测试"},
		otherAppointment:  models.Appointment{ID: 31, OrderNo: "HCEXPORT002", UserID: 3, DoctorID: 5, InstitutionID: 10, PackageID: 21, AppointmentType: "复查体检", Category: "影像专项", Date: "2026-07-03", Period: "下午", StartTime: "14:00", EndTime: "14:30", Status: "reported", PaymentStatus: "unpaid", OriginalAmount: 299, PayableAmount: 299},
	}
	for _, row := range []any{&fixture.admin, &fixture.user, &fixture.otherUser, &fixture.doctor, &fixture.otherDoctor, &fixture.institution, &fixture.pkg, &fixture.otherPackage, &fixture.doctorAppointment, &fixture.otherAppointment} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}, db, fixture
}

func newAppointmentExportRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.GET("/appointments/export", handler.exportAppointments)
	return router
}

func performAppointmentExportRequest(t *testing.T, router *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeAppointmentExportCSV(t *testing.T, response *httptest.ResponseRecorder) [][]string {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	reader := csv.NewReader(strings.NewReader(response.Body.String()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("decode csv: %v", err)
	}
	return records
}

func assertExportOperationLog(t *testing.T, db *gorm.DB, userID uint, resource string, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&models.OperationLog{}).Where("user_id = ? AND action = ? AND resource = ?", userID, "export", resource).Count(&count).Error; err != nil {
		t.Fatalf("count operation logs: %v", err)
	}
	if count != want {
		t.Fatalf("expected operation log count %d, got %d", want, count)
	}
}

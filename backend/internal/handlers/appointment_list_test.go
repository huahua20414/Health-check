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

func TestAppointmentsSupportsWhitelistedSortAndPagination(t *testing.T) {
	handler, _, fixture := newAppointmentListFixture(t)
	router := newAppointmentListRouter(handler, fixture.admin)

	response := performAppointmentListRequest(t, router, "/appointments?page=1&pageSize=2&sort=appointment_time_asc")

	payload := decodeAppointmentListPage(t, response)
	if payload.Total != 3 || payload.Page != 1 || payload.PageSize != 2 {
		t.Fatalf("unexpected pagination payload: %#v", payload)
	}
	if len(payload.Items) != 2 {
		t.Fatalf("expected two items on first page, got %#v", payload.Items)
	}
	if payload.Items[0].OrderNo != "HCLIST002" || payload.Items[1].OrderNo != "HCLIST001" {
		t.Fatalf("appointments were not sorted by appointment time asc: %#v", payload.Items)
	}
}

func TestAppointmentsDefaultsToNewestCreatedSort(t *testing.T) {
	handler, _, fixture := newAppointmentListFixture(t)
	router := newAppointmentListRouter(handler, fixture.admin)

	response := performAppointmentListRequest(t, router, "/appointments?page=1&pageSize=10&sort=unsupported")

	payload := decodeAppointmentListPage(t, response)
	if len(payload.Items) != 3 {
		t.Fatalf("expected three items, got %#v", payload.Items)
	}
	if payload.Items[0].OrderNo != "HCLIST003" {
		t.Fatalf("unsupported sort should fall back to newest created order, got %#v", payload.Items)
	}
}

type appointmentListFixture struct {
	admin        models.User
	user         models.User
	doctor       models.User
	institution  models.CheckupInstitution
	pkg          models.CheckupPackage
	appointment1 models.Appointment
	appointment2 models.Appointment
	appointment3 models.Appointment
}

func newAppointmentListFixture(t *testing.T) (*Handler, *gorm.DB, appointmentListFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.CheckupInstitution{}, &models.CheckupPackage{}, &models.Appointment{}, &models.Report{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	fixture := appointmentListFixture{
		admin:       models.User{ID: 1, Name: "管理员", Email: "admin@example.com", Phone: "13800000001", Role: "admin", Status: "active", PasswordHash: "hash"},
		user:        models.User{ID: 2, Name: "用户甲", Email: "user@example.com", Phone: "13800000002", Role: "user", Status: "active", PasswordHash: "hash"},
		doctor:      models.User{ID: 4, Name: "医生甲", Email: "doctor@example.com", Phone: "13800000004", Role: "doctor", Status: "active", PasswordHash: "hash"},
		institution: models.CheckupInstitution{ID: 10, Name: "主院区", Address: "健康路 1 号", Status: "active"},
		pkg:         models.CheckupPackage{ID: 20, Name: "年度套餐", Category: "年度综合", Price: 399, Items: "血常规", Status: "active"},
		appointment1: models.Appointment{
			ID: 30, OrderNo: "HCLIST001", UserID: 2, DoctorID: 4, InstitutionID: 10, PackageID: 20, Date: "2026-07-02", Period: "上午", StartTime: "09:00", EndTime: "09:30", Status: "booked", PaymentStatus: "paid", PayableAmount: 399,
		},
		appointment2: models.Appointment{
			ID: 31, OrderNo: "HCLIST002", UserID: 2, DoctorID: 4, InstitutionID: 10, PackageID: 20, Date: "2026-07-01", Period: "上午", StartTime: "10:00", EndTime: "10:30", Status: "booked", PaymentStatus: "unpaid", PayableAmount: 299,
		},
		appointment3: models.Appointment{
			ID: 32, OrderNo: "HCLIST003", UserID: 2, DoctorID: 4, InstitutionID: 10, PackageID: 20, Date: "2026-07-03", Period: "下午", StartTime: "14:00", EndTime: "14:30", Status: "reported", PaymentStatus: "paid", PayableAmount: 599,
		},
	}
	for _, row := range []any{&fixture.admin, &fixture.user, &fixture.doctor, &fixture.institution, &fixture.pkg, &fixture.appointment1, &fixture.appointment2, &fixture.appointment3} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}, db, fixture
}

func newAppointmentListRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.GET("/appointments", handler.appointments)
	return router
}

func performAppointmentListRequest(t *testing.T, router *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeAppointmentListPage(t *testing.T, response *httptest.ResponseRecorder) struct {
	Items    []models.Appointment `json:"items"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"pageSize"`
} {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Items    []models.Appointment `json:"items"`
		Total    int64                `json:"total"`
		Page     int                  `json:"page"`
		PageSize int                  `json:"pageSize"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode appointment page: %v", err)
	}
	return payload
}

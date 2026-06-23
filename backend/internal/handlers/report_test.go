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

func TestReportsAreScopedByUser(t *testing.T) {
	handler, _, fixture := newReportFixture(t)
	router := newReportRouter(handler, fixture.user)

	response := performReportRequest(t, router, "/reports")

	reports := decodeReportList(t, response, 1)
	if reports[0].UserID != fixture.user.ID || reports[0].ID != fixture.userReport.ID {
		t.Fatalf("user should only see own report, got %#v", reports)
	}
}

func TestReportsAreScopedByDoctor(t *testing.T) {
	handler, _, fixture := newReportFixture(t)
	router := newReportRouter(handler, fixture.doctor)

	response := performReportRequest(t, router, "/reports")

	reports := decodeReportList(t, response, 1)
	if reports[0].DoctorID != fixture.doctor.ID || reports[0].ID != fixture.userReport.ID {
		t.Fatalf("doctor should only see assigned reports, got %#v", reports)
	}
}

func TestAdminCanListAllReports(t *testing.T) {
	handler, _, fixture := newReportFixture(t)
	router := newReportRouter(handler, fixture.admin)

	response := performReportRequest(t, router, "/reports")

	reports := decodeReportList(t, response, 2)
	seen := map[uint]bool{}
	for _, report := range reports {
		seen[report.ID] = true
	}
	if !seen[fixture.userReport.ID] || !seen[fixture.otherReport.ID] {
		t.Fatalf("admin should see all reports, got %#v", reports)
	}
}

func TestAdminCanFilterReportsByUserID(t *testing.T) {
	handler, _, fixture := newReportFixture(t)
	router := newReportRouter(handler, fixture.admin)

	response := performReportRequest(t, router, "/reports?userId=101")

	reports := decodeReportList(t, response, 1)
	if reports[0].UserID != fixture.otherUser.ID || reports[0].ID != fixture.otherReport.ID {
		t.Fatalf("admin user filter returned wrong reports: %#v", reports)
	}
}

func TestUserIgnoresUserIDFilterForReports(t *testing.T) {
	handler, _, fixture := newReportFixture(t)
	router := newReportRouter(handler, fixture.user)

	response := performReportRequest(t, router, "/reports?userId=101")

	reports := decodeReportList(t, response, 1)
	if reports[0].UserID != fixture.user.ID || reports[0].ID != fixture.userReport.ID {
		t.Fatalf("userId filter should not let users read others reports: %#v", reports)
	}
}

func TestReportsPaginationKeepsRoleScope(t *testing.T) {
	handler, _, fixture := newReportFixture(t)
	router := newReportRouter(handler, fixture.doctor)

	response := performReportRequest(t, router, "/reports?page=1&pageSize=10")

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Items []models.Report `json:"items"`
		Total int64           `json:"total"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode paginated reports: %v", err)
	}
	if payload.Total != 1 || len(payload.Items) != 1 || payload.Items[0].DoctorID != fixture.doctor.ID {
		t.Fatalf("doctor pagination should stay scoped, got %#v", payload)
	}
}

type reportFixture struct {
	user             models.User
	otherUser        models.User
	doctor           models.User
	otherDoctor      models.User
	admin            models.User
	institution      models.CheckupInstitution
	pkg              models.CheckupPackage
	userAppointment  models.Appointment
	otherAppointment models.Appointment
	userReport       models.Report
	otherReport      models.Report
}

func newReportFixture(t *testing.T) (*Handler, *gorm.DB, reportFixture) {
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
		&models.Appointment{},
		&models.Report{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	fixture := reportFixture{
		user:             models.User{ID: 100, Name: "用户", Phone: "13800000100", Email: "user@example.com", Role: "user", Status: "active", PasswordHash: "hash"},
		otherUser:        models.User{ID: 101, Name: "其他用户", Phone: "13800000101", Email: "other@example.com", Role: "user", Status: "active", PasswordHash: "hash"},
		doctor:           models.User{ID: 200, Name: "医生", Phone: "13800000200", Email: "doctor@example.com", Role: "doctor", Status: "active", PasswordHash: "hash"},
		otherDoctor:      models.User{ID: 201, Name: "其他医生", Phone: "13800000201", Email: "doctor2@example.com", Role: "doctor", Status: "active", PasswordHash: "hash"},
		admin:            models.User{ID: 300, Name: "管理员", Phone: "13800000300", Email: "admin@example.com", Role: "admin", Status: "active", PasswordHash: "hash"},
		institution:      models.CheckupInstitution{ID: 10, Name: "主院区", Address: "健康路 1 号", Status: "active"},
		pkg:              models.CheckupPackage{ID: 20, Name: "年度体检", Category: "年度综合", Price: 399, Items: "血常规", Status: "active"},
		userAppointment:  models.Appointment{ID: 30, OrderNo: "HC202607010001", UserID: 100, DoctorID: 200, InstitutionID: 10, PackageID: 20, AppointmentType: "个人体检", Category: "年度综合", Date: "2026-07-01", Period: "上午", Status: "reported"},
		otherAppointment: models.Appointment{ID: 31, OrderNo: "HC202607010002", UserID: 101, DoctorID: 201, InstitutionID: 10, PackageID: 20, AppointmentType: "个人体检", Category: "年度综合", Date: "2026-07-02", Period: "上午", Status: "reported"},
		userReport:       models.Report{ID: 40, ReportNo: "R202607010001", AppointmentID: 30, UserID: 100, DoctorID: 200, Summary: "用户报告", Conclusion: "正常", Recommendation: "保持运动"},
		otherReport:      models.Report{ID: 41, ReportNo: "R202607010002", AppointmentID: 31, UserID: 101, DoctorID: 201, Summary: "其他报告", Conclusion: "正常", Recommendation: "保持运动"},
	}
	for _, row := range []any{
		&fixture.user,
		&fixture.otherUser,
		&fixture.doctor,
		&fixture.otherDoctor,
		&fixture.admin,
		&fixture.institution,
		&fixture.pkg,
		&fixture.userAppointment,
		&fixture.otherAppointment,
		&fixture.userReport,
		&fixture.otherReport,
	} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}
	handler := &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}
	return handler, db, fixture
}

func newReportRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.GET("/reports", handler.reports)
	return router
}

func performReportRequest(t *testing.T, router *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeReportList(t *testing.T, response *httptest.ResponseRecorder, want int) []models.Report {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var reports []models.Report
	if err := json.Unmarshal(response.Body.Bytes(), &reports); err != nil {
		t.Fatalf("decode reports: %v", err)
	}
	if len(reports) != want {
		t.Fatalf("expected %d reports, got %d: %#v", want, len(reports), reports)
	}
	return reports
}

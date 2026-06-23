package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"health-checkup/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAdminDashboardFiltersTrendsByRequestedDays(t *testing.T) {
	handler, _, fixture := newAdminDashboardFixture(t)
	router := newAdminDashboardRouter(handler)

	response := performAdminDashboardRequest(t, router, "/admin/dashboard?days=7")

	payload := decodeAdminDashboardPayload(t, response)
	if payload.Range.Days != 7 {
		t.Fatalf("expected range days 7, got %#v", payload.Range)
	}
	assertDashboardLabels(t, payload.AppointmentTrend, []string{fixture.recentAppointmentDay})
	assertDashboardLabels(t, payload.PackageSales, []string{"近期套餐"})
	assertDashboardLabels(t, payload.UserGrowth, []string{fixture.recentUserDay})
	assertDashboardLabels(t, payload.PaymentStatus, []string{"paid"})
	if payload.PaymentStatus[0].Total != 399 {
		t.Fatalf("expected paid revenue 399, got %#v", payload.PaymentStatus)
	}
	if int64FromSummary(t, payload.Summary, "waitlist") != 1 {
		t.Fatalf("expected one in-range waitlist entry, got %#v", payload.Summary)
	}
	if int64FromSummary(t, payload.Summary, "slotCapacity") != 3 || int64FromSummary(t, payload.Summary, "slotBooked") != 2 {
		t.Fatalf("expected capacity 3 and booked 2, got %#v", payload.Summary)
	}
}

func TestAdminDashboardClampsDaysRange(t *testing.T) {
	handler, _, _ := newAdminDashboardFixture(t)
	router := newAdminDashboardRouter(handler)

	low := decodeAdminDashboardPayload(t, performAdminDashboardRequest(t, router, "/admin/dashboard?days=1"))
	high := decodeAdminDashboardPayload(t, performAdminDashboardRequest(t, router, "/admin/dashboard?days=365"))

	if low.Range.Days != 7 {
		t.Fatalf("expected low range to clamp to 7, got %#v", low.Range)
	}
	if high.Range.Days != 90 {
		t.Fatalf("expected high range to clamp to 90, got %#v", high.Range)
	}
}

type dashboardPayload struct {
	Summary          map[string]any `json:"summary"`
	Range            dashboardRange `json:"range"`
	AppointmentTrend []dashboardRow `json:"appointmentTrend"`
	PackageSales     []dashboardRow `json:"packageSales"`
	PaymentStatus    []dashboardRow `json:"paymentStatus"`
	UserGrowth       []dashboardRow `json:"userGrowth"`
}

type dashboardRange struct {
	Days                 int    `json:"days"`
	AppointmentStartDate string `json:"appointmentStartDate"`
	AppointmentEndDate   string `json:"appointmentEndDate"`
	GrowthStartDate      string `json:"growthStartDate"`
	GrowthEndDate        string `json:"growthEndDate"`
}

type dashboardRow struct {
	Label string  `json:"label"`
	Count int64   `json:"count"`
	Total float64 `json:"total"`
}

type adminDashboardFixture struct {
	recentAppointmentDay string
	recentUserDay        string
}

func newAdminDashboardFixture(t *testing.T) (*Handler, *gorm.DB, adminDashboardFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.CheckupPackage{}, &models.CheckupInstitution{}, &models.ScheduleSlot{}, &models.Appointment{}, &models.WaitlistEntry{}, &models.Report{}, &models.ServiceReview{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	now := time.Now()
	recentAppointmentDay := now.AddDate(0, 0, 3).Format("2006-01-02")
	outsideAppointmentDay := now.AddDate(0, 0, 30).Format("2006-01-02")
	recentUser := models.User{ID: 1, Name: "近期用户", Email: "recent@example.com", Phone: "13800000001", Role: "user", Status: "active", PasswordHash: "hash", CreatedAt: now}
	oldUser := models.User{ID: 2, Name: "历史用户", Email: "old@example.com", Phone: "13800000002", Role: "user", Status: "active", PasswordHash: "hash", CreatedAt: now.AddDate(0, 0, -30)}
	doctor := models.User{ID: 3, Name: "医生", Email: "doctor@example.com", Phone: "13800000003", Role: "doctor", Status: "active", PasswordHash: "hash", CreatedAt: now}
	institution := models.CheckupInstitution{ID: 10, Name: "主院区", Address: "健康路 1 号", Status: "active"}
	recentPackage := models.CheckupPackage{ID: 20, Name: "近期套餐", Category: "年度综合", Price: 399, Items: "血常规", Status: "active"}
	oldPackage := models.CheckupPackage{ID: 21, Name: "历史套餐", Category: "年度综合", Price: 199, Items: "血常规", Status: "active"}
	slot := models.ScheduleSlot{ID: 30, DoctorID: doctor.ID, InstitutionID: institution.ID, Date: recentAppointmentDay, Period: "上午", Category: "年度综合", StartTime: "09:00", EndTime: "09:30", Capacity: 3, BookedCount: 2, Status: "available"}
	recentAppointment := models.Appointment{ID: 40, OrderNo: "HCRECENT", UserID: recentUser.ID, DoctorID: doctor.ID, InstitutionID: institution.ID, SlotID: slot.ID, PackageID: recentPackage.ID, AppointmentType: "个人体检", Category: "年度综合", Date: recentAppointmentDay, Period: "上午", StartTime: "09:00", EndTime: "09:30", Status: "booked", PaymentStatus: "paid", OriginalAmount: 399, PayableAmount: 399}
	oldAppointment := models.Appointment{ID: 41, OrderNo: "HCOLD", UserID: oldUser.ID, DoctorID: doctor.ID, InstitutionID: institution.ID, SlotID: slot.ID, PackageID: oldPackage.ID, AppointmentType: "个人体检", Category: "年度综合", Date: outsideAppointmentDay, Period: "上午", StartTime: "10:00", EndTime: "10:30", Status: "booked", PaymentStatus: "paid", OriginalAmount: 199, PayableAmount: 199}
	recentWaitlist := models.WaitlistEntry{ID: 50, UserID: recentUser.ID, PackageID: recentPackage.ID, InstitutionID: institution.ID, AppointmentType: "个人体检", Category: "年度综合", Date: recentAppointmentDay, Period: "上午", StartTime: "10:00", EndTime: "10:30", Status: "waiting"}
	oldWaitlist := models.WaitlistEntry{ID: 51, UserID: oldUser.ID, PackageID: oldPackage.ID, InstitutionID: institution.ID, AppointmentType: "个人体检", Category: "年度综合", Date: outsideAppointmentDay, Period: "上午", StartTime: "10:00", EndTime: "10:30", Status: "waiting"}
	for _, row := range []any{&recentUser, &oldUser, &doctor, &institution, &recentPackage, &oldPackage, &slot, &recentAppointment, &oldAppointment, &recentWaitlist, &oldWaitlist} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}, db, adminDashboardFixture{
		recentAppointmentDay: recentAppointmentDay,
		recentUserDay:        now.UTC().Format("2006-01-02"),
	}
}

func newAdminDashboardRouter(handler *Handler) *gin.Engine {
	router := gin.New()
	router.GET("/admin/dashboard", handler.adminDashboard)
	return router
}

func performAdminDashboardRequest(t *testing.T, router *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeAdminDashboardPayload(t *testing.T, response *httptest.ResponseRecorder) dashboardPayload {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload dashboardPayload
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode dashboard payload: %v", err)
	}
	return payload
}

func assertDashboardLabels(t *testing.T, rows []dashboardRow, want []string) {
	t.Helper()
	if len(rows) != len(want) {
		t.Fatalf("expected labels %v, got %#v", want, rows)
	}
	for i, row := range rows {
		if row.Label != want[i] {
			t.Fatalf("expected label %q at %d, got %#v", want[i], i, rows)
		}
	}
}

func int64FromSummary(t *testing.T, summary map[string]any, key string) int64 {
	t.Helper()
	value, ok := summary[key].(float64)
	if !ok {
		t.Fatalf("summary key %s missing or not numeric: %#v", key, summary)
	}
	return int64(value)
}

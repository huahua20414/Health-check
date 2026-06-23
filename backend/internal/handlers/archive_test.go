package handlers

import (
	"encoding/csv"
	"encoding/json"
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

func TestArchiveByIDMarksResourceDeletedAndRejectsRepeat(t *testing.T) {
	handler, db := newArchiveTestHandler(t)
	coupon := models.Coupon{Name: "新人券", Code: "NEW100", Type: "amount", Value: 100, Status: "active"}
	if err := db.Create(&coupon).Error; err != nil {
		t.Fatalf("create coupon: %v", err)
	}

	if err := handler.archiveByID(&models.Coupon{}, int(coupon.ID), "coupon"); err != nil {
		t.Fatalf("archive coupon: %v", err)
	}

	var archived models.Coupon
	if err := db.First(&archived, coupon.ID).Error; err != nil {
		t.Fatalf("load archived coupon: %v", err)
	}
	if archived.Status != "deleted" {
		t.Fatalf("expected deleted status, got %q", archived.Status)
	}
	if err := handler.archiveByID(&models.Coupon{}, int(coupon.ID), "coupon"); err == nil {
		t.Fatal("expected repeat archive to fail")
	}
}

func TestAdminResourceListsHideArchivedRowsByDefault(t *testing.T) {
	handler, db := newArchiveTestHandler(t)
	seedArchiveRows(t, db)

	tests := []struct {
		name    string
		path    string
		handler gin.HandlerFunc
		want    string
	}{
		{name: "coupons", path: "/api/coupons", handler: handler.coupons, want: "启用优惠券"},
		{name: "announcements", path: "/api/announcements", handler: handler.announcements, want: "启用公告"},
		{name: "packages", path: "/api/packages", handler: handler.packages, want: "启用套餐"},
		{name: "checkup items", path: "/api/checkup-items", handler: handler.checkupItems, want: "启用项目"},
		{name: "schedule slots", path: "/api/schedule/slots", handler: handler.scheduleSlots, want: "2026-07-01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := performArchiveRequest(t, tt.handler, http.MethodGet, tt.path, true)
			if len(body) != 1 {
				t.Fatalf("expected one visible row, got %d: %#v", len(body), body)
			}
			encoded := encodeJSON(t, body[0])
			if !jsonContains(encoded, tt.want) {
				t.Fatalf("expected response to contain %q, got %s", tt.want, encoded)
			}
			if jsonContains(encoded, "归档") || jsonContains(encoded, "2026-07-02") {
				t.Fatalf("default response leaked archived row: %s", encoded)
			}
		})
	}
}

func TestAdminResourceListsCanQueryArchivedRowsExplicitly(t *testing.T) {
	handler, db := newArchiveTestHandler(t)
	seedArchiveRows(t, db)

	tests := []struct {
		name    string
		path    string
		handler gin.HandlerFunc
		want    string
	}{
		{name: "coupons", path: "/api/coupons?status=deleted", handler: handler.coupons, want: "归档优惠券"},
		{name: "announcements", path: "/api/announcements?status=deleted", handler: handler.announcements, want: "归档公告"},
		{name: "packages", path: "/api/packages?status=deleted", handler: handler.packages, want: "归档套餐"},
		{name: "checkup items", path: "/api/checkup-items?status=deleted", handler: handler.checkupItems, want: "归档项目"},
		{name: "schedule slots", path: "/api/schedule/slots?status=deleted", handler: handler.scheduleSlots, want: "2026-07-02"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := performArchiveRequest(t, tt.handler, http.MethodGet, tt.path, true)
			if len(body) != 1 {
				t.Fatalf("expected one archived row, got %d: %#v", len(body), body)
			}
			encoded := encodeJSON(t, body[0])
			if !jsonContains(encoded, tt.want) {
				t.Fatalf("expected response to contain %q, got %s", tt.want, encoded)
			}
		})
	}
}

func TestAnnouncementsSupportKeywordAndPagination(t *testing.T) {
	handler, db := newArchiveTestHandler(t)
	seedArchiveRows(t, db)
	if err := db.Create(&models.SystemAnnouncement{Title: "医生排班调整", Content: "本周医生号源有调整", Audience: "doctor", Status: "published"}).Error; err != nil {
		t.Fatalf("create announcement: %v", err)
	}

	page := performArchivePageRequest(t, handler.announcements, http.MethodGet, "/api/announcements?keyword=医生&page=1&pageSize=1", true)

	if page.Total != 1 || page.Page != 1 || page.PageSize != 1 {
		t.Fatalf("unexpected pagination metadata: %#v", page)
	}
	if len(page.Items) != 1 || !jsonContains(encodeJSON(t, page.Items[0]), "医生排班调整") {
		t.Fatalf("unexpected announcement page: %#v", page.Items)
	}
}

func TestExportAnnouncementsUsesFiltersAndAudits(t *testing.T) {
	handler, db := newArchiveTestHandler(t)
	seedArchiveRows(t, db)
	admin := models.User{ID: 99, Name: "管理员", Role: "admin", Status: "active"}
	router := newArchiveUserRouter(handler.exportAnnouncements, http.MethodGet, "/api/announcements/export", admin)

	response := performArchiveRouterRequest(t, router, http.MethodGet, "/api/announcements/export?status=published&keyword=启用")

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	rows, err := csv.NewReader(strings.NewReader(response.Body.String())).ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected header plus one announcement, got %d rows: %#v", len(rows), rows)
	}
	if rows[1][1] != "启用公告" || rows[1][2] != "all" || rows[1][3] != "published" {
		t.Fatalf("unexpected announcement csv row: %#v", rows[1])
	}
	assertArchiveOperationLogCount(t, db, admin.ID, "export", "announcement", 1)
}

func newArchiveTestHandler(t *testing.T) (*Handler, *gorm.DB) {
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
		&models.CheckupItem{},
		&models.Coupon{},
		&models.SystemAnnouncement{},
		&models.ScheduleSlot{},
		&models.OperationLog{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}, db
}

func seedArchiveRows(t *testing.T, db *gorm.DB) {
	t.Helper()
	doctor := models.User{Name: "医生", Role: "doctor", Status: "active"}
	institution := models.CheckupInstitution{Name: "主院区", Address: "健康路 1 号", Status: "active"}
	rows := []any{
		&doctor,
		&institution,
		&models.Coupon{Name: "启用优惠券", Code: "ACTIVE100", Type: "amount", Value: 100, Status: "active"},
		&models.Coupon{Name: "归档优惠券", Code: "DELETED100", Type: "amount", Value: 100, Status: "deleted"},
		&models.SystemAnnouncement{Title: "启用公告", Content: "公告内容", Audience: "all", Status: "published"},
		&models.SystemAnnouncement{Title: "归档公告", Content: "公告内容", Audience: "all", Status: "deleted"},
		&models.CheckupPackage{Name: "启用套餐", Category: "年度综合", Price: 399, Items: "血常规", Status: "active"},
		&models.CheckupPackage{Name: "归档套餐", Category: "年度综合", Price: 399, Items: "血常规", Status: "deleted"},
		&models.CheckupItem{Name: "启用项目", Category: "检验", Department: "检验科", Price: 30, DurationMin: 10, Status: "active"},
		&models.CheckupItem{Name: "归档项目", Category: "检验", Department: "检验科", Price: 30, DurationMin: 10, Status: "deleted"},
	}
	for _, row := range rows {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create seed row %#v: %v", row, err)
		}
	}
	slots := []models.ScheduleSlot{
		{DoctorID: doctor.ID, InstitutionID: institution.ID, Date: "2026-07-01", Period: "上午", Category: "年度综合", StartTime: "09:00", EndTime: "09:30", Capacity: 1, Status: "available"},
		{DoctorID: doctor.ID, InstitutionID: institution.ID, Date: "2026-07-02", Period: "上午", Category: "年度综合", StartTime: "09:00", EndTime: "09:30", Capacity: 1, Status: "deleted"},
	}
	if err := db.Create(&slots).Error; err != nil {
		t.Fatalf("create schedule slots: %v", err)
	}
}

func performArchiveRequest(t *testing.T, handler gin.HandlerFunc, method, path string, withAuthHeader bool) []map[string]any {
	t.Helper()
	router := gin.New()
	router.Handle(method, pathWithoutQuery(path), handler)
	req := httptest.NewRequest(method, path, nil)
	if withAuthHeader {
		req.Header.Set("Authorization", "Bearer test")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response %s: %v", rec.Body.String(), err)
	}
	return body
}

func newArchiveUserRouter(handler gin.HandlerFunc, method, path string, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.Handle(method, path, handler)
	return router
}

func performArchiveRouterRequest(t *testing.T, router *gin.Engine, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func assertArchiveOperationLogCount(t *testing.T, db *gorm.DB, userID uint, action, resource string, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&models.OperationLog{}).Where("user_id = ? AND action = ? AND resource = ?", userID, action, resource).Count(&count).Error; err != nil {
		t.Fatalf("count operation logs: %v", err)
	}
	if count != want {
		t.Fatalf("expected %d operation logs, got %d", want, count)
	}
}

type archivePageResponse struct {
	Items    []map[string]any `json:"items"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"pageSize"`
}

func performArchivePageRequest(t *testing.T, handler gin.HandlerFunc, method, path string, withAuthHeader bool) archivePageResponse {
	t.Helper()
	router := gin.New()
	router.Handle(method, pathWithoutQuery(path), handler)
	req := httptest.NewRequest(method, path, nil)
	if withAuthHeader {
		req.Header.Set("Authorization", "Bearer test")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body archivePageResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response %s: %v", rec.Body.String(), err)
	}
	return body
}

func pathWithoutQuery(path string) string {
	if i := len(path); i > 0 {
		for idx, ch := range path {
			if ch == '?' {
				return path[:idx]
			}
		}
	}
	return path
}

func encodeJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal value: %v", err)
	}
	return string(data)
}

func jsonContains(body, want string) bool {
	return len(want) == 0 || strings.Contains(body, want)
}

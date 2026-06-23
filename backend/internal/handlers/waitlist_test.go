package handlers

import (
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

func TestCancelWaitlistMarksOwnWaitingEntryCanceled(t *testing.T) {
	handler, db, fixture := newWaitlistFixture(t)
	router := newWaitlistRouter(handler, fixture.user)

	response := performCancelWaitlistRequest(t, router, fixture.waitingEntry.ID)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	assertWaitlistStatus(t, db, fixture.waitingEntry.ID, "canceled")
	assertWaitlistNotification(t, db, fixture.user.ID)
	assertWaitlistOperationLog(t, db, fixture.user.ID, fixture.waitingEntry.ID)
}

func TestCancelWaitlistRejectsOtherUsersEntry(t *testing.T) {
	handler, db, fixture := newWaitlistFixture(t)
	router := newWaitlistRouter(handler, fixture.user)

	response := performCancelWaitlistRequest(t, router, fixture.otherEntry.ID)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "waitlist entry not found")
	assertWaitlistStatus(t, db, fixture.otherEntry.ID, "waiting")
}

func TestCancelWaitlistRejectsPromotedEntry(t *testing.T) {
	handler, db, fixture := newWaitlistFixture(t)
	router := newWaitlistRouter(handler, fixture.user)

	response := performCancelWaitlistRequest(t, router, fixture.promotedEntry.ID)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "waitlist entry not found")
	assertWaitlistStatus(t, db, fixture.promotedEntry.ID, "promoted")
}

type waitlistFixture struct {
	user          models.User
	otherUser     models.User
	waitingEntry  models.WaitlistEntry
	otherEntry    models.WaitlistEntry
	promotedEntry models.WaitlistEntry
}

func newWaitlistFixture(t *testing.T) (*Handler, *gorm.DB, waitlistFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.WaitlistEntry{}, &models.Notification{}, &models.OperationLog{}, &models.SystemSetting{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	fixture := waitlistFixture{
		user:          models.User{ID: 100, Name: "用户", Phone: "13800000100", Email: "user@example.com", Role: "user", Status: "active", PasswordHash: "hash"},
		otherUser:     models.User{ID: 101, Name: "其他用户", Phone: "13800000101", Email: "other@example.com", Role: "user", Status: "active", PasswordHash: "hash"},
		waitingEntry:  models.WaitlistEntry{ID: 10, UserID: 100, PackageID: 20, InstitutionID: 30, AppointmentType: "个人体检", Category: "年度综合", Date: "2026-07-01", Period: "上午", Status: "waiting"},
		otherEntry:    models.WaitlistEntry{ID: 11, UserID: 101, PackageID: 20, InstitutionID: 30, AppointmentType: "个人体检", Category: "年度综合", Date: "2026-07-01", Period: "上午", Status: "waiting"},
		promotedEntry: models.WaitlistEntry{ID: 12, UserID: 100, PackageID: 20, InstitutionID: 30, AppointmentType: "个人体检", Category: "年度综合", Date: "2026-07-02", Period: "上午", Status: "promoted"},
	}
	inAppSetting := models.SystemSetting{Key: "notification.in_app_enabled", Value: "true", ValueType: "boolean", Group: "notification", Label: "站内信通知", Status: "active"}
	for _, row := range []any{&fixture.user, &fixture.otherUser, &fixture.waitingEntry, &fixture.otherEntry, &fixture.promotedEntry, &inAppSetting} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}, db, fixture
}

func newWaitlistRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.PATCH("/waitlist/:id/cancel", handler.cancelWaitlist)
	return router
}

func performCancelWaitlistRequest(t *testing.T, router *gin.Engine, waitlistID uint) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, "/waitlist/"+strconv.Itoa(int(waitlistID))+"/cancel", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func assertWaitlistStatus(t *testing.T, db *gorm.DB, waitlistID uint, want string) {
	t.Helper()
	var entry models.WaitlistEntry
	if err := db.First(&entry, waitlistID).Error; err != nil {
		t.Fatalf("load waitlist entry: %v", err)
	}
	if entry.Status != want {
		t.Fatalf("expected waitlist %d status %q, got %q", waitlistID, want, entry.Status)
	}
}

func assertWaitlistNotification(t *testing.T, db *gorm.DB, userID uint) {
	t.Helper()
	var count int64
	if err := db.Model(&models.Notification{}).Where("user_id = ? AND type = ?", userID, "waitlist_canceled").Count(&count).Error; err != nil {
		t.Fatalf("count waitlist cancel notifications: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one waitlist cancel notification, got %d", count)
	}
}

func assertWaitlistOperationLog(t *testing.T, db *gorm.DB, userID, waitlistID uint) {
	t.Helper()
	var count int64
	if err := db.Model(&models.OperationLog{}).Where("user_id = ? AND action = ? AND resource = ? AND resource_id = ?", userID, "cancel", "waitlist", strconv.Itoa(int(waitlistID))).Count(&count).Error; err != nil {
		t.Fatalf("count waitlist cancel operation log: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one waitlist cancel operation log, got %d", count)
	}
}

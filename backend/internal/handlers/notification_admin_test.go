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

func TestAdminCreateNotificationTargetsActiveRoleRecipients(t *testing.T) {
	handler, db, fixture := newNotificationAdminFixture(t)
	router := newNotificationAdminRouter(handler, fixture.admin)

	response := performNotificationAdminJSON(t, router, http.MethodPost, "/admin/notifications", notificationRequest{
		Role: "user", Channel: "in_app", Type: "campaign", Title: "活动提醒", Content: "本周体检活动已开始",
	})

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	assertNotificationTitleCount(t, db, fixture.user.ID, "活动提醒", 1)
	assertNotificationTitleCount(t, db, fixture.inactiveUser.ID, "活动提醒", 0)
	assertOperationCount(t, db, "create", "notification", 1)
}

func TestAdminCreateNotificationCanTargetOneUser(t *testing.T) {
	handler, db, fixture := newNotificationAdminFixture(t)
	router := newNotificationAdminRouter(handler, fixture.admin)

	response := performNotificationAdminJSON(t, router, http.MethodPost, "/admin/notifications", notificationRequest{
		UserID: fixture.doctor.ID, Channel: "sms_mock", Title: "排班提醒", Content: "请确认明日排班",
	})

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	assertNotificationTitleCount(t, db, fixture.doctor.ID, "排班提醒", 1)
	assertNotificationTitleCount(t, db, fixture.user.ID, "排班提醒", 0)
}

func TestUserNotificationsHideArchivedRows(t *testing.T) {
	handler, _, fixture := newNotificationAdminFixture(t)
	router := newNotificationAdminRouter(handler, fixture.user)

	response := performNotificationAdminRequest(t, router, http.MethodGet, "/notifications")

	notifications := decodeNotificationList(t, response)
	if len(notifications) != 1 || notifications[0].ID != fixture.userUnread.ID {
		t.Fatalf("user should only see non-archived own notifications, got %#v", notifications)
	}
}

func TestAdminNotificationsFilterByKeywordAndChannel(t *testing.T) {
	handler, _, fixture := newNotificationAdminFixture(t)
	router := newNotificationAdminRouter(handler, fixture.admin)

	response := performNotificationAdminRequest(t, router, http.MethodGet, "/admin/notifications?page=1&pageSize=10&channel=in_app&keyword="+fixture.user.Name)

	payload := decodeNotificationPage(t, response)
	if payload.Total != 1 || len(payload.Items) != 1 || payload.Items[0].ID != fixture.userUnread.ID {
		t.Fatalf("admin notification filter returned wrong page: %#v", payload)
	}
}

func TestAdminArchiveNotificationHidesItFromUserAndRecordsOperation(t *testing.T) {
	handler, db, fixture := newNotificationAdminFixture(t)
	router := newNotificationAdminRouter(handler, fixture.admin)

	response := performNotificationAdminRequest(t, router, http.MethodDelete, "/admin/notifications/"+strconv.Itoa(int(fixture.userUnread.ID)))

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	assertNotificationStatus(t, db, fixture.userUnread.ID, "archived")
	assertOperationCount(t, db, "archive", "notification", 1)
}

type notificationAdminFixture struct {
	admin        models.User
	user         models.User
	doctor       models.User
	inactiveUser models.User
	userUnread   models.Notification
	userArchived models.Notification
	doctorNotice models.Notification
}

type notificationPage struct {
	Items []models.Notification `json:"items"`
	Total int64                 `json:"total"`
}

func newNotificationAdminFixture(t *testing.T) (*Handler, *gorm.DB, notificationAdminFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Notification{}, &models.OperationLog{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	fixture := notificationAdminFixture{
		admin:        models.User{ID: 1, Name: "管理员", Email: "admin@example.com", Phone: "13800000001", Role: "admin", Status: "active", PasswordHash: "hash"},
		user:         models.User{ID: 2, Name: "用户甲", Email: "user@example.com", Phone: "13800000002", Role: "user", Status: "active", PasswordHash: "hash"},
		doctor:       models.User{ID: 3, Name: "医生甲", Email: "doctor@example.com", Phone: "13800000003", Role: "doctor", Status: "active", PasswordHash: "hash"},
		inactiveUser: models.User{ID: 4, Name: "停用用户", Email: "disabled@example.com", Phone: "13800000004", Role: "user", Status: "disabled", PasswordHash: "hash"},
		userUnread:   models.Notification{ID: 10, UserID: 2, Channel: "in_app", Type: "admin_notice", Title: "用户提醒", Content: "请查看", Status: "unread"},
		userArchived: models.Notification{ID: 11, UserID: 2, Channel: "in_app", Type: "admin_notice", Title: "归档提醒", Content: "旧通知", Status: "archived"},
		doctorNotice: models.Notification{ID: 12, UserID: 3, Channel: "sms_mock", Type: "schedule", Title: "医生提醒", Content: "排班", Status: "unread"},
	}
	for _, row := range []any{&fixture.admin, &fixture.user, &fixture.doctor, &fixture.inactiveUser, &fixture.userUnread, &fixture.userArchived, &fixture.doctorNotice} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}, db, fixture
}

func newNotificationAdminRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.GET("/notifications", handler.notifications)
	router.GET("/admin/notifications", handler.adminNotifications)
	router.POST("/admin/notifications", handler.createAdminNotification)
	router.DELETE("/admin/notifications/:id", handler.archiveAdminNotification)
	return router
}

func performNotificationAdminRequest(t *testing.T, router *gin.Engine, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func performNotificationAdminJSON(t *testing.T, router *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeNotificationList(t *testing.T, response *httptest.ResponseRecorder) []models.Notification {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var notifications []models.Notification
	if err := json.Unmarshal(response.Body.Bytes(), &notifications); err != nil {
		t.Fatalf("decode notifications: %v", err)
	}
	return notifications
}

func decodeNotificationPage(t *testing.T, response *httptest.ResponseRecorder) notificationPage {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload notificationPage
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode notification page: %v", err)
	}
	return payload
}

func assertNotificationTitleCount(t *testing.T, db *gorm.DB, userID uint, title string, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&models.Notification{}).Where("user_id = ? AND title = ?", userID, title).Count(&count).Error; err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	if count != want {
		t.Fatalf("expected notification count %d for user=%d title=%s, got %d", want, userID, title, count)
	}
}

func assertNotificationStatus(t *testing.T, db *gorm.DB, id uint, want string) {
	t.Helper()
	var notification models.Notification
	if err := db.First(&notification, id).Error; err != nil {
		t.Fatalf("load notification: %v", err)
	}
	if notification.Status != want {
		t.Fatalf("expected notification %d status %s, got %s", id, want, notification.Status)
	}
}

func assertOperationCount(t *testing.T, db *gorm.DB, action, resource string, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&models.OperationLog{}).Where("action = ? AND resource = ?", action, resource).Count(&count).Error; err != nil {
		t.Fatalf("count operation logs: %v", err)
	}
	if count != want {
		t.Fatalf("expected operation count %d for %s/%s, got %d", want, action, resource, count)
	}
}

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

func TestUpdateSystemSettingAcceptsStructuredFAQ(t *testing.T) {
	handler, db, setting := newSystemSettingFixture(t)
	router := newSystemSettingRouter(handler, models.User{ID: 1, Name: "管理员", Role: "admin", Status: "active"})

	response := performSystemSettingPatch(t, router, setting.ID, systemSettingRequest{
		Value:       `[{"question":"如何改期？","answer":"在我的预约中选择可改期预约提交新时间。"}]`,
		ValueType:   "json",
		Label:       "常见问题 FAQ",
		Description: "结构化 FAQ",
		Status:      "active",
	})

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var updated models.SystemSetting
	if err := db.First(&updated, setting.ID).Error; err != nil {
		t.Fatalf("load setting: %v", err)
	}
	if updated.Value != `[{"question":"如何改期？","answer":"在我的预约中选择可改期预约提交新时间。"}]` {
		t.Fatalf("FAQ setting was not updated: %#v", updated)
	}
	assertOperationCount(t, db, "update", "system_setting", 1)
}

func TestUpdateSystemSettingRejectsInvalidFAQ(t *testing.T) {
	handler, db, setting := newSystemSettingFixture(t)
	router := newSystemSettingRouter(handler, models.User{ID: 1, Name: "管理员", Role: "admin", Status: "active"})

	response := performSystemSettingPatch(t, router, setting.ID, systemSettingRequest{
		Value:       `[{"question":"","answer":"缺少问题"}]`,
		ValueType:   "json",
		Label:       "常见问题 FAQ",
		Description: "结构化 FAQ",
		Status:      "active",
	})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	var unchanged models.SystemSetting
	if err := db.First(&unchanged, setting.ID).Error; err != nil {
		t.Fatalf("load setting: %v", err)
	}
	if unchanged.Value != setting.Value {
		t.Fatalf("invalid FAQ should not change persisted value: %#v", unchanged)
	}
}

func newSystemSettingFixture(t *testing.T) (*Handler, *gorm.DB, models.SystemSetting) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.SystemSetting{}, &models.OperationLog{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	setting := models.SystemSetting{
		ID:          10,
		Key:         "service.faq",
		Value:       `[{"question":"体检前需要注意什么？","answer":"请携带证件并清淡饮食。"}]`,
		ValueType:   "json",
		Group:       "service",
		Label:       "常见问题 FAQ",
		Description: "用户端 FAQ 列表",
		Status:      "active",
	}
	if err := db.Create(&setting).Error; err != nil {
		t.Fatalf("create setting: %v", err)
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}, db, setting
}

func newSystemSettingRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.PATCH("/system-settings/:id", handler.updateSystemSetting)
	return router
}

func performSystemSettingPatch(t *testing.T, router *gin.Engine, id uint, body systemSettingRequest) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/system-settings/"+strconv.Itoa(int(id)), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

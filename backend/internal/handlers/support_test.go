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

func TestSupportInfoReturnsActiveSettings(t *testing.T) {
	handler, _ := newSupportTestHandler(t, []models.SystemSetting{
		{Key: "service.customer_service_url", Value: "https://support.example.com", ValueType: "string", Group: "service", Label: "在线客服入口", Status: "active"},
		{Key: "service.customer_service_hours", Value: "09:00-17:30", ValueType: "string", Group: "service", Label: "客服时间", Status: "active"},
		{Key: "service.faq", Value: `[{"question":"如何预约？","answer":"选择套餐和时间后提交预约。"}]`, ValueType: "json", Group: "service", Label: "FAQ", Status: "active"},
	})
	response := performSupportRequest(t, handler)

	payload := decodeSupportPayload(t, response)
	if payload.CustomerServiceURL != "https://support.example.com" || payload.CustomerServiceHours != "09:00-17:30" {
		t.Fatalf("unexpected support settings: %#v", payload)
	}
	if len(payload.FAQ) != 1 || payload.FAQ[0].Question != "如何预约？" || payload.FAQ[0].Answer != "选择套餐和时间后提交预约。" {
		t.Fatalf("unexpected faq: %#v", payload.FAQ)
	}
}

func TestSupportInfoFallsBackWhenFAQIsInvalid(t *testing.T) {
	handler, _ := newSupportTestHandler(t, []models.SystemSetting{
		{Key: "service.faq", Value: `not-json`, ValueType: "json", Group: "service", Label: "FAQ", Status: "active"},
	})
	response := performSupportRequest(t, handler)

	payload := decodeSupportPayload(t, response)
	if len(payload.FAQ) < 3 {
		t.Fatalf("expected fallback FAQ items, got %#v", payload.FAQ)
	}
}

func TestSupportInfoIgnoresInactiveSettings(t *testing.T) {
	handler, _ := newSupportTestHandler(t, []models.SystemSetting{
		{Key: "service.customer_service_url", Value: "https://disabled.example.com", ValueType: "string", Group: "service", Label: "在线客服入口", Status: "disabled"},
		{Key: "service.customer_service_hours", Value: "00:00-01:00", ValueType: "string", Group: "service", Label: "客服时间", Status: "disabled"},
	})
	response := performSupportRequest(t, handler)

	payload := decodeSupportPayload(t, response)
	if payload.CustomerServiceURL != "" || payload.CustomerServiceHours != "" {
		t.Fatalf("inactive settings should not be exposed: %#v", payload)
	}
	if len(payload.FAQ) == 0 {
		t.Fatal("expected fallback FAQ when no active FAQ setting exists")
	}
}

type supportPayload struct {
	CustomerServiceURL   string    `json:"customerServiceUrl"`
	CustomerServiceHours string    `json:"customerServiceHours"`
	FAQ                  []faqItem `json:"faq"`
}

func newSupportTestHandler(t *testing.T, settings []models.SystemSetting) (*Handler, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.SystemSetting{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	for i := range settings {
		if err := db.Create(&settings[i]).Error; err != nil {
			t.Fatalf("create setting: %v", err)
		}
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}, db
}

func performSupportRequest(t *testing.T, handler *Handler) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	router.GET("/support", handler.supportInfo)
	req := httptest.NewRequest(http.MethodGet, "/support", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeSupportPayload(t *testing.T, response *httptest.ResponseRecorder) supportPayload {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload supportPayload
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode support payload: %v", err)
	}
	return payload
}

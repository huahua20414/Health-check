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

func TestMailLogsFilterByStatusAndKeyword(t *testing.T) {
	handler := newLogFilterFixture(t)
	router := newLogFilterRouter(handler)

	response := performLogFilterRequest(t, router, "/mail-logs?page=1&pageSize=10&status=failed&keyword=smtp")

	payload := decodeMailLogPage(t, response)
	if payload.Total != 1 || len(payload.Items) != 1 || payload.Items[0].To != "fail@example.com" {
		t.Fatalf("mail log filter returned wrong page: %#v", payload)
	}
}

func TestLoginLogsFilterByStatusAndKeyword(t *testing.T) {
	handler := newLogFilterFixture(t)
	router := newLogFilterRouter(handler)

	response := performLogFilterRequest(t, router, "/login-logs?page=1&pageSize=10&status=blocked&keyword=10.0.0.9")

	payload := decodeLoginLogPage(t, response)
	if payload.Total != 1 || len(payload.Items) != 1 || payload.Items[0].Email != "blocked@example.com" {
		t.Fatalf("login log filter returned wrong page: %#v", payload)
	}
}

func TestOperationLogsFilterByResourceAndKeyword(t *testing.T) {
	handler := newLogFilterFixture(t)
	router := newLogFilterRouter(handler)

	response := performLogFilterRequest(t, router, "/operation-logs?page=1&pageSize=10&resource=package&keyword=导入")

	payload := decodeOperationLogPage(t, response)
	if payload.Total != 1 || len(payload.Items) != 1 || payload.Items[0].Action != "import" {
		t.Fatalf("operation log filter returned wrong page: %#v", payload)
	}
}

type mailLogPage struct {
	Items []models.MailLog `json:"items"`
	Total int64            `json:"total"`
}

type loginLogPage struct {
	Items []models.LoginLog `json:"items"`
	Total int64             `json:"total"`
}

type operationLogPage struct {
	Items []models.OperationLog `json:"items"`
	Total int64                 `json:"total"`
}

func newLogFilterFixture(t *testing.T) *Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.MailLog{}, &models.LoginLog{}, &models.OperationLog{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	now := time.Now()
	rows := []any{
		&models.MailLog{ID: 1, To: "ok@example.com", Subject: "预约成功", Status: "sent", CreatedAt: now},
		&models.MailLog{ID: 2, To: "fail@example.com", Subject: "报告生成", Status: "failed", Error: "smtp timeout", CreatedAt: now.Add(time.Minute)},
		&models.LoginLog{ID: 10, Email: "ok@example.com", Role: "user", IP: "10.0.0.1", Status: "success", CreatedAt: now},
		&models.LoginLog{ID: 11, Email: "blocked@example.com", Role: "admin", IP: "10.0.0.9", Status: "blocked", Reason: "too many attempts", CreatedAt: now.Add(time.Minute)},
		&models.OperationLog{ID: 20, UserName: "管理员", Action: "update", Resource: "coupon", Status: "success", Detail: "更新优惠券", CreatedAt: now},
		&models.OperationLog{ID: 21, UserName: "管理员", Action: "import", Resource: "package", Status: "success", Detail: "导入套餐 3 条", CreatedAt: now.Add(time.Minute)},
	}
	for _, row := range rows {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create log fixture row %#v: %v", row, err)
		}
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}
}

func newLogFilterRouter(handler *Handler) *gin.Engine {
	router := gin.New()
	router.GET("/mail-logs", handler.mailLogs)
	router.GET("/login-logs", handler.loginLogs)
	router.GET("/operation-logs", handler.operationLogs)
	return router
}

func performLogFilterRequest(t *testing.T, router *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeMailLogPage(t *testing.T, response *httptest.ResponseRecorder) mailLogPage {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload mailLogPage
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode mail log page: %v", err)
	}
	return payload
}

func decodeLoginLogPage(t *testing.T, response *httptest.ResponseRecorder) loginLogPage {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload loginLogPage
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode login log page: %v", err)
	}
	return payload
}

func decodeOperationLogPage(t *testing.T, response *httptest.ResponseRecorder) operationLogPage {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload operationLogPage
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode operation log page: %v", err)
	}
	return payload
}

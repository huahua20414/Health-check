package handlers

import (
	"bytes"
	"encoding/csv"
	"io"
	"mime/multipart"
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

func TestExportCouponsSupportsStatusFilterAndAudits(t *testing.T) {
	handler, db, fixture := newCouponExchangeFixture(t)
	router := newCouponExchangeRouter(handler, fixture.admin)

	response := performCouponExchangeRequest(t, router, http.MethodGet, "/coupons/export?status=active", nil, "")

	records := decodeCouponCSV(t, response)
	if len(records) != 2 {
		t.Fatalf("expected header plus one coupon, got %#v", records)
	}
	if records[1][0] != "SAVE50" || records[1][1] != "立减券" || records[1][4] != "100.00" {
		t.Fatalf("export returned wrong coupon row: %#v", records[1])
	}
	assertCouponOperationLogCount(t, db, fixture.admin.ID, "export", 1)
}

func TestImportCouponsCreatesAndUpdatesByCode(t *testing.T) {
	handler, db, fixture := newCouponExchangeFixture(t)
	router := newCouponExchangeRouter(handler, fixture.admin)
	csvText := strings.Join([]string{
		"code,name,type,value,min_amount,package_id,start_date,end_date,status,description",
		"SAVE50,更新立减券,amount,80,200,20,2026-02-01,2026-11-30,active,批量更新",
		"NEW20,新人折扣,percent,20,0,0,2026-01-01,2026-12-31,active,批量新增",
		"",
	}, "\n")

	response := performCouponExchangeRequest(t, router, http.MethodPost, "/coupons/import", strings.NewReader(csvText), "coupons.csv")

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	assertCouponImportResult(t, response.Body.String(), `"created":1`, `"updated":1`)
	assertCoupon(t, db, "SAVE50", "更新立减券", "amount", 80, 200, 20)
	assertCoupon(t, db, "NEW20", "新人折扣", "percent", 20, 0, 0)
	assertCouponOperationLogCount(t, db, fixture.admin.ID, "import", 1)
}

func TestImportCouponsRejectsInvalidNumericField(t *testing.T) {
	handler, db, fixture := newCouponExchangeFixture(t)
	router := newCouponExchangeRouter(handler, fixture.admin)
	csvText := strings.Join([]string{
		"code,name,type,value,min_amount",
		"BAD,错误券,amount,not-number,0",
		"",
	}, "\n")

	response := performCouponExchangeRequest(t, router, http.MethodPost, "/coupons/import", strings.NewReader(csvText), "coupons.csv")

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "invalid coupon value for BAD")
	assertCouponOperationLogCount(t, db, fixture.admin.ID, "import", 1)
}

type couponExchangeFixture struct {
	admin models.User
}

func newCouponExchangeFixture(t *testing.T) (*Handler, *gorm.DB, couponExchangeFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Coupon{}, &models.OperationLog{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	fixture := couponExchangeFixture{
		admin: models.User{ID: 1, Name: "管理员", Email: "admin-coupon@example.com", Phone: "13800002001", Role: "admin", Status: "active", PasswordHash: "hash"},
	}
	rows := []any{
		&fixture.admin,
		&models.Coupon{ID: 10, Name: "立减券", Code: "SAVE50", Type: "amount", Value: 50, MinAmount: 100, PackageID: 20, StartDate: "2026-01-01", EndDate: "2026-12-31", Status: "active", Description: "可用"},
		&models.Coupon{ID: 11, Name: "归档券", Code: "OLD50", Type: "amount", Value: 50, Status: "deleted"},
	}
	for _, row := range rows {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}, db, fixture
}

func newCouponExchangeRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.GET("/coupons/export", handler.exportCoupons)
	router.POST("/coupons/import", handler.importCoupons)
	return router
}

func performCouponExchangeRequest(t *testing.T, router *gin.Engine, method, path string, body io.Reader, filename string) *httptest.ResponseRecorder {
	t.Helper()
	var requestBody io.Reader
	contentType := ""
	if body != nil {
		var multipartBody bytes.Buffer
		writer := multipart.NewWriter(&multipartBody)
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := io.Copy(part, body); err != nil {
			t.Fatalf("copy csv body: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("close multipart writer: %v", err)
		}
		requestBody = &multipartBody
		contentType = writer.FormDataContentType()
	}
	req := httptest.NewRequest(method, path, requestBody)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeCouponCSV(t *testing.T, response *httptest.ResponseRecorder) [][]string {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	reader := csv.NewReader(strings.NewReader(response.Body.String()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("decode coupon csv: %v", err)
	}
	return records
}

func assertCouponImportResult(t *testing.T, body string, parts ...string) {
	t.Helper()
	for _, part := range parts {
		if !strings.Contains(body, part) {
			t.Fatalf("expected import response to contain %s, got %s", part, body)
		}
	}
}

func assertCoupon(t *testing.T, db *gorm.DB, code, name, couponType string, value, minAmount float64, packageID uint) {
	t.Helper()
	var coupon models.Coupon
	if err := db.Where("code = ?", code).First(&coupon).Error; err != nil {
		t.Fatalf("load coupon %s: %v", code, err)
	}
	if coupon.Name != name || coupon.Type != couponType || coupon.Value != value || coupon.MinAmount != minAmount || coupon.PackageID != packageID {
		t.Fatalf("unexpected coupon %s: %#v", code, coupon)
	}
}

func assertCouponOperationLogCount(t *testing.T, db *gorm.DB, userID uint, action string, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&models.OperationLog{}).Where("user_id = ? AND action = ? AND resource = ?", userID, action, "coupon").Count(&count).Error; err != nil {
		t.Fatalf("count operation logs: %v", err)
	}
	if count != want {
		t.Fatalf("expected %d %s coupon operation logs, got %d", want, action, count)
	}
}

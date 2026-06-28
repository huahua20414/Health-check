package handlers

import (
	"bytes"
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

func TestScheduleTemplateUpsertCreatesSingleTemplateAndGeneratedSlots(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.CheckupInstitution{},
		&models.CheckupPackage{},
		&models.InstitutionPackage{},
		&models.ScheduleTemplate{},
		&models.ScheduleSlot{},
		&models.OperationLog{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	admin := models.User{ID: 1, Name: "管理员", Email: "admin@example.com", Phone: "13800000001", Role: "admin", Status: "active", PasswordHash: "hash"}
	doctor := models.User{ID: 2, Name: "李四", Email: "doctor@example.com", Phone: "13800000002", Role: "doctor", Status: "active", PasswordHash: "hash"}
	institution := models.CheckupInstitution{ID: 10, Name: "总院", Address: "健康路 1 号", Status: "active"}
	pkg := models.CheckupPackage{ID: 20, Name: "入职基础体检", Category: "入职体检", Price: 199, Status: "active"}
	link := models.InstitutionPackage{InstitutionID: institution.ID, PackageID: pkg.ID}
	for _, row := range []any{&admin, &doctor, &institution, &pkg, &link} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}

	handler := &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", admin)
		c.Next()
	})
	router.POST("/schedule/slots", handler.createScheduleSlot)
	router.GET("/schedule/slots", handler.scheduleSlots)

	body, err := json.Marshal(scheduleSlotRequest{
		DoctorID:      doctor.ID,
		InstitutionID: institution.ID,
		Category:      "入职体检",
		Weekdays:      []int{1, 3},
		StartTimes:    []string{"09:00", "15:00"},
		Capacity:      3,
		Status:        "available",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/schedule/slots?template=true", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var templates []models.ScheduleTemplate
	req = httptest.NewRequest(http.MethodGet, "/schedule/slots?template=true", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &templates); err != nil {
		t.Fatalf("decode templates: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("expected 1 template, got %d", len(templates))
	}
	if len(templates[0].Weekdays) != 2 || len(templates[0].StartTimes) != 2 {
		t.Fatalf("unexpected template payload: %#v", templates[0])
	}

	var count int64
	today := time.Now().Format("2006-01-02")
	if err := db.Model(&models.ScheduleSlot{}).Where("template_id = ? AND date >= ? AND status = ?", templates[0].ID, today, "available").Count(&count).Error; err != nil {
		t.Fatalf("count generated slots: %v", err)
	}
	if count == 0 {
		t.Fatalf("expected generated future slots, got %d", count)
	}
}

package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func TestScheduleTemplateUpdateAllowsRemovingPartialStartTimes(t *testing.T) {
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
	router.PATCH("/schedule/slots/:id", handler.updateScheduleSlot)
	router.GET("/schedule/slots", handler.scheduleSlots)

	createBody, _ := json.Marshal(scheduleSlotRequest{
		DoctorID:      doctor.ID,
		InstitutionID: institution.ID,
		Category:      "入职体检",
		Weekdays:      []int{1},
		StartTimes:    []string{"08:00", "08:30", "09:00"},
		Capacity:      3,
		Status:        "available",
	})
	createReq := httptest.NewRequest(http.MethodPost, "/schedule/slots?template=true", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d: %s", createRec.Code, createRec.Body.String())
	}

	var created models.ScheduleTemplate
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode template: %v", err)
	}

	updateBody, _ := json.Marshal(scheduleSlotRequest{
		DoctorID:      doctor.ID,
		InstitutionID: institution.ID,
		Category:      "入职体检",
		Weekdays:      []int{1},
		StartTimes:    []string{"08:00", "09:00"},
		Capacity:      3,
		Status:        "available",
	})
	updateReq := httptest.NewRequest(http.MethodPatch, "/schedule/slots/"+strconv.Itoa(int(created.ID))+"?template=true", bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	router.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected update 200, got %d: %s", updateRec.Code, updateRec.Body.String())
	}

	var slots []models.ScheduleSlot
	if err := db.Where("template_id = ? AND status = ?", created.ID, "available").Find(&slots).Error; err != nil {
		t.Fatalf("load slots: %v", err)
	}
	for _, slot := range slots {
		if slot.StartTime == "08:30" {
			t.Fatalf("expected removed start time not to remain available: %#v", slot)
		}
	}
}

func TestScheduleTemplateUpdateReusesScopedFutureSlots(t *testing.T) {
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
	router.PATCH("/schedule/slots/:id", handler.updateScheduleSlot)

	createBody, _ := json.Marshal(scheduleSlotRequest{
		DoctorID:      doctor.ID,
		InstitutionID: institution.ID,
		Category:      "入职体检",
		Weekdays:      []int{1},
		StartTimes:    []string{"08:00"},
		Capacity:      3,
		Status:        "available",
	})
	createReq := httptest.NewRequest(http.MethodPost, "/schedule/slots?template=true", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d: %s", createRec.Code, createRec.Body.String())
	}

	var created models.ScheduleTemplate
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode template: %v", err)
	}

	var future models.ScheduleSlot
	if err := db.Where("template_id = ? AND start_time = ?", created.ID, "08:00").First(&future).Error; err != nil {
		t.Fatalf("load generated slot: %v", err)
	}
	loose := models.ScheduleSlot{
		DoctorID:      doctor.ID,
		InstitutionID: institution.ID,
		Date:          future.Date,
		Period:        "上午",
		Category:      "入职体检",
		StartTime:     "09:00",
		EndTime:       "09:30",
		Capacity:      2,
		Status:        "available",
	}
	if err := db.Create(&loose).Error; err != nil {
		t.Fatalf("create loose scoped slot: %v", err)
	}

	updateBody, _ := json.Marshal(scheduleSlotRequest{
		DoctorID:      doctor.ID,
		InstitutionID: institution.ID,
		Category:      "入职体检",
		Weekdays:      []int{1},
		StartTimes:    []string{"08:00", "09:00"},
		Capacity:      3,
		Status:        "available",
	})
	updateReq := httptest.NewRequest(http.MethodPatch, "/schedule/slots/"+strconv.Itoa(int(created.ID))+"?template=true", bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	router.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected update 200, got %d: %s", updateRec.Code, updateRec.Body.String())
	}

	var rebound models.ScheduleSlot
	if err := db.First(&rebound, loose.ID).Error; err != nil {
		t.Fatalf("reload rebound slot: %v", err)
	}
	if rebound.TemplateID != created.ID {
		t.Fatalf("expected scoped slot to be rebound to template, got template_id=%d", rebound.TemplateID)
	}
}

func TestArchiveScheduleTemplateRemovesItFromTemplateQuery(t *testing.T) {
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
	router.DELETE("/schedule/slots/:id", handler.archiveScheduleSlot)
	router.GET("/schedule/slots", handler.scheduleSlots)

	createBody, _ := json.Marshal(scheduleSlotRequest{
		DoctorID:      doctor.ID,
		InstitutionID: institution.ID,
		Category:      "入职体检",
		Weekdays:      []int{1},
		StartTimes:    []string{"08:00"},
		Capacity:      1,
		Status:        "available",
	})
	createReq := httptest.NewRequest(http.MethodPost, "/schedule/slots?template=true", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d: %s", createRec.Code, createRec.Body.String())
	}

	var created models.ScheduleTemplate
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode template: %v", err)
	}

	archiveReq := httptest.NewRequest(http.MethodDelete, "/schedule/slots/"+strconv.Itoa(int(created.ID))+"?template=true", nil)
	archiveRec := httptest.NewRecorder()
	router.ServeHTTP(archiveRec, archiveReq)
	if archiveRec.Code != http.StatusOK {
		t.Fatalf("expected archive 200, got %d: %s", archiveRec.Code, archiveRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/schedule/slots?template=true&doctorId=2&institutionId=10&category=入职体检", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", listRec.Code, listRec.Body.String())
	}

	var templates []models.ScheduleTemplate
	if err := json.Unmarshal(listRec.Body.Bytes(), &templates); err != nil {
		t.Fatalf("decode templates: %v", err)
	}
	if len(templates) != 0 {
		t.Fatalf("expected archived template to disappear from template query, got %d", len(templates))
	}
}

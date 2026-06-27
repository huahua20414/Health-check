package handlers

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"health-checkup/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAdminCannotDisableOwnAccount(t *testing.T) {
	handler, db, fixture := newUserStatusFixture(t)
	router := newUserStatusRouter(handler, fixture.admin)

	response := performUserStatusPatch(t, router, fixture.admin.ID, statusRequest{Status: "disabled"})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertUserStatus(t, db, fixture.admin.ID, "active")
}

func TestCannotDisableLastActiveAdmin(t *testing.T) {
	handler, db, fixture := newUserStatusFixture(t)
	if err := db.Model(&models.User{}).Where("id = ?", fixture.otherAdmin.ID).Update("status", "disabled").Error; err != nil {
		t.Fatalf("disable other admin: %v", err)
	}
	router := newUserStatusRouter(handler, fixture.otherAdmin)

	response := performUserStatusPatch(t, router, fixture.admin.ID, statusRequest{Status: "disabled"})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertUserStatus(t, db, fixture.admin.ID, "active")
}

func TestAdminCanDisableAnotherAdminWhenAnotherActiveAdminRemains(t *testing.T) {
	handler, db, fixture := newUserStatusFixture(t)
	router := newUserStatusRouter(handler, fixture.admin)

	response := performUserStatusPatch(t, router, fixture.otherAdmin.ID, statusRequest{Status: "disabled"})

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	assertUserStatus(t, db, fixture.otherAdmin.ID, "disabled")
	assertOperationCount(t, db, "update_status", "user", 1)
}

func TestAdminCanReactivateDisabledUser(t *testing.T) {
	handler, db, fixture := newUserStatusFixture(t)
	router := newUserStatusRouter(handler, fixture.admin)

	response := performUserStatusPatch(t, router, fixture.disabledUser.ID, statusRequest{Status: "active"})

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	assertUserStatus(t, db, fixture.disabledUser.ID, "active")
}

func TestAdminActivatingDoctorEnsuresFutureScheduleSlots(t *testing.T) {
	handler, db, fixture := newUserStatusFixture(t)
	institution := models.CheckupInstitution{ID: 10, Name: "已有排班机构", Address: "健康路 1 号", Status: "active"}
	pkg := models.CheckupPackage{ID: 20, Name: "入职基础体检", Category: "入职体检", Price: 199, Status: "active"}
	activeDoctor := models.User{ID: 30, Name: "已有医生", Email: "active-doctor@example.com", Phone: "D001", Role: "doctor", Status: "active", PasswordHash: "hash"}
	pendingDoctor := models.User{ID: 31, Name: "待审医生", Email: "pending-doctor@example.com", Phone: "D002", Role: "doctor", Status: "pending", PasswordHash: "hash"}
	templateSlot := models.ScheduleSlot{ID: 40, DoctorID: activeDoctor.ID, InstitutionID: institution.ID, Date: time.Now().Format("2006-01-02"), Period: "上午", Category: "入职体检", StartTime: "09:00", EndTime: "09:30", Capacity: 1, Status: "available"}
	for _, row := range []any{&institution, &pkg, &activeDoctor, &pendingDoctor, &templateSlot} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create row %#v: %v", row, err)
		}
	}
	router := newUserStatusRouter(handler, fixture.admin)

	response := performUserStatusPatch(t, router, pendingDoctor.ID, statusRequest{Status: "active"})

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	assertUserStatus(t, db, pendingDoctor.ID, "active")
	assertMinHandlerCount(t, db, &models.ScheduleSlot{}, "doctor_id = ? AND institution_id = ?", []any{pendingDoctor.ID, institution.ID}, 1)
}

func TestAdminCanUpdateUserProfile(t *testing.T) {
	handler, db, fixture := newUserStatusFixture(t)
	router := newUserStatusRouter(handler, fixture.admin)
	emailNotify := false

	response := performUserPatch(t, router, fixture.disabledUser.ID, adminUserRequest{
		Name:        "更新用户",
		Email:       "updated@example.com",
		Gender:      "男",
		IDCard:      "11010519491231002X",
		Bio:         "重点客户",
		EmailNotify: &emailNotify,
		Status:      "active",
	})

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var user models.User
	if err := db.First(&user, fixture.disabledUser.ID).Error; err != nil {
		t.Fatalf("load updated user: %v", err)
	}
	if user.Name != "更新用户" || user.Email != "updated@example.com" || user.Age <= 0 || user.EmailNotify {
		t.Fatalf("unexpected updated user: %#v", user)
	}
	assertUserOperationLogCount(t, db, fixture.admin.ID, "update", "user", 1)
}

func TestAdminUpdateUserRejectsDoctorAccount(t *testing.T) {
	handler, db, fixture := newUserStatusFixture(t)
	doctor := models.User{ID: 4, Name: "医生", Email: "doctor@example.com", Phone: "13800000004", Role: "doctor", Status: "active", PasswordHash: "hash"}
	if err := db.Create(&doctor).Error; err != nil {
		t.Fatalf("create doctor: %v", err)
	}
	router := newUserStatusRouter(handler, fixture.admin)

	response := performUserPatch(t, router, doctor.ID, adminUserRequest{
		Name:   "改名医生",
		Email:  "changed-doctor@example.com",
		Status: "active",
	})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	var updated models.User
	if err := db.First(&updated, doctor.ID).Error; err != nil {
		t.Fatalf("load doctor: %v", err)
	}
	if updated.Name != doctor.Name || updated.Email != doctor.Email {
		t.Fatalf("doctor account was edited through user endpoint: %#v", updated)
	}
}

func TestAdminUpdateUserRejectsDuplicateEmail(t *testing.T) {
	handler, db, fixture := newUserStatusFixture(t)
	router := newUserStatusRouter(handler, fixture.admin)

	response := performUserPatch(t, router, fixture.disabledUser.ID, adminUserRequest{
		Name:   "停用用户",
		Email:  fixture.otherAdmin.Email,
		Status: "active",
	})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	var user models.User
	if err := db.First(&user, fixture.disabledUser.ID).Error; err != nil {
		t.Fatalf("load user: %v", err)
	}
	if user.Email == fixture.otherAdmin.Email {
		t.Fatalf("duplicate email was saved: %#v", user)
	}
}

func TestAdminUpdateUserCannotDisableSelf(t *testing.T) {
	handler, db, fixture := newUserStatusFixture(t)
	router := newUserStatusRouter(handler, fixture.admin)

	response := performUserPatch(t, router, fixture.admin.ID, adminUserRequest{
		Name:   fixture.admin.Name,
		Email:  fixture.admin.Email,
		Status: "disabled",
	})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertUserStatus(t, db, fixture.admin.ID, "active")
}

func TestAdminExportUsersUsesFiltersAndAudits(t *testing.T) {
	handler, db, fixture := newUserStatusFixture(t)
	router := newUserStatusRouter(handler, fixture.admin)

	response := performUserStatusRequest(t, router, http.MethodGet, "/users/export?role=admin&status=active&keyword=管理员乙")

	records := decodeUserCSV(t, response)
	if len(records) != 2 {
		t.Fatalf("expected header plus one user, got %#v", records)
	}
	if records[1][0] != strconv.Itoa(int(fixture.otherAdmin.ID)) || records[1][1] != fixture.otherAdmin.Name || records[1][4] != "admin" {
		t.Fatalf("export returned wrong user row: %#v", records[1])
	}
	assertUserOperationLogCount(t, db, fixture.admin.ID, "export", "user", 1)
}

type userStatusFixture struct {
	admin        models.User
	otherAdmin   models.User
	disabledUser models.User
}

func newUserStatusFixture(t *testing.T) (*Handler, *gorm.DB, userStatusFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.OperationLog{}, &models.CheckupInstitution{}, &models.CheckupPackage{}, &models.ScheduleSlot{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	fixture := userStatusFixture{
		admin:        models.User{ID: 1, Name: "管理员甲", Email: "admin@example.com", Phone: "13800000001", Role: "admin", Status: "active", PasswordHash: "hash"},
		otherAdmin:   models.User{ID: 2, Name: "管理员乙", Email: "admin2@example.com", Phone: "13800000002", Role: "admin", Status: "active", PasswordHash: "hash"},
		disabledUser: models.User{ID: 3, Name: "停用用户", Email: "disabled@example.com", Phone: "13800000003", Role: "user", Status: "disabled", PasswordHash: "hash"},
	}
	for _, row := range []any{&fixture.admin, &fixture.otherAdmin, &fixture.disabledUser} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}, db, fixture
}

func assertMinHandlerCount(t *testing.T, db *gorm.DB, model any, condition string, args []any, min int64) {
	t.Helper()
	var count int64
	if err := db.Model(model).Where(condition, args...).Count(&count).Error; err != nil {
		t.Fatalf("count %T: %v", model, err)
	}
	if count < min {
		t.Fatalf("expected at least %d rows for %T, got %d", min, model, count)
	}
}

func newUserStatusRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.GET("/users/export", handler.exportUsers)
	router.PATCH("/users/:id", handler.updateUser)
	router.PATCH("/users/:id/status", handler.updateUserStatus)
	return router
}

func performUserStatusRequest(t *testing.T, router *gin.Engine, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func performUserStatusPatch(t *testing.T, router *gin.Engine, id uint, body statusRequest) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/users/"+strconv.Itoa(int(id))+"/status", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func performUserPatch(t *testing.T, router *gin.Engine, id uint, body adminUserRequest) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/users/"+strconv.Itoa(int(id)), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeUserCSV(t *testing.T, response *httptest.ResponseRecorder) [][]string {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	reader := csv.NewReader(strings.NewReader(response.Body.String()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("decode user csv: %v", err)
	}
	return records
}

func assertUserStatus(t *testing.T, db *gorm.DB, id uint, want string) {
	t.Helper()
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		t.Fatalf("load user: %v", err)
	}
	if user.Status != want {
		t.Fatalf("expected user %d status %s, got %s", id, want, user.Status)
	}
}

func assertUserOperationLogCount(t *testing.T, db *gorm.DB, userID uint, action, resource string, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&models.OperationLog{}).Where("user_id = ? AND action = ? AND resource = ?", userID, action, resource).Count(&count).Error; err != nil {
		t.Fatalf("count operation logs: %v", err)
	}
	if count != want {
		t.Fatalf("expected %d %s/%s operation logs, got %d", want, action, resource, count)
	}
}

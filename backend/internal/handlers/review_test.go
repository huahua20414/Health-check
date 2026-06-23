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

func TestCreateReviewRequiresCurrentUsersCompletedAppointment(t *testing.T) {
	handler, db, fixture := newReviewFixture(t)
	router := newReviewRouter(handler, fixture.user)

	response := performReviewRequest(t, router, http.MethodPost, "/reviews", reviewRequest{
		AppointmentID: fixture.reportedAppointment.ID,
		Rating:        5,
		Content:       "服务很细致",
	})

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	var review models.ServiceReview
	if err := db.Where("appointment_id = ?", fixture.reportedAppointment.ID).First(&review).Error; err != nil {
		t.Fatalf("load created review: %v", err)
	}
	if review.UserID != fixture.user.ID || review.PackageID != fixture.pkg.ID || review.InstitutionID != fixture.institution.ID || review.DoctorID != fixture.doctor.ID {
		t.Fatalf("review does not match appointment context: %#v", review)
	}
	if review.Rating != 5 || review.Content != "服务很细致" || review.Status != "published" {
		t.Fatalf("review content not saved correctly: %#v", review)
	}
}

func TestCreateReviewRejectsOtherUsersAppointment(t *testing.T) {
	handler, db, fixture := newReviewFixture(t)
	router := newReviewRouter(handler, fixture.user)

	response := performReviewRequest(t, router, http.MethodPost, "/reviews", reviewRequest{
		AppointmentID: fixture.otherAppointment.ID,
		Rating:        5,
		Content:       "越权评价",
	})

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "appointment not found")
	assertReviewCount(t, db, fixture.otherAppointment.ID, 0)
}

func TestCreateReviewRejectsUnfinishedAppointment(t *testing.T) {
	handler, db, fixture := newReviewFixture(t)
	router := newReviewRouter(handler, fixture.user)

	response := performReviewRequest(t, router, http.MethodPost, "/reviews", reviewRequest{
		AppointmentID: fixture.bookedAppointment.ID,
		Rating:        4,
		Content:       "未完成也评价",
	})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "review can only be created after checkup")
	assertReviewCount(t, db, fixture.bookedAppointment.ID, 0)
}

func TestCreateReviewRejectsDuplicateAppointmentReview(t *testing.T) {
	handler, _, fixture := newReviewFixture(t)
	router := newReviewRouter(handler, fixture.user)

	first := performReviewRequest(t, router, http.MethodPost, "/reviews", reviewRequest{
		AppointmentID: fixture.reportedAppointment.ID,
		Rating:        5,
		Content:       "第一次评价",
	})
	second := performReviewRequest(t, router, http.MethodPost, "/reviews", reviewRequest{
		AppointmentID: fixture.reportedAppointment.ID,
		Rating:        4,
		Content:       "重复评价",
	})

	if first.Code != http.StatusCreated {
		t.Fatalf("expected first review to succeed, got %d: %s", first.Code, first.Body.String())
	}
	if second.Code != http.StatusBadRequest {
		t.Fatalf("expected duplicate review to fail, got %d: %s", second.Code, second.Body.String())
	}
	assertErrorMessage(t, second.Body.Bytes(), "appointment has already been reviewed")
}

func TestCreateReviewRejectsInvalidRating(t *testing.T) {
	handler, _, fixture := newReviewFixture(t)
	router := newReviewRouter(handler, fixture.user)

	response := performReviewRequest(t, router, http.MethodPost, "/reviews", reviewRequest{
		AppointmentID: fixture.reportedAppointment.ID,
		Rating:        6,
		Content:       "分数非法",
	})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "rating must be between 1 and 5")
}

func TestReviewsAreScopedByRole(t *testing.T) {
	handler, _, fixture := newReviewFixture(t)
	createSeedReview(t, handler.db, fixture)

	userResponse := performReviewRequest(t, newReviewRouter(handler, fixture.user), http.MethodGet, "/reviews", nil)
	doctorResponse := performReviewRequest(t, newReviewRouter(handler, fixture.doctor), http.MethodGet, "/reviews", nil)
	adminResponse := performReviewRequest(t, newReviewRouter(handler, fixture.admin), http.MethodGet, "/reviews", nil)

	assertReviewListLength(t, userResponse, 1)
	assertReviewListLength(t, doctorResponse, 1)
	assertReviewListLength(t, adminResponse, 2)
}

func TestAdminCanReplyReviewAndHideIt(t *testing.T) {
	handler, db, fixture := newReviewFixture(t)
	review := createSeedReview(t, db, fixture)
	router := newReviewRouter(handler, fixture.admin)

	response := performReviewRequest(t, router, http.MethodPatch, "/reviews/"+strconv.Itoa(int(review.ID))+"/reply", reviewReplyRequest{
		Reply:  "感谢反馈，我们会继续优化。",
		Status: "hidden",
	})

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var updated models.ServiceReview
	if err := db.First(&updated, review.ID).Error; err != nil {
		t.Fatalf("load updated review: %v", err)
	}
	if updated.Reply != "感谢反馈，我们会继续优化。" || updated.Status != "hidden" {
		t.Fatalf("review reply not saved correctly: %#v", updated)
	}
}

func TestNonAdminCannotReplyReview(t *testing.T) {
	handler, db, fixture := newReviewFixture(t)
	review := createSeedReview(t, db, fixture)
	router := newReviewRouter(handler, fixture.user)

	response := performReviewRequest(t, router, http.MethodPatch, "/reviews/"+strconv.Itoa(int(review.ID))+"/reply", reviewReplyRequest{
		Reply:  "用户不能回复",
		Status: "hidden",
	})

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "permission denied")
	var unchanged models.ServiceReview
	if err := db.First(&unchanged, review.ID).Error; err != nil {
		t.Fatalf("load review: %v", err)
	}
	if unchanged.Reply != "" || unchanged.Status != "published" {
		t.Fatalf("review should not be changed by non-admin: %#v", unchanged)
	}
}

func TestReplyReviewRejectsInvalidStatus(t *testing.T) {
	handler, db, fixture := newReviewFixture(t)
	review := createSeedReview(t, db, fixture)
	router := newReviewRouter(handler, fixture.admin)

	response := performReviewRequest(t, router, http.MethodPatch, "/reviews/"+strconv.Itoa(int(review.ID))+"/reply", reviewReplyRequest{
		Reply:  "非法状态",
		Status: "deleted",
	})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "invalid review status")
}

type reviewFixture struct {
	user                models.User
	otherUser           models.User
	doctor              models.User
	otherDoctor         models.User
	admin               models.User
	institution         models.CheckupInstitution
	pkg                 models.CheckupPackage
	reportedAppointment models.Appointment
	bookedAppointment   models.Appointment
	otherAppointment    models.Appointment
}

func newReviewFixture(t *testing.T) (*Handler, *gorm.DB, reviewFixture) {
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
		&models.Appointment{},
		&models.ServiceReview{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	fixture := reviewFixture{
		user:                models.User{ID: 100, Name: "用户", Phone: "13800000100", Email: "user@example.com", Role: "user", Status: "active", PasswordHash: "hash"},
		otherUser:           models.User{ID: 101, Name: "其他用户", Phone: "13800000101", Email: "other@example.com", Role: "user", Status: "active", PasswordHash: "hash"},
		doctor:              models.User{ID: 200, Name: "医生", Phone: "13800000200", Email: "doctor@example.com", Role: "doctor", Status: "active", PasswordHash: "hash"},
		otherDoctor:         models.User{ID: 201, Name: "其他医生", Phone: "13800000201", Email: "doctor2@example.com", Role: "doctor", Status: "active", PasswordHash: "hash"},
		admin:               models.User{ID: 300, Name: "管理员", Phone: "13800000300", Email: "admin@example.com", Role: "admin", Status: "active", PasswordHash: "hash"},
		institution:         models.CheckupInstitution{ID: 10, Name: "主院区", Address: "健康路 1 号", Status: "active"},
		pkg:                 models.CheckupPackage{ID: 20, Name: "年度体检", Category: "年度综合", Price: 399, Items: "血常规", Status: "active"},
		reportedAppointment: models.Appointment{ID: 30, OrderNo: "HC202607010001", UserID: 100, DoctorID: 200, InstitutionID: 10, PackageID: 20, AppointmentType: "个人体检", Category: "年度综合", Date: "2026-07-01", Period: "上午", Status: "reported"},
		bookedAppointment:   models.Appointment{ID: 31, OrderNo: "HC202607010002", UserID: 100, DoctorID: 200, InstitutionID: 10, PackageID: 20, AppointmentType: "个人体检", Category: "年度综合", Date: "2026-07-02", Period: "上午", Status: "booked"},
		otherAppointment:    models.Appointment{ID: 32, OrderNo: "HC202607010003", UserID: 101, DoctorID: 201, InstitutionID: 10, PackageID: 20, AppointmentType: "个人体检", Category: "年度综合", Date: "2026-07-03", Period: "上午", Status: "reported"},
	}
	for _, row := range []any{
		&fixture.user,
		&fixture.otherUser,
		&fixture.doctor,
		&fixture.otherDoctor,
		&fixture.admin,
		&fixture.institution,
		&fixture.pkg,
		&fixture.reportedAppointment,
		&fixture.bookedAppointment,
		&fixture.otherAppointment,
	} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}
	handler := &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}
	return handler, db, fixture
}

func newReviewRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.GET("/reviews", handler.reviews)
	router.POST("/reviews", handler.createReview)
	router.PATCH("/reviews/:id/reply", handler.requireRole("admin"), handler.replyReview)
	return router
}

func performReviewRequest(t *testing.T, router *gin.Engine, method, path string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func createSeedReview(t *testing.T, db *gorm.DB, fixture reviewFixture) models.ServiceReview {
	t.Helper()
	review := models.ServiceReview{
		UserID:        fixture.user.ID,
		AppointmentID: fixture.reportedAppointment.ID,
		PackageID:     fixture.pkg.ID,
		InstitutionID: fixture.institution.ID,
		DoctorID:      fixture.otherDoctor.ID,
		Rating:        4,
		Content:       "已有评价",
		Status:        "published",
	}
	if err := db.Create(&review).Error; err != nil {
		t.Fatalf("create seed review: %v", err)
	}
	otherReview := models.ServiceReview{
		UserID:        fixture.otherUser.ID,
		AppointmentID: fixture.otherAppointment.ID,
		PackageID:     fixture.pkg.ID,
		InstitutionID: fixture.institution.ID,
		DoctorID:      fixture.doctor.ID,
		Rating:        3,
		Content:       "其他用户评价",
		Status:        "published",
	}
	if err := db.Create(&otherReview).Error; err != nil {
		t.Fatalf("create other review: %v", err)
	}
	return review
}

func assertReviewCount(t *testing.T, db *gorm.DB, appointmentID uint, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&models.ServiceReview{}).Where("appointment_id = ?", appointmentID).Count(&count).Error; err != nil {
		t.Fatalf("count reviews: %v", err)
	}
	if count != want {
		t.Fatalf("expected %d reviews for appointment %d, got %d", want, appointmentID, count)
	}
}

func assertReviewListLength(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var reviews []models.ServiceReview
	if err := json.Unmarshal(response.Body.Bytes(), &reviews); err != nil {
		t.Fatalf("decode reviews: %v", err)
	}
	if len(reviews) != want {
		t.Fatalf("expected %d reviews, got %d: %#v", want, len(reviews), reviews)
	}
}

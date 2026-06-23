package handlers

import (
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

func TestFavoritePackageCreatesSingleFavoriteForCurrentUser(t *testing.T) {
	handler, db, fixture := newPackageEngagementFixture(t)
	router := newPackageEngagementRouter(handler, fixture.user)

	first := performPackageEngagementRequest(t, router, http.MethodPost, "/package-favorites/"+strconv.Itoa(int(fixture.activePackage.ID)))
	second := performPackageEngagementRequest(t, router, http.MethodPost, "/package-favorites/"+strconv.Itoa(int(fixture.activePackage.ID)))

	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("expected both favorite requests to succeed, got %d/%d: %s / %s", first.Code, second.Code, first.Body.String(), second.Body.String())
	}
	assertFavoriteCount(t, db, fixture.user.ID, fixture.activePackage.ID, 1)
	assertFavoriteCount(t, db, fixture.otherUser.ID, fixture.activePackage.ID, 1)
}

func TestPackageFavoritesOnlyReturnsCurrentUsersFavorites(t *testing.T) {
	handler, _, fixture := newPackageEngagementFixture(t)
	router := newPackageEngagementRouter(handler, fixture.user)

	response := performPackageEngagementRequest(t, router, http.MethodGet, "/package-favorites")

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var favorites []models.PackageFavorite
	if err := json.Unmarshal(response.Body.Bytes(), &favorites); err != nil {
		t.Fatalf("decode favorites: %v", err)
	}
	if len(favorites) != 1 {
		t.Fatalf("expected one current-user favorite, got %d: %#v", len(favorites), favorites)
	}
	if favorites[0].UserID != fixture.user.ID || favorites[0].PackageID != fixture.seedFavorite.PackageID {
		t.Fatalf("unexpected favorite list: %#v", favorites)
	}
}

func TestFavoritePackageRejectsInactivePackage(t *testing.T) {
	handler, db, fixture := newPackageEngagementFixture(t)
	router := newPackageEngagementRouter(handler, fixture.user)

	response := performPackageEngagementRequest(t, router, http.MethodPost, "/package-favorites/"+strconv.Itoa(int(fixture.inactivePackage.ID)))

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "package not found")
	assertFavoriteCount(t, db, fixture.user.ID, fixture.inactivePackage.ID, 0)
}

func TestUnfavoritePackageOnlyDeletesCurrentUsersFavorite(t *testing.T) {
	handler, db, fixture := newPackageEngagementFixture(t)
	router := newPackageEngagementRouter(handler, fixture.user)

	response := performPackageEngagementRequest(t, router, http.MethodDelete, "/package-favorites/"+strconv.Itoa(int(fixture.activePackage.ID)))

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	assertFavoriteCount(t, db, fixture.user.ID, fixture.activePackage.ID, 0)
	assertFavoriteCount(t, db, fixture.otherUser.ID, fixture.activePackage.ID, 1)
}

func TestRecordPackageBrowseCreatesAndIncrementsCurrentUsersHistory(t *testing.T) {
	handler, db, fixture := newPackageEngagementFixture(t)
	router := newPackageEngagementRouter(handler, fixture.user)

	first := performPackageEngagementRequest(t, router, http.MethodPost, "/packages/"+strconv.Itoa(int(fixture.secondPackage.ID))+"/browse")
	second := performPackageEngagementRequest(t, router, http.MethodPost, "/packages/"+strconv.Itoa(int(fixture.secondPackage.ID))+"/browse")

	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("expected both browse requests to succeed, got %d/%d: %s / %s", first.Code, second.Code, first.Body.String(), second.Body.String())
	}
	assertBrowseHistory(t, db, fixture.user.ID, fixture.secondPackage.ID, 2)
	assertBrowseHistory(t, db, fixture.otherUser.ID, fixture.secondPackage.ID, 5)
}

func TestPackageBrowsesOnlyReturnsCurrentUsersHistory(t *testing.T) {
	handler, _, fixture := newPackageEngagementFixture(t)
	router := newPackageEngagementRouter(handler, fixture.user)

	response := performPackageEngagementRequest(t, router, http.MethodGet, "/package-browses")

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var histories []models.PackageBrowseHistory
	if err := json.Unmarshal(response.Body.Bytes(), &histories); err != nil {
		t.Fatalf("decode histories: %v", err)
	}
	if len(histories) != 1 {
		t.Fatalf("expected one current-user history, got %d: %#v", len(histories), histories)
	}
	if histories[0].UserID != fixture.user.ID || histories[0].PackageID != fixture.seedBrowse.PackageID {
		t.Fatalf("unexpected browse list: %#v", histories)
	}
}

func TestRecordPackageBrowseRejectsInactivePackage(t *testing.T) {
	handler, db, fixture := newPackageEngagementFixture(t)
	router := newPackageEngagementRouter(handler, fixture.user)

	response := performPackageEngagementRequest(t, router, http.MethodPost, "/packages/"+strconv.Itoa(int(fixture.inactivePackage.ID))+"/browse")

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "package not found")
	assertBrowseHistory(t, db, fixture.user.ID, fixture.inactivePackage.ID, 0)
}

type packageEngagementFixture struct {
	user            models.User
	otherUser       models.User
	activePackage   models.CheckupPackage
	secondPackage   models.CheckupPackage
	inactivePackage models.CheckupPackage
	seedFavorite    models.PackageFavorite
	otherFavorite   models.PackageFavorite
	seedBrowse      models.PackageBrowseHistory
	otherBrowse     models.PackageBrowseHistory
}

func newPackageEngagementFixture(t *testing.T) (*Handler, *gorm.DB, packageEngagementFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.CheckupPackage{}, &models.PackageFavorite{}, &models.PackageBrowseHistory{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	fixture := packageEngagementFixture{
		user:            models.User{ID: 100, Name: "用户", Phone: "13800000100", Email: "user@example.com", Role: "user", Status: "active", PasswordHash: "hash"},
		otherUser:       models.User{ID: 200, Name: "其他用户", Phone: "13800000200", Email: "other@example.com", Role: "user", Status: "active", PasswordHash: "hash"},
		activePackage:   models.CheckupPackage{ID: 10, Name: "年度体检", Category: "年度综合", Price: 399, Items: "血常规", Status: "active"},
		secondPackage:   models.CheckupPackage{ID: 11, Name: "入职体检", Category: "入职体检", Price: 199, Items: "胸片", Status: "active"},
		inactivePackage: models.CheckupPackage{ID: 12, Name: "下架套餐", Category: "年度综合", Price: 299, Items: "血常规", Status: "disabled"},
		seedFavorite:    models.PackageFavorite{ID: 20, UserID: 100, PackageID: 10},
		otherFavorite:   models.PackageFavorite{ID: 21, UserID: 200, PackageID: 10},
		seedBrowse:      models.PackageBrowseHistory{ID: 30, UserID: 100, PackageID: 10, ViewCount: 3},
		otherBrowse:     models.PackageBrowseHistory{ID: 31, UserID: 200, PackageID: 11, ViewCount: 5},
	}
	for _, row := range []any{
		&fixture.user,
		&fixture.otherUser,
		&fixture.activePackage,
		&fixture.secondPackage,
		&fixture.inactivePackage,
		&fixture.seedFavorite,
		&fixture.otherFavorite,
		&fixture.seedBrowse,
		&fixture.otherBrowse,
	} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}
	handler := &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}
	return handler, db, fixture
}

func newPackageEngagementRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.GET("/package-favorites", handler.packageFavorites)
	router.POST("/package-favorites/:id", handler.favoritePackage)
	router.DELETE("/package-favorites/:id", handler.unfavoritePackage)
	router.POST("/packages/:id/browse", handler.recordPackageBrowse)
	router.GET("/package-browses", handler.packageBrowses)
	return router
}

func performPackageEngagementRequest(t *testing.T, router *gin.Engine, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func assertFavoriteCount(t *testing.T, db *gorm.DB, userID, packageID uint, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&models.PackageFavorite{}).Where("user_id = ? AND package_id = ?", userID, packageID).Count(&count).Error; err != nil {
		t.Fatalf("count favorites: %v", err)
	}
	if count != want {
		t.Fatalf("expected favorite count %d for user=%d package=%d, got %d", want, userID, packageID, count)
	}
}

func assertBrowseHistory(t *testing.T, db *gorm.DB, userID, packageID uint, wantViews int) {
	t.Helper()
	var history models.PackageBrowseHistory
	err := db.Where("user_id = ? AND package_id = ?", userID, packageID).First(&history).Error
	if wantViews == 0 {
		if err == nil {
			t.Fatalf("expected no browse history for user=%d package=%d, got %#v", userID, packageID, history)
		}
		return
	}
	if err != nil {
		t.Fatalf("load browse history: %v", err)
	}
	if history.ViewCount != wantViews {
		t.Fatalf("expected browse views %d, got %d", wantViews, history.ViewCount)
	}
}

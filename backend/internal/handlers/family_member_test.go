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

func TestFamilyMembersOnlyReturnsCurrentUsersMembers(t *testing.T) {
	handler, _, fixture := newFamilyMemberFixture(t)
	router := newFamilyMemberTestRouter(handler, fixture.user)

	response := performFamilyRequest(t, router, http.MethodGet, "/family-members", nil)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var members []models.FamilyMember
	if err := json.Unmarshal(response.Body.Bytes(), &members); err != nil {
		t.Fatalf("decode members: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("expected one current-user member, got %d: %#v", len(members), members)
	}
	if members[0].UserID != fixture.user.ID || members[0].Name != "父亲" {
		t.Fatalf("unexpected member list: %#v", members)
	}
}

func TestFamilyMembersHidesDeletedMembers(t *testing.T) {
	handler, db, fixture := newFamilyMemberFixture(t)
	if err := db.Model(&models.FamilyMember{}).Where("id = ?", fixture.member.ID).Update("status", "deleted").Error; err != nil {
		t.Fatalf("archive member: %v", err)
	}
	router := newFamilyMemberTestRouter(handler, fixture.user)

	response := performFamilyRequest(t, router, http.MethodGet, "/family-members", nil)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var members []models.FamilyMember
	if err := json.Unmarshal(response.Body.Bytes(), &members); err != nil {
		t.Fatalf("decode members: %v", err)
	}
	if len(members) != 0 {
		t.Fatalf("deleted members should be hidden, got %#v", members)
	}
	if handler.familyMemberBelongsTo(fixture.user.ID, fixture.member.ID) {
		t.Fatal("deleted family member should not be usable for appointments")
	}
}

func TestCreateFamilyMemberAssignsCurrentUser(t *testing.T) {
	handler, db, fixture := newFamilyMemberFixture(t)
	router := newFamilyMemberTestRouter(handler, fixture.user)

	response := performFamilyRequest(t, router, http.MethodPost, "/family-members", familyMemberRequest{
		Name:     "母亲",
		Relation: "母亲",
		Gender:   "female",
		IDCard:   testIDCard("19680101"),
		Phone:    "13900000001",
	})

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	var created models.FamilyMember
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created member: %v", err)
	}
	if created.UserID != fixture.user.ID || created.Name != "母亲" {
		t.Fatalf("created member should belong to current user, got %#v", created)
	}
	if created.Status != "active" {
		t.Fatalf("created member should be active, got %q", created.Status)
	}
	if created.Age <= 0 || created.IDCard == "" {
		t.Fatalf("created member should have age calculated from id card, got %#v", created)
	}
	var count int64
	if err := db.Model(&models.FamilyMember{}).Where("user_id = ?", fixture.user.ID).Count(&count).Error; err != nil {
		t.Fatalf("count members: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected current user to have 2 members, got %d", count)
	}
	assertFamilyOperationLog(t, db, fixture.user.ID, created.ID, "create", "母亲")
}

func TestUpdateFamilyMemberRejectsOtherUsersMember(t *testing.T) {
	handler, db, fixture := newFamilyMemberFixture(t)
	router := newFamilyMemberTestRouter(handler, fixture.user)

	response := performFamilyRequest(t, router, http.MethodPatch, "/family-members/"+strconv.Itoa(int(fixture.otherMember.ID)), familyMemberRequest{
		Name:     "越权修改",
		Relation: "父亲",
	})

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "family member not found")
	var member models.FamilyMember
	if err := db.First(&member, fixture.otherMember.ID).Error; err != nil {
		t.Fatalf("load other member: %v", err)
	}
	if member.Name != fixture.otherMember.Name {
		t.Fatalf("other user's member was modified: %#v", member)
	}
}

func TestUpdateFamilyMemberUpdatesCurrentUsersMember(t *testing.T) {
	handler, db, fixture := newFamilyMemberFixture(t)
	router := newFamilyMemberTestRouter(handler, fixture.user)

	response := performFamilyRequest(t, router, http.MethodPatch, "/family-members/"+strconv.Itoa(int(fixture.member.ID)), familyMemberRequest{
		Name:     "父亲-更新",
		Relation: "父亲",
		Gender:   "male",
		IDCard:   testIDCard("19620101"),
		Phone:    "13900000002",
	})

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var member models.FamilyMember
	if err := db.First(&member, fixture.member.ID).Error; err != nil {
		t.Fatalf("load updated member: %v", err)
	}
	if member.Name != "父亲-更新" || member.UserID != fixture.user.ID || member.Age <= 0 || member.IDCard == "" {
		t.Fatalf("member was not updated correctly: %#v", member)
	}
	assertFamilyOperationLog(t, db, fixture.user.ID, fixture.member.ID, "update", "父亲-更新")
}

func TestDeleteFamilyMemberRejectsOtherUsersMember(t *testing.T) {
	handler, db, fixture := newFamilyMemberFixture(t)
	router := newFamilyMemberTestRouter(handler, fixture.user)

	response := performFamilyRequest(t, router, http.MethodDelete, "/family-members/"+strconv.Itoa(int(fixture.otherMember.ID)), nil)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "family member not found")
	var count int64
	if err := db.Model(&models.FamilyMember{}).Where("id = ?", fixture.otherMember.ID).Count(&count).Error; err != nil {
		t.Fatalf("count other member: %v", err)
	}
	if count != 1 {
		t.Fatalf("other user's member should remain, got count %d", count)
	}
}

func TestDeleteFamilyMemberArchivesCurrentUsersMember(t *testing.T) {
	handler, db, fixture := newFamilyMemberFixture(t)
	router := newFamilyMemberTestRouter(handler, fixture.user)

	response := performFamilyRequest(t, router, http.MethodDelete, "/family-members/"+strconv.Itoa(int(fixture.member.ID)), nil)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var member models.FamilyMember
	if err := db.First(&member, fixture.member.ID).Error; err != nil {
		t.Fatalf("load archived member: %v", err)
	}
	if member.Status != "deleted" {
		t.Fatalf("current user's member should be archived, got %q", member.Status)
	}
	assertFamilyOperationLog(t, db, fixture.user.ID, fixture.member.ID, "archive", "")
}

type familyMemberFixture struct {
	user        models.User
	otherUser   models.User
	member      models.FamilyMember
	otherMember models.FamilyMember
}

func newFamilyMemberFixture(t *testing.T) (*Handler, *gorm.DB, familyMemberFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.FamilyMember{}, &models.OperationLog{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	fixture := familyMemberFixture{
		user:        models.User{ID: 100, Name: "用户", Phone: "13800000100", Email: "user@example.com", Role: "user", Status: "active", PasswordHash: "hash"},
		otherUser:   models.User{ID: 200, Name: "其他用户", Phone: "13800000200", Email: "other@example.com", Role: "user", Status: "active", PasswordHash: "hash"},
		member:      models.FamilyMember{ID: 10, UserID: 100, Name: "父亲", Relation: "父亲", Gender: "male", Age: 60, Phone: "13900000000", Status: "active"},
		otherMember: models.FamilyMember{ID: 20, UserID: 200, Name: "其他家属", Relation: "母亲", Gender: "female", Age: 59, Phone: "13900000020", Status: "active"},
	}
	for _, row := range []any{&fixture.user, &fixture.otherUser, &fixture.member, &fixture.otherMember} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}
	handler := &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}
	return handler, db, fixture
}

func newFamilyMemberTestRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.GET("/family-members", handler.familyMembers)
	router.POST("/family-members", handler.createFamilyMember)
	router.PATCH("/family-members/:id", handler.updateFamilyMember)
	router.DELETE("/family-members/:id", handler.deleteFamilyMember)
	return router
}

func performFamilyRequest(t *testing.T, router *gin.Engine, method, path string, payload any) *httptest.ResponseRecorder {
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

func assertFamilyOperationLog(t *testing.T, db *gorm.DB, userID, memberID uint, action, detail string) {
	t.Helper()
	var count int64
	if err := db.Model(&models.OperationLog{}).
		Where("user_id = ? AND action = ? AND resource = ? AND resource_id = ? AND detail = ?", userID, action, "family_member", strconv.Itoa(int(memberID)), detail).
		Count(&count).Error; err != nil {
		t.Fatalf("count family operation log: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one %s family operation log with detail %q, got %d", action, detail, count)
	}
}

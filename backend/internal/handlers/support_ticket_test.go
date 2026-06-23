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

func TestCreateSupportTicketAssignsCurrentUserAndAudits(t *testing.T) {
	handler, db, fixture := newSupportTicketFixture(t)
	router := newSupportTicketRouter(handler, fixture.user)

	response := performSupportTicketRequest(t, router, http.MethodPost, "/support-tickets", supportTicketRequest{
		Subject: "  发票咨询  ",
		Content: "  想确认发票什么时候开具  ",
	})

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	var ticket models.SupportTicket
	if err := json.Unmarshal(response.Body.Bytes(), &ticket); err != nil {
		t.Fatalf("decode ticket: %v", err)
	}
	if ticket.UserID != fixture.user.ID || ticket.Subject != "发票咨询" || ticket.Status != "open" {
		t.Fatalf("unexpected support ticket: %#v", ticket)
	}
	assertSupportTicketOperationLog(t, db, fixture.user.ID, ticket.ID, "create", "发票咨询")
}

func TestMySupportTicketsOnlyReturnsOwnTickets(t *testing.T) {
	handler, _, fixture := newSupportTicketFixture(t)
	router := newSupportTicketRouter(handler, fixture.user)

	response := performSupportTicketRequest(t, router, http.MethodGet, "/support-tickets?page=1&pageSize=10", nil)

	payload := decodeSupportTicketPage(t, response)
	if payload.Total != 1 || len(payload.Items) != 1 {
		t.Fatalf("expected one own ticket, got %#v", payload)
	}
	if payload.Items[0].ID != fixture.openTicket.ID {
		t.Fatalf("unexpected own ticket payload: %#v", payload.Items)
	}
}

func TestAdminSupportTicketsFilterAndReply(t *testing.T) {
	handler, db, fixture := newSupportTicketFixture(t)
	router := newSupportTicketRouter(handler, fixture.admin)

	list := performSupportTicketRequest(t, router, http.MethodGet, "/admin/support-tickets?page=1&pageSize=10&status=open&keyword=改期", nil)
	payload := decodeSupportTicketPage(t, list)
	if payload.Total != 1 || payload.Items[0].ID != fixture.openTicket.ID {
		t.Fatalf("admin filter returned wrong tickets: %#v", payload)
	}

	reply := performSupportTicketRequest(t, router, http.MethodPatch, "/admin/support-tickets/"+strconv.Itoa(int(fixture.openTicket.ID))+"/reply", supportTicketReplyRequest{
		Reply:  "可以在预约页提交改期申请。",
		Status: "replied",
	})

	if reply.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", reply.Code, reply.Body.String())
	}
	var ticket models.SupportTicket
	if err := json.Unmarshal(reply.Body.Bytes(), &ticket); err != nil {
		t.Fatalf("decode replied ticket: %v", err)
	}
	if ticket.Status != "replied" || ticket.Reply != "可以在预约页提交改期申请。" {
		t.Fatalf("unexpected replied ticket: %#v", ticket)
	}
	assertSupportTicketNotification(t, db, fixture.user.ID)
	assertSupportTicketOperationLog(t, db, fixture.admin.ID, fixture.openTicket.ID, "reply", "replied")
}

func TestReplySupportTicketRejectsMissingReplyForResolvedStatus(t *testing.T) {
	handler, _, fixture := newSupportTicketFixture(t)
	router := newSupportTicketRouter(handler, fixture.admin)

	response := performSupportTicketRequest(t, router, http.MethodPatch, "/admin/support-tickets/"+strconv.Itoa(int(fixture.openTicket.ID))+"/reply", supportTicketReplyRequest{
		Status: "closed",
	})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "support ticket reply is required")
}

type supportTicketFixture struct {
	admin       models.User
	user        models.User
	otherUser   models.User
	openTicket  models.SupportTicket
	otherTicket models.SupportTicket
}

func newSupportTicketFixture(t *testing.T) (*Handler, *gorm.DB, supportTicketFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.SupportTicket{}, &models.Notification{}, &models.OperationLog{}, &models.SystemSetting{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	fixture := supportTicketFixture{
		admin:       models.User{ID: 1, Name: "管理员", Email: "admin@example.com", Phone: "13800000001", Role: "admin", Status: "active", PasswordHash: "hash"},
		user:        models.User{ID: 2, Name: "用户甲", Email: "user@example.com", Phone: "13800000002", Role: "user", Status: "active", PasswordHash: "hash"},
		otherUser:   models.User{ID: 3, Name: "用户乙", Email: "other@example.com", Phone: "13800000003", Role: "user", Status: "active", PasswordHash: "hash"},
		openTicket:  models.SupportTicket{ID: 10, UserID: 2, Subject: "预约改期", Content: "怎么改期？", Status: "open"},
		otherTicket: models.SupportTicket{ID: 11, UserID: 3, Subject: "发票", Content: "发票问题", Status: "open"},
	}
	inAppSetting := models.SystemSetting{Key: "notification.in_app_enabled", Value: "true", ValueType: "boolean", Group: "notification", Label: "站内信通知", Status: "active"}
	for _, row := range []any{&fixture.admin, &fixture.user, &fixture.otherUser, &fixture.openTicket, &fixture.otherTicket, &inAppSetting} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create fixture row %#v: %v", row, err)
		}
	}
	return &Handler{db: db, redis: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}, db, fixture
}

func newSupportTicketRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.GET("/support-tickets", handler.mySupportTickets)
	router.POST("/support-tickets", handler.createSupportTicket)
	router.GET("/admin/support-tickets", handler.supportTickets)
	router.PATCH("/admin/support-tickets/:id/reply", handler.replySupportTicket)
	return router
}

func performSupportTicketRequest(t *testing.T, router *gin.Engine, method, path string, payload any) *httptest.ResponseRecorder {
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

func decodeSupportTicketPage(t *testing.T, response *httptest.ResponseRecorder) struct {
	Items    []models.SupportTicket `json:"items"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"pageSize"`
} {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Items    []models.SupportTicket `json:"items"`
		Total    int64                  `json:"total"`
		Page     int                    `json:"page"`
		PageSize int                    `json:"pageSize"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode ticket page: %v", err)
	}
	return payload
}

func assertSupportTicketNotification(t *testing.T, db *gorm.DB, userID uint) {
	t.Helper()
	var count int64
	if err := db.Model(&models.Notification{}).Where("user_id = ? AND type = ?", userID, "support_ticket_reply").Count(&count).Error; err != nil {
		t.Fatalf("count support ticket notifications: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one support ticket reply notification, got %d", count)
	}
}

func assertSupportTicketOperationLog(t *testing.T, db *gorm.DB, userID, ticketID uint, action, detail string) {
	t.Helper()
	var count int64
	if err := db.Model(&models.OperationLog{}).
		Where("user_id = ? AND action = ? AND resource = ? AND resource_id = ? AND detail = ?", userID, action, "support_ticket", strconv.Itoa(int(ticketID)), detail).
		Count(&count).Error; err != nil {
		t.Fatalf("count support ticket operation log: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one %s support ticket operation log with detail %q, got %d", action, detail, count)
	}
}

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
	"gorm.io/gorm"
)

func TestUpdateAppointmentInvoiceUpdatesOwnAppointment(t *testing.T) {
	handler, db, fixture := newPaymentStatusFixture(t)
	router := newInvoiceRouter(handler, fixture.user)

	response := performInvoiceRequest(t, router, fixture.bookedAppointment.ID, invoiceRequest{InvoiceTitle: "  东软熙心科技有限公司  ", InvoiceTaxNo: "  TAX123456  "})

	appointment := decodeInvoiceAppointment(t, response)
	if appointment.InvoiceTitle != "东软熙心科技有限公司" || appointment.InvoiceTaxNo != "TAX123456" {
		t.Fatalf("unexpected invoice data: %#v", appointment)
	}
	assertAppointmentInvoice(t, db, fixture.bookedAppointment.ID, "东软熙心科技有限公司", "TAX123456")
}

func TestUpdateAppointmentInvoiceRejectsOtherUsersAppointment(t *testing.T) {
	handler, db, fixture := newPaymentStatusFixture(t)
	router := newInvoiceRouter(handler, fixture.user)

	response := performInvoiceRequest(t, router, fixture.otherAppointment.ID, invoiceRequest{InvoiceTitle: "越权公司", InvoiceTaxNo: "TAX999"})

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
	assertAppointmentInvoice(t, db, fixture.otherAppointment.ID, "", "")
}

func TestUpdateAppointmentInvoiceRejectsCanceledAppointment(t *testing.T) {
	handler, db, fixture := newPaymentStatusFixture(t)
	if err := db.Model(&models.Appointment{}).Where("id = ?", fixture.bookedAppointment.ID).Update("status", "canceled").Error; err != nil {
		t.Fatalf("mark appointment canceled: %v", err)
	}
	router := newInvoiceRouter(handler, fixture.user)

	response := performInvoiceRequest(t, router, fixture.bookedAppointment.ID, invoiceRequest{InvoiceTitle: "取消后公司", InvoiceTaxNo: "TAX000"})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "invoice cannot be updated for canceled appointments")
	assertAppointmentInvoice(t, db, fixture.bookedAppointment.ID, "", "")
}

func TestUpdateAppointmentInvoiceRejectsTooLongFields(t *testing.T) {
	handler, _, fixture := newPaymentStatusFixture(t)
	router := newInvoiceRouter(handler, fixture.user)

	response := performInvoiceRequest(t, router, fixture.bookedAppointment.ID, invoiceRequest{InvoiceTitle: longString(129), InvoiceTaxNo: "TAX123"})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	assertErrorMessage(t, response.Body.Bytes(), "invoice fields are too long")
}

func newInvoiceRouter(handler *Handler, current models.User) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", current)
		c.Next()
	})
	router.PATCH("/appointments/:id/invoice", handler.updateAppointmentInvoice)
	return router
}

func performInvoiceRequest(t *testing.T, router *gin.Engine, appointmentID uint, req invoiceRequest) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	httpReq := httptest.NewRequest(http.MethodPatch, "/appointments/"+strconv.Itoa(int(appointmentID))+"/invoice", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httpReq)
	return rec
}

func decodeInvoiceAppointment(t *testing.T, response *httptest.ResponseRecorder) models.Appointment {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var appointment models.Appointment
	if err := json.Unmarshal(response.Body.Bytes(), &appointment); err != nil {
		t.Fatalf("decode appointment: %v", err)
	}
	return appointment
}

func assertAppointmentInvoice(t *testing.T, db *gorm.DB, appointmentID uint, title, taxNo string) {
	t.Helper()
	var appointment models.Appointment
	if err := db.First(&appointment, appointmentID).Error; err != nil {
		t.Fatalf("load appointment: %v", err)
	}
	if appointment.InvoiceTitle != title || appointment.InvoiceTaxNo != taxNo {
		t.Fatalf("expected invoice %q/%q, got %q/%q", title, taxNo, appointment.InvoiceTitle, appointment.InvoiceTaxNo)
	}
}

func longString(length int) string {
	bytes := make([]byte, length)
	for i := range bytes {
		bytes[i] = 'A'
	}
	return string(bytes)
}

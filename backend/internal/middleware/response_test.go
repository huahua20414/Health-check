package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUnifiedJSONResponseWrapsSuccessfulJSON(t *testing.T) {
	body, headers, status := performResponseMiddlewareRequest(t, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if headers.Get("X-Request-Id") != "req-test" {
		t.Fatalf("expected request id header to be preserved, got %q", headers.Get("X-Request-Id"))
	}
	assertPayload(t, body, 0, "ok", map[string]any{"status": "ok"})
}

func TestUnifiedJSONResponseWrapsErrorMessage(t *testing.T) {
	body, _, status := performResponseMiddlewareRequest(t, func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
	})

	if status != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", status)
	}
	assertPayload(t, body, http.StatusBadRequest, "invalid input", nil)
}

func TestUnifiedJSONResponseKeepsAlreadyWrappedPayload(t *testing.T) {
	expected := gin.H{"code": 0, "message": "ok", "data": gin.H{"ready": true}, "requestId": "custom-request"}
	body, _, status := performResponseMiddlewareRequest(t, func(c *gin.Context) {
		c.JSON(http.StatusOK, expected)
	})

	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got["requestId"] != "custom-request" {
		t.Fatalf("expected original request id, got %#v", got["requestId"])
	}
	if got["code"].(float64) != 0 || got["message"] != "ok" {
		t.Fatalf("unexpected payload: %#v", got)
	}
}

func TestUnifiedJSONResponseSkipsNonJSONResponses(t *testing.T) {
	body, headers, status := performResponseMiddlewareRequest(t, func(c *gin.Context) {
		c.String(http.StatusOK, "plain text")
	})

	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if string(body) != "plain text" {
		t.Fatalf("expected plain response, got %q", string(body))
	}
	if headers.Get("Content-Type") == "application/json; charset=utf-8" {
		t.Fatalf("plain text response should not be forced to JSON")
	}
}

func TestRecoveryReturnsUnifiedInternalServerError(t *testing.T) {
	body, _, status := performResponseMiddlewareRequest(t, func(c *gin.Context) {
		panic("boom")
	})

	if status != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", status)
	}
	assertPayload(t, body, http.StatusInternalServerError, "internal server error", nil)
}

func performResponseMiddlewareRequest(t *testing.T, handler gin.HandlerFunc) ([]byte, http.Header, int) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID(), UnifiedJSONResponse(), Recovery())
	router.GET("/test", handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-Id", "req-test")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec.Body.Bytes(), rec.Header(), rec.Code
}

func assertPayload(t *testing.T, body []byte, code int, message string, data any) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode payload %q: %v", string(body), err)
	}
	if int(payload["code"].(float64)) != code {
		t.Fatalf("expected code %d, got %#v", code, payload["code"])
	}
	if payload["message"] != message {
		t.Fatalf("expected message %q, got %#v", message, payload["message"])
	}
	if payload["requestId"] != "req-test" {
		t.Fatalf("expected request id req-test, got %#v", payload["requestId"])
	}
	if data == nil {
		if payload["data"] != nil {
			t.Fatalf("expected nil data, got %#v", payload["data"])
		}
		return
	}
	expected, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal expected data: %v", err)
	}
	actual, err := json.Marshal(payload["data"])
	if err != nil {
		t.Fatalf("marshal actual data: %v", err)
	}
	if string(actual) != string(expected) {
		t.Fatalf("expected data %s, got %s", expected, actual)
	}
}

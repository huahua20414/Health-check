package middleware

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w responseBodyWriter) Write(data []byte) (int, error) {
	if w.body != nil {
		return w.body.Write(data)
	}
	return w.ResponseWriter.Write(data)
}

func (w responseBodyWriter) WriteString(data string) (int, error) {
	if w.body != nil {
		return w.body.WriteString(data)
	}
	return w.ResponseWriter.WriteString(data)
}

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-Id")
		if strings.TrimSpace(requestID) == "" {
			requestID = newRequestID()
		}
		c.Set("requestID", requestID)
		c.Header("X-Request-Id", requestID)
		c.Next()
	}
}

func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		requestID := requestID(c)
		c.Header("X-Request-Id", requestID)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"code":      http.StatusInternalServerError,
			"message":   "internal server error",
			"data":      nil,
			"requestId": requestID,
		})
	})
}

func UnifiedJSONResponse() gin.HandlerFunc {
	return func(c *gin.Context) {
		buffer := &bytes.Buffer{}
		writer := &responseBodyWriter{ResponseWriter: c.Writer, body: buffer}
		c.Writer = writer
		c.Next()

		status := c.Writer.Status()
		contentType := c.Writer.Header().Get("Content-Type")
		if status == http.StatusNoContent || !strings.Contains(contentType, "application/json") {
			_, _ = writer.ResponseWriter.Write(buffer.Bytes())
			return
		}

		var raw any
		if buffer.Len() > 0 {
			if err := json.Unmarshal(buffer.Bytes(), &raw); err != nil {
				_, _ = writer.ResponseWriter.Write(buffer.Bytes())
				return
			}
		}

		payload := normalizePayload(status, requestID(c), raw)
		writer.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(writer.ResponseWriter).Encode(payload)
	}
}

func normalizePayload(status int, requestID string, raw any) gin.H {
	if body, ok := raw.(map[string]any); ok {
		if _, ok := body["code"]; ok {
			if _, ok := body["message"]; ok {
				if _, ok := body["requestId"]; ok {
					return body
				}
			}
		}
	}

	if status >= 400 {
		message := http.StatusText(status)
		if body, ok := raw.(map[string]any); ok {
			if value, ok := body["error"].(string); ok && value != "" {
				message = value
			} else if value, ok := body["message"].(string); ok && value != "" {
				message = value
			}
		}
		return gin.H{"code": status, "message": message, "data": nil, "requestId": requestID}
	}

	return gin.H{"code": 0, "message": "ok", "data": raw, "requestId": requestID}
}

func requestID(c *gin.Context) string {
	if value, ok := c.Get("requestID"); ok {
		if text, ok := value.(string); ok && text != "" {
			return text
		}
	}
	return newRequestID()
}

func newRequestID() string {
	var data [8]byte
	if _, err := rand.Read(data[:]); err != nil {
		return fmt.Sprintf("req-%d", randIntFallback())
	}
	return hex.EncodeToString(data[:])
}

func randIntFallback() int64 {
	return time.Now().UnixNano()
}

package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type visitor struct {
	windowStart time.Time
	count       int
	lastSeen    time.Time
}

func IPRateLimit(limit int, window time.Duration) gin.HandlerFunc {
	var mu sync.Mutex
	visitors := map[string]*visitor{}

	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, item := range visitors {
				if time.Since(item.lastSeen) > 3*time.Minute {
					delete(visitors, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		mu.Lock()
		item, ok := visitors[ip]
		if !ok || now.Sub(item.windowStart) > window {
			item = &visitor{windowStart: now}
			visitors[ip] = item
		}
		item.count++
		item.lastSeen = now
		allowed := item.count <= limit
		mu.Unlock()

		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}
		c.Next()
	}
}

package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/platform/utils"
)

const rateLimitSweepSize = 10000

type rateWindow struct {
	count   int
	resetAt time.Time
}

// RateLimit allows at most limit requests per client IP in each window (fixed window, kept in
// memory, so it applies per process). Meant for endpoints that have no login.
func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	var mu sync.Mutex
	clients := make(map[string]*rateWindow)

	return func(c *gin.Context) {
		now := time.Now()
		ip := c.ClientIP()

		mu.Lock()
		if len(clients) > rateLimitSweepSize {
			for k, w := range clients {
				if now.After(w.resetAt) {
					delete(clients, k)
				}
			}
		}
		w, ok := clients[ip]
		if !ok || now.After(w.resetAt) {
			w = &rateWindow{resetAt: now.Add(window)}
			clients[ip] = w
		}
		w.count++
		over, retryAfter := w.count > limit, time.Until(w.resetAt)
		mu.Unlock()

		if over {
			c.Header("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
			utils.AbortWithError(c, http.StatusTooManyRequests, "too many requests, try again later")
			return
		}
		c.Next()
	}
}

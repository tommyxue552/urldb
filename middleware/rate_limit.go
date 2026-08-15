package middleware

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimitByIP provides a small, process-local fixed-window limit for public
// endpoints. It intentionally does not trust a caller-supplied source/channel
// parameter: requests are grouped by Gin's configured client IP and endpoint
// scope. Deployments with multiple API replicas should enforce the same policy
// at the edge or replace this with a shared-store limiter.
func RateLimitByIP(scope string, maxRequests int, window time.Duration) gin.HandlerFunc {
	if maxRequests < 1 {
		maxRequests = 1
	}
	if window <= 0 {
		window = time.Minute
	}

	type entry struct {
		count     int
		resetTime time.Time
	}

	var mu sync.Mutex
	entries := make(map[string]entry)

	return func(c *gin.Context) {
		now := time.Now()
		key := scope + ":" + c.ClientIP()

		mu.Lock()
		current := entries[key]
		if current.resetTime.IsZero() || !now.Before(current.resetTime) {
			current = entry{resetTime: now.Add(window)}
		}
		if current.count >= maxRequests {
			retryAfter := int(math.Ceil(time.Until(current.resetTime).Seconds()))
			if retryAfter < 1 {
				retryAfter = 1
			}
			mu.Unlock()
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"message": "请求过于频繁，请稍后再试",
				"code":    http.StatusTooManyRequests,
			})
			return
		}
		current.count++
		entries[key] = current
		mu.Unlock()
		c.Next()
	}
}

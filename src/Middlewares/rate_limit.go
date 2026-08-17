package Middlewares

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimitEntry struct {
	count     int
	windowEnd time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	entries map[string]*rateLimitEntry
	max     int
	window  time.Duration
}

func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		entries: make(map[string]*rateLimitEntry),
		max:     max,
		window:  window,
	}
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		rl.mu.Lock()
		entry, ok := rl.entries[ip]
		now := time.Now()
		if !ok || now.After(entry.windowEnd) {
			rl.entries[ip] = &rateLimitEntry{count: 1, windowEnd: now.Add(rl.window)}
			rl.mu.Unlock()
			c.Next()
			return
		}
		entry.count++
		if entry.count > rl.max {
			rl.mu.Unlock()
			_ = c.Error(&AppError{Code: http.StatusTooManyRequests, Message: "Too many requests, please try again later"})
			c.Abort()
			return
		}
		rl.mu.Unlock()
		c.Next()
	}
}

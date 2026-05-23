package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	rdb    *redis.Client
	limit  int
	window time.Duration
}

func NewRateLimiter(rdb *redis.Client, limit int, window time.Duration) *RateLimiter {
	if window <= 0 {
		window = time.Second
	}
	return &RateLimiter{rdb: rdb, limit: limit, window: window}
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	if rl == nil || rl.limit <= 0 || rl.rdb == nil {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		key := extractKey(c)
		if key == "" {
			key = c.ClientIP()
		}
		allowed, err := rl.allow(c.Request.Context(), key)
		if err != nil {
			abortJSON(c, http.StatusServiceUnavailable, "rate limiter unavailable")
			return
		}
		if !allowed {
			abortJSON(c, http.StatusTooManyRequests, "ingest rate limit exceeded")
			return
		}
		c.Next()
	}
}

func (rl *RateLimiter) allow(ctx context.Context, key string) (bool, error) {
	rkey := "observa:ratelimit:ingest:" + key
	n, err := rl.rdb.Incr(ctx, rkey).Result()
	if err != nil {
		return false, err
	}
	if n == 1 {
		if err := rl.rdb.Expire(ctx, rkey, rl.window).Err(); err != nil {
			return false, err
		}
	}
	return n <= int64(rl.limit), nil
}

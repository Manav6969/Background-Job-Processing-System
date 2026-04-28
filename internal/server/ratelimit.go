package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func RateLimitRedis(rdb *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use user_id if authenticated, otherwise fall back to IP
		key := c.ClientIP()
		if userID, exists := c.Get("user_id"); exists {
			key = fmt.Sprintf("user:%v", userID)
		}
		redisKey := fmt.Sprintf("ratelimit:%s", key)

		ctx := context.Background()
		now := time.Now().UnixMilli()
		windowStart := now - window.Milliseconds()

		pipe := rdb.Pipeline()

		// Remove expired entries
		pipe.ZRemRangeByScore(ctx, redisKey, "0", fmt.Sprintf("%d", windowStart))

		// Count current window
		countCmd := pipe.ZCard(ctx, redisKey)

		// Add current request
		pipe.ZAdd(ctx, redisKey, redis.Z{Score: float64(now), Member: now})

		// Set expiry on key
		pipe.Expire(ctx, redisKey, window)

		_, err := pipe.Exec(ctx)
		if err != nil {
			// On Redis error, allow the request through
			c.Next()
			return
		}

		count := countCmd.Val()
		if count >= int64(limit) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": window.Seconds(),
			})
			return
		}

		c.Next()
	}
}

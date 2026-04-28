package server

import (
	"context"
	"net/http"
	"time"

	"github.com/Manav6969/Background-Job-Processing-System/internal/config"
	"github.com/Manav6969/Background-Job-Processing-System/internal/db"
	"github.com/Manav6969/Background-Job-Processing-System/internal/job"
	"github.com/Manav6969/Background-Job-Processing-System/internal/user"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func Run(cfg *config.Config) {
	r := gin.Default()

	// Redis client for rate limiting
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})

	// Rate limit: 100 requests per minute per user/IP
	r.Use(RateLimitRedis(rdb, cfg.RateLimit, 1*time.Minute))

	// Health check endpoints
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "alive"})
	})

	r.GET("/readyz", func(c *gin.Context) {
		// Check DB
		if err := db.Pool.Ping(context.Background()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"db":     "down",
			})
			return
		}
		// Check Redis
		if err := rdb.Ping(context.Background()).Err(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"redis":  "down",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
			"db":     "up",
			"redis":  "up",
		})
	})

	r.GET("/metrics", prometheusHandler())

	// Public routes
	r.POST("/register", user.Register)
	r.POST("/login", user.Login(cfg.JWTSecret))

	// Authenticated routes
	auth := r.Group("/")
	auth.Use(JWT(cfg.JWTSecret))
	{
		auth.POST("/jobs", job.Create)
		auth.GET("/jobs/:id", job.Get)
	}

	// Admin routes
	admin := r.Group("/admin")
	admin.Use(JWT(cfg.JWTSecret))
	// TODO: Add AdminRole middleware check
	{
		admin.POST("/jobs/:id/replay", job.Replay)
	}

	r.Run(cfg.Port)
}

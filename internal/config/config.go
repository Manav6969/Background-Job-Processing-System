package config

import (
	"os"
	"strconv"
)

type Config struct {
	JWTSecret            string
	Port                 string
	RateLimit            int
	DatabaseURL          string
	RedisAddr            string
	MigrationsPath       string
	WorkerConcurrency    int
	ShutdownGracePeriod  int
}

func Load() *Config {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "default_secret_change_me"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = ":6969"
	}

	rateLimit := 5
	if rl := os.Getenv("RATE_LIMIT"); rl != "" {
		if val, err := strconv.Atoi(rl); err == nil {
			rateLimit = val
		}
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/backgroundjobprocessingsystem?sslmode=disable"
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "./migrations"
	}

	workerConcurrency := 10
	if wc := os.Getenv("WORKER_CONCURRENCY"); wc != "" {
		if val, err := strconv.Atoi(wc); err == nil {
			workerConcurrency = val
		}
	}

	shutdownGracePeriod := 30
	if sgp := os.Getenv("SHUTDOWN_GRACE_PERIOD"); sgp != "" {
		if val, err := strconv.Atoi(sgp); err == nil {
			shutdownGracePeriod = val
		}
	}

	return &Config{
		JWTSecret:            jwtSecret,
		Port:                 port,
		RateLimit:            rateLimit,
		DatabaseURL:          databaseURL,
		RedisAddr:            redisAddr,
		MigrationsPath:       migrationsPath,
		WorkerConcurrency:    workerConcurrency,
		ShutdownGracePeriod:  shutdownGracePeriod,
	}
}

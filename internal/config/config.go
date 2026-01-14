package config

import (
	"os"
)

type Config struct {
	JWTSecret string
	Port      string
	RateLimit int
}

func Load() *Config {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		panic("JWT_SECRET is not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = ":6969"
	}

	return &Config{
		JWTSecret: jwtSecret,
		Port:      port,
		RateLimit: 5,
	}
}

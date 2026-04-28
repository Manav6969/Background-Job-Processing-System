package main

import (
	"github.com/Manav6969/Background-Job-Processing-System/internal/config"
	"github.com/Manav6969/Background-Job-Processing-System/internal/db"
	"github.com/Manav6969/Background-Job-Processing-System/internal/job"
	"github.com/Manav6969/Background-Job-Processing-System/internal/logger"
	"github.com/Manav6969/Background-Job-Processing-System/internal/server"
)

func main() {
	logger.Init()
	log := logger.Log

	cfg := config.Load()

	// Connect to Database
	err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to DB")
	}
	log.Info().Msg("Connected to database")

	// Run Migrations
	err = db.RunMigrations(cfg.DatabaseURL, cfg.MigrationsPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to run migrations")
	}
	log.Info().Msg("Migrations complete")

	// Initialize Queue
	job.InitQueue(cfg.RedisAddr)
	log.Info().Str("redis_addr", cfg.RedisAddr).Msg("Queue initialized")

	// Run Server
	log.Info().Str("port", cfg.Port).Msg("Starting API server")
	server.Run(cfg)
}

package main

import (
	"log"
	"github.com/Manav6969/Background-Job-Processing-System/internal/config"
	"github.com/Manav6969/Background-Job-Processing-System/internal/db"
	"github.com/Manav6969/Background-Job-Processing-System/internal/server"
	"github.com/Manav6969/Background-Job-Processing-System/internal/job"
)

func main() {
	cfg := config.Load()

	// Connect to Database
	err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	// Run Migrations
	err = db.RunMigrations(cfg.DatabaseURL, cfg.MigrationsPath)
	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize Queue
	job.InitQueue(cfg.RedisAddr)

	// Run Server
	server.Run(cfg)
}

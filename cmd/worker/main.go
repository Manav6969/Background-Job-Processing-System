package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/Manav6969/Background-Job-Processing-System/internal/config"
	"github.com/Manav6969/Background-Job-Processing-System/internal/db"
	"github.com/Manav6969/Background-Job-Processing-System/internal/logger"
	"github.com/Manav6969/Background-Job-Processing-System/internal/queue"
	"github.com/Manav6969/Background-Job-Processing-System/internal/worker"
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

	q := queue.NewRedisQueue(cfg.RedisAddr, "jobs")

	log.Info().Int("concurrency", cfg.WorkerConcurrency).Msg("Worker started")

	pool := worker.NewPool(cfg.WorkerConcurrency, q)
	pool.Start()

	// Graceful shutdown setup
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	sig := <-stopChan
	log.Info().Str("signal", sig.String()).Msg("Shutting down gracefully...")

	pool.Stop(cfg.ShutdownGracePeriod)
}

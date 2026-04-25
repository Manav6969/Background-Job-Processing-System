package main

import (
	"context"
	"encoding/json"
	"log"
	"github.com/Manav6969/Background-Job-Processing-System/internal/config"
	"github.com/Manav6969/Background-Job-Processing-System/internal/db"
	"github.com/Manav6969/Background-Job-Processing-System/internal/queue"
)

type Job struct {
	ID      int         `json:"id"`
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func main() {
	cfg := config.Load()

	err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	q := queue.NewRedisQueue(cfg.RedisAddr, "jobs")

	log.Println("Worker started, waiting for jobs...")

	for {
		msg, err := q.Pop()
		if err != nil {
			continue
		}

		var job Job
		_ = json.Unmarshal([]byte(msg), &job)

		_, _ = db.Pool.Exec(
			context.Background(),
			"UPDATE jobs SET status=$1, started_at=NOW() WHERE id=$2",
			"running",
			job.ID,
		)

		log.Printf("Processing job %d: %s", job.ID, job.Type)

		// Simulate processing
		_, _ = db.Pool.Exec(
			context.Background(),
			"UPDATE jobs SET status=$1, finished_at=NOW() WHERE id=$2",
			"completed",
			job.ID,
		)

		log.Printf("Job %d completed", job.ID)
	}
}

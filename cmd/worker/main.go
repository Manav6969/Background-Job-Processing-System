package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Manav6969/Background-Job-Processing-System/internal/db"
	"github.com/Manav6969/Background-Job-Processing-System/internal/queue"
)

type Job struct {
	ID      int         `json:"id"`
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func main() {
	q := queue.NewRedisQueue("localhost:6379", "jobs")

	fmt.Println("Worker started, waiting for jobs...")

	for {
		msg, err := q.Pop()
		if err != nil {
			continue
		}

		var job Job
		_ = json.Unmarshal([]byte(msg), &job)

		db.Conn.Exec(
			context.Background(),
			"UPDATE jobs SET status=$1 WHERE id=$2",
			"running",
			job.ID,
		)

		db.Conn.Exec(
			context.Background(),
			"UPDATE jobs SET status=$1 WHERE id=$2",
			"completed",
			job.ID,
		)

		fmt.Println("Processing job:", job.Type, job.Payload)

	}
}

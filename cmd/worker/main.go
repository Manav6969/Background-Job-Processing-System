package main

import (
	"encoding/json"
	"fmt"

	"github.com/Manav6969/Background-Job-Processing-System/internal/queue"
)

type Job struct {
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

		fmt.Println("Processing job:", job.Type, job.Payload)

	}
}

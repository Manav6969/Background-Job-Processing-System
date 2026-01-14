package job

import (
	"context"
	"encoding/json"
	"github.com/Manav6969/Background-Job-Processing-System/internal/db"
	"github.com/Manav6969/Background-Job-Processing-System/internal/queue"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Job struct {
	ID      int         `json:"id"`
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

var q = queue.NewRedisQueue("localhost:6379", "jobs")

func Create(c *gin.Context) {
	var job Job
	if err := c.BindJSON(&job); err != nil {
		c.JSON(400, nil)
		return
	}

	var jobID int

	err := db.Conn.QueryRow(
		context.Background(),
		"INSERT INTO jobs(type, payload, status) VALUES($1, $2, $3) RETURNING id",
		job.Type,
		job.Payload,
		"pending",
	).Scan(&jobID)

	if err != nil {
		c.JSON(500, gin.H{"error": "db error"})
		return
	}

	job.ID = jobID

	data, _ := json.Marshal(job)
	_ = q.Push(string(data))

	c.JSON(http.StatusAccepted, gin.H{
		"status": "queued",
	})
}

func Get(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "pending"})
}

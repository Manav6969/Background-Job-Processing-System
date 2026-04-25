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

var q *queue.RedisQueue

func InitQueue(addr string) {
	q = queue.NewRedisQueue(addr, "jobs")
}

func Create(c *gin.Context) {
	var job Job
	if err := c.BindJSON(&job); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var jobID int

	err := db.Pool.QueryRow(
		context.Background(),
		"INSERT INTO jobs(type, payload, status) VALUES($1, $2, $3) RETURNING id",
		job.Type,
		job.Payload,
		"pending",
	).Scan(&jobID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	job.ID = jobID

	data, _ := json.Marshal(job)
	_ = q.Push(string(data))

	c.JSON(http.StatusAccepted, gin.H{
		"id":     jobID,
		"status": "queued",
	})
}

func Get(c *gin.Context) {
	id := c.Param("id")
	var status string
	err := db.Pool.QueryRow(context.Background(), "SELECT status FROM jobs WHERE id=$1", id).Scan(&status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": status})
}

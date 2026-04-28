package job

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Manav6969/Background-Job-Processing-System/internal/db"
	"github.com/Manav6969/Background-Job-Processing-System/internal/metrics"
	"github.com/Manav6969/Background-Job-Processing-System/internal/queue"
	"github.com/gin-gonic/gin"
)

const MaxPayloadBytes = 1 * 1024 * 1024 // 1MB

type Job struct {
	ID             int         `json:"id"`
	Type           string      `json:"type"`
	Payload        interface{} `json:"payload"`
	IdempotencyKey string      `json:"idempotency_key,omitempty"`
	Priority       string      `json:"priority,omitempty"` // high, default, low
}

var q *queue.RedisQueue

func InitQueue(addr string) {
	q = queue.NewRedisQueue(addr, "jobs")
}

func Create(c *gin.Context) {
	// Payload size validation
	if c.Request.ContentLength > MaxPayloadBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "payload too large"})
		return
	}

	var job Job
	if err := c.BindJSON(&job); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if job.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job type is required"})
		return
	}

	// Get user_id from JWT context (set by JWT middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		userID = nil
	}

	// Idempotency check — if key provided, check for existing job
	if job.IdempotencyKey != "" {
		var existingID int
		var existingStatus string
		err := db.Pool.QueryRow(
			c.Request.Context(),
			"SELECT id, status FROM jobs WHERE idempotency_key=$1",
			job.IdempotencyKey,
		).Scan(&existingID, &existingStatus)
		if err == nil {
			// Job already exists with this key
			c.JSON(http.StatusOK, gin.H{
				"id":      existingID,
				"status":  existingStatus,
				"message": "duplicate job, returning existing",
			})
			return
		}
	}

	var jobID int
	err := db.Pool.QueryRow(
		c.Request.Context(),
		`INSERT INTO jobs(user_id, type, payload, status, idempotency_key) 
		 VALUES($1, $2, $3, 'pending', $4) RETURNING id`,
		userID,
		job.Type,
		job.Payload,
		nilIfEmpty(job.IdempotencyKey),
	).Scan(&jobID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	job.ID = jobID
	data, _ := json.Marshal(job)
	
	priority := job.Priority
	if priority == "" {
		priority = "default"
	}
	_ = q.PushWithPriority(c.Request.Context(), string(data), priority)

	metrics.JobsEnqueuedTotal.WithLabelValues(priority).Inc()

	c.JSON(http.StatusAccepted, gin.H{
		"id":     jobID,
		"status": "queued",
	})
}

func Get(c *gin.Context) {
	id := c.Param("id")

	var status string
	var jobType string
	var retryCount int
	var errorMsg *string

	err := db.Pool.QueryRow(
		c.Request.Context(),
		"SELECT status, type, retry_count, error_message FROM jobs WHERE id=$1",
		id,
	).Scan(&status, &jobType, &retryCount, &errorMsg)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	response := gin.H{
		"id":          id,
		"status":      status,
		"type":        jobType,
		"retry_count": retryCount,
	}
	if errorMsg != nil {
		response["error_message"] = *errorMsg
	}
	c.JSON(http.StatusOK, response)
}

func Replay(c *gin.Context) {
	id := c.Param("id")

	var status string
	var jobType string
	var payloadBytes []byte
	var idempotencyKey *string

	err := db.Pool.QueryRow(
		c.Request.Context(),
		"SELECT status, type, payload, idempotency_key FROM jobs WHERE id=$1",
		id,
	).Scan(&status, &jobType, &payloadBytes, &idempotencyKey)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	if status != "dead" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only dead jobs can be replayed"})
		return
	}

	_, err = db.Pool.Exec(
		c.Request.Context(),
		"UPDATE jobs SET status='pending', retry_count=0, error_message=NULL WHERE id=$1",
		id,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update job status"})
		return
	}

	var payload interface{}
	_ = json.Unmarshal(payloadBytes, &payload)

	jobID, _ := strconv.Atoi(id)
	job := Job{
		ID:      jobID,
		Type:    jobType,
		Payload: payload,
	}
	if idempotencyKey != nil {
		job.IdempotencyKey = *idempotencyKey
	}

	data, _ := json.Marshal(job)
	_ = q.PushWithPriority(c.Request.Context(), string(data), "high") // Replay with high priority

	metrics.JobsEnqueuedTotal.WithLabelValues("high").Inc()

	c.JSON(http.StatusOK, gin.H{"message": "job replayed successfully"})
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

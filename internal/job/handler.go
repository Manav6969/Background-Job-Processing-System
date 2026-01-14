package job

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/Manav6969/Background-Job-Processing-System/internal/queue"
)

type Job struct {
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

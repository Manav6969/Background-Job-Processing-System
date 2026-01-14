package job

import (
    "github.com/gin-gonic/gin"
    "net/http"
)

func Create(c *gin.Context) {
    c.JSON(http.StatusAccepted, gin.H{"status": "job queued"})
}

func Get(c *gin.Context) {
    id := c.Param("id")
    c.JSON(http.StatusOK, gin.H{"id": id, "status": "pending"})
}

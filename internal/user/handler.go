package user

import (
    "github.com/gin-gonic/gin"
    "net/http"
    "github.com/Manav6969/Background-Job-Processing-System/internal/auth"
)

type LoginReq struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

func Register(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"message": "registered"})
}

func Login(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req LoginReq
        if err := c.BindJSON(&req); err != nil {
            c.JSON(400, nil)
            return
        }

        token, _ := auth.Generate(secret, req.Username)
        c.JSON(200, gin.H{"token": token})
    }
}

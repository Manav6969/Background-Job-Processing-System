package server

import (
    "github.com/gin-gonic/gin"
    "github.com/Manav6969/Background-Job-Processing-System/internal/config"
    "github.com/Manav6969/Background-Job-Processing-System/internal/user"
    "github.com/Manav6969/Background-Job-Processing-System/internal/job"
)

func Run(cfg *config.Config) {
    r := gin.Default()
    r.Use(RateLimit(cfg.RateLimit))

    r.POST("/register", user.Register)
    r.POST("/login", user.Login(cfg.JWTSecret))

    auth := r.Group("/")
    auth.Use(JWT(cfg.JWTSecret))
    {
        auth.POST("/jobs", job.Create)
        auth.GET("/jobs/:id", job.Get)
    }

    r.Run(cfg.Port)
}

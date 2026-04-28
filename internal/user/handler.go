package user

import (
	"context"
	"net/http"

	"github.com/Manav6969/Background-Job-Processing-System/internal/auth"
	"github.com/Manav6969/Background-Job-Processing-System/internal/db"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type RegisterReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Register(c *gin.Context) {
	var req RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and password are required"})
		return
	}

	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 6 characters"})
		return
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process password"})
		return
	}

	var userID int
	err = db.Pool.QueryRow(
		context.Background(),
		"INSERT INTO users(username, password_hash) VALUES($1, $2) RETURNING id",
		req.Username,
		string(hash),
	).Scan(&userID)

	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "username already taken"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":       userID,
		"username": req.Username,
		"message":  "user registered successfully",
	})
}

func Login(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username and password are required"})
			return
		}

		var userID int
		var passwordHash string
		err := db.Pool.QueryRow(
			context.Background(),
			"SELECT id, password_hash FROM users WHERE username=$1",
			req.Username,
		).Scan(&userID, &passwordHash)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		// Verify password
		if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		token, err := auth.Generate(secret, userID, req.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token":    token,
			"user_id":  userID,
			"username": req.Username,
		})
	}
}

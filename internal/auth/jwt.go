package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func Generate(secret string, userID int, username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	})
	return token.SignedString([]byte(secret))
}

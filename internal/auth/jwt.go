package auth

import (
    "github.com/golang-jwt/jwt/v5"
    "time"
)

func Generate(secret, userID string) (string, error) {
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "sub": userID,
        "exp": time.Now().Add(24 * time.Hour).Unix(),
    })
    return token.SignedString([]byte(secret))
}

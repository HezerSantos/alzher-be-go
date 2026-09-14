package jwt

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func GenerateUserJWT(userId uuid.UUID, email string, exp int) (string, error) {
	AUTH_SECRET := []byte(os.Getenv("SECURE_AUTH_SECRET"))
	claims := jwt.MapClaims{
		"sub":   userId,
		"email": email,
		"exp":   time.Now().Add(time.Duration(exp) * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(AUTH_SECRET)
}

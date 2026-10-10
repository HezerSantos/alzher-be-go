package jwt

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func GenerateUserJWT(userId uuid.UUID, email string, exp int, unit time.Duration, secretKey string) (*JwtWrapper, error) {
	AUTH_SECRET := []byte(os.Getenv(secretKey))
	expAt := time.Now().Add(time.Duration(exp) * unit).Unix()
	iat := time.Now().Unix()

	jwtWrapper := JwtWrapper{Iat: iat, Exp: expAt}

	claims := jwt.MapClaims{
		"sub":   userId,
		"email": email,
		"exp":   expAt,
		"iat":   iat,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(AUTH_SECRET)

	if err != nil {
		return nil, err
	}

	jwtWrapper.Token = signedToken

	return &jwtWrapper, nil
}

type JwtWrapper struct {
	Token string
	Iat   int64
	Exp   int64
}

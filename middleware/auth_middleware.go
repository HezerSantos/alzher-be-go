package middleware

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/HezerSantos/alzher-api/services/railway/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type UserContextKeyType string

const UserContextKey UserContextKeyType = "userID"

func GetUserDetail(ginCtx *gin.Context, rawToken string, SECRET []byte) (*models.User, error) {

	token, err := jwt.ParseWithClaims(
		rawToken,
		&jwt.MapClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(SECRET), nil
		},
	)

	if err != nil || !token.Valid {
		ginCtx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		fmt.Println("Auth Secret Is Not Valid Or Token Is Expired")
		return nil, err
	}

	claims, ok := token.Claims.(*jwt.MapClaims)

	if !ok {
		ginCtx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		fmt.Println("Auth Cookie Claims Is Not Map Claims")
		return nil, fmt.Errorf("Auth Cookie Claims Is Not Map Claims")
	}
	sub, err := (*claims).GetSubject()

	if err != nil || sub == "" {
		ginCtx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		fmt.Println("Error Fetching Auth Cookie Sub")
		return nil, fmt.Errorf("missing or invalid subject")
	}

	email, ok := (*claims)["email"]

	if !ok {
		ginCtx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		fmt.Println("Error Fetching Auth Cookie Email")
		return nil, fmt.Errorf("Error Fetching Auth Cookie Email")
	}

	assertedEmail, ok := email.(string)

	if !ok {
		ginCtx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		fmt.Println("Auth Cookie Email Is Not String")
		return nil, fmt.Errorf("Auth Cookie Email Is Not String")
	}

	parsedUserId, err := uuid.Parse(sub)

	if err != nil {
		ginCtx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		fmt.Println("Auth Cookie Sub Is Not UUID")
		return nil, err
	}
	user := models.User{
		ID:    parsedUserId,
		Email: assertedEmail,
	}

	return &user, nil
}

func AuthMiddleware() gin.HandlerFunc {
	var SECURE_AUTH_SECRET = []byte(os.Getenv("SECURE_AUTH_SECRET"))
	return func(ginCtx *gin.Context) {

		rawToken := ginCtx.Request.Header.Get("Authorization")

		if rawToken == "" {
			ginCtx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		parts := strings.Split(rawToken, " ")

		if len(parts) != 2 || parts[0] != "Bearer" {
			ginCtx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		token := parts[1]

		user, err := GetUserDetail(ginCtx, token, SECURE_AUTH_SECRET)

		if err != nil {
			return
		}

		newCtx := context.WithValue(ginCtx.Request.Context(), UserContextKey, *user)

		ginCtx.Request = ginCtx.Request.WithContext(newCtx)

		ginCtx.Next()
	}
}

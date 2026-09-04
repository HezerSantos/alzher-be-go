package middleware

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/HezerSantos/alzher-api/services/railway/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var SECURE_AUTH_SECRET = []byte(os.Getenv("SECURE_AUTH_SECRET"))

type UserContextKeyType string

const UserContextKey UserContextKeyType = "userID"

func AuthMiddleware() gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		cookie, err := ginCtx.Request.Cookie("__Secure-secure-auth.access")

		if err != nil {
			ginCtx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			fmt.Println("Auth Cookie Is Missing")
			return
		}

		token, err := jwt.ParseWithClaims(
			cookie.Value,
			&jwt.MapClaims{},
			func(token *jwt.Token) (interface{}, error) {
				if token.Method != jwt.SigningMethodHS256 {
					return nil, fmt.Errorf("unexpected signing method")
				}
				return []byte(SECURE_AUTH_SECRET), nil
			},
		)

		if err != nil {
			ginCtx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			fmt.Println("Auth Secret Is Not Valid Or Token Is Expired")
			return
		}

		claims, ok := token.Claims.(*jwt.MapClaims)

		if !ok {
			ginCtx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			fmt.Println("Auth Cookie Claims Is Not Map Claims")
			return
		}
		sub, err := (*claims).GetSubject()

		if err != nil || sub == "" {
			ginCtx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			fmt.Println("Error Fetching Auth Cookie Sub")
			return
		}

		email, ok := (*claims)["email"]

		if !ok {
			ginCtx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			fmt.Println("Error Fetching Auth Cookie Email")
			return
		}

		assertedEmail, ok := email.(string)

		if !ok {
			ginCtx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			fmt.Println("Auth Cookie Email Is Not String")
			return
		}

		parsedUserId, err := uuid.Parse(sub)

		if err != nil {
			ginCtx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			fmt.Println("Auth Cookie Sub Is Not UUID")
			return
		}
		user := models.User{
			ID:    parsedUserId,
			Email: assertedEmail,
		}

		newCtx := context.WithValue(ginCtx.Request.Context(), UserContextKey, user)

		ginCtx.Request = ginCtx.Request.WithContext(newCtx)

		ginCtx.Next()
	}
}

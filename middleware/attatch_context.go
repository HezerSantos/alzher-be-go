package middleware

import (
	"context"

	"github.com/HezerSantos/alzher-api/common/api"
	"github.com/gin-gonic/gin"
)

func MakeCallResultContainer(ctx context.Context) context.Context {
	callResultContainer := api.CallResultContainer{}

	newCtx := context.WithValue(ctx, api.CallResultContextKey, &callResultContainer)

	return newCtx
}
func AttatchContext() gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		newCtx := MakeCallResultContainer(ginCtx.Request.Context())

		ginCtx.Request = ginCtx.Request.WithContext(newCtx)

		ginCtx.Next()
	}
}

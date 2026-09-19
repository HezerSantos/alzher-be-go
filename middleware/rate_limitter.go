package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func RateLimit(rps rate.Limit, burst int) gin.HandlerFunc {
	limiter := rate.NewLimiter(rps, burst)
	return func(ginCtx *gin.Context) {
		if !limiter.Allow() {
			ginCtx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			return
		}

		ginCtx.Next()
	}
}

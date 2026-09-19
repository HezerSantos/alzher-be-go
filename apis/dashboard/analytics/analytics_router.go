package analytics

import (
	"github.com/HezerSantos/alzher-api/middleware"
	"github.com/gin-gonic/gin"
)

func ConnectAnalyticsRouter(r *gin.RouterGroup) {
	activity := r.Group("/analytics")
	activity.GET("", middleware.RateLimit(5, 10), GetAnalyticsHandler)
}

package analytics

import "github.com/gin-gonic/gin"

func ConnectAnalyticsRouter(r *gin.RouterGroup) {
	activity := r.Group("/analytics")
	activity.GET("/", GetAnalyticsHandler)
}

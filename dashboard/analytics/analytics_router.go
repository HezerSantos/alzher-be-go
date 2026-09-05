package analytics

import "github.com/gin-gonic/gin"

func ConnectAnalyticsRouter(r *gin.RouterGroup) {
	r.GET("/analytics", GetAnalyticsHandler)
}

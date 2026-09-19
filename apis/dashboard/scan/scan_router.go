package scan

import (
	"github.com/HezerSantos/alzher-api/middleware"
	"github.com/gin-gonic/gin"
)

func ConnectScanRouter(r *gin.RouterGroup) {
	scan := r.Group("/scan")

	scan.POST("", middleware.RateLimit(5, 5), PostDashboardDocument)
}

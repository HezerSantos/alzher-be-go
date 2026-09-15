package scan

import "github.com/gin-gonic/gin"

func ConnectScanRouter(r *gin.RouterGroup) {
	scan := r.Group("/scan")

	scan.POST("", PostDashboardDocument)
}

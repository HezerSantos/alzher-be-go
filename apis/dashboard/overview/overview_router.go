package overview

import "github.com/gin-gonic/gin"

func ConnectOverviewRouter(r *gin.RouterGroup) {
	overview := r.Group("/overview")
	overview.GET("", GetDashboardOverviewHandler)
}

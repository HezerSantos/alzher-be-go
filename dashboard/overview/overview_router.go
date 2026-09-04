package overview

import "github.com/gin-gonic/gin"

func ConnectOverviewRouter(r *gin.RouterGroup) {
	r.GET("/overview", GetDashboardOverviewHandler)
}

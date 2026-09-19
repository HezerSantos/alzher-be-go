package overview

import (
	"github.com/HezerSantos/alzher-api/middleware"
	"github.com/gin-gonic/gin"
)

func ConnectOverviewRouter(r *gin.RouterGroup) {
	overview := r.Group("/overview")
	overview.GET("", middleware.RateLimit(5, 10), GetDashboardOverviewHandler)
}

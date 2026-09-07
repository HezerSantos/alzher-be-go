package dashboard

import (
	"github.com/HezerSantos/alzher-api/dashboard/activity"
	"github.com/HezerSantos/alzher-api/dashboard/analytics"
	"github.com/HezerSantos/alzher-api/dashboard/overview"
	"github.com/HezerSantos/alzher-api/dashboard/settings"
	"github.com/HezerSantos/alzher-api/middleware"
	"github.com/gin-gonic/gin"
)

func ConnectDashboardRouter(r *gin.RouterGroup) {
	dashboard := r.Group("/dashboard", middleware.AuthMiddleware())
	overview.ConnectOverviewRouter(dashboard)
	analytics.ConnectAnalyticsRouter(dashboard)
	activity.ConnectActivityRouter(dashboard)
	settings.ConnectSettingsRouter(dashboard)
}

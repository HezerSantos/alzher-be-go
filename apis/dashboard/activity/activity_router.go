package activity

import (
	"github.com/HezerSantos/alzher-api/middleware"
	"github.com/gin-gonic/gin"
)

func ConnectActivityRouter(r *gin.RouterGroup) {
	activity := r.Group("/activity")
	activity.GET("", middleware.RateLimit(5, 10), GetActivityHandler)

	activity.DELETE("/:id", middleware.RateLimit(5, 10), DeleteActivityByIDHandler)
	activity.PATCH("/:id", middleware.RateLimit(5, 10), PatchActivityByIDHandler)
}

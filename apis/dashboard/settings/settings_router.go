package settings

import (
	"github.com/HezerSantos/alzher-api/middleware"
	"github.com/gin-gonic/gin"
)

func ConnectSettingsRouter(r *gin.RouterGroup) {
	settings := r.Group("/settings")
	settings.GET("", middleware.RateLimit(5, 10), GetSettingsHandler)
	settings.PATCH("/email", middleware.RateLimit(5, 10), PatchUserEmail)
	settings.PATCH("/password", middleware.RateLimit(5, 10), PatchUserPassword)
}

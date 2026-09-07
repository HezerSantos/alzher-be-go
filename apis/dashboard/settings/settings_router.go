package settings

import "github.com/gin-gonic/gin"

func ConnectSettingsRouter(r *gin.RouterGroup) {
	settings := r.Group("/settings")
	settings.GET("/", GetSettingsHandler)
}

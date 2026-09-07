package activity

import "github.com/gin-gonic/gin"

func ConnectActivityRouter(r *gin.RouterGroup) {
	activity := r.Group("/activity")
	activity.GET("/", GetActivityHandler)

}

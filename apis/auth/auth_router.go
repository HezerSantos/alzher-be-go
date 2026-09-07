package auth

import "github.com/gin-gonic/gin"

func ConnectAuthRouter(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	auth.POST("/login", GetAuthTokenHandler)
}

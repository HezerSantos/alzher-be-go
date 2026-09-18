package auth

import "github.com/gin-gonic/gin"

func ConnectAuthRouter(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	auth.POST("/login", GetAuthTokenHandler)
	auth.POST("/signup", CreateUserHandler)
	auth.POST("/logout", LogoutUserHandler)
}

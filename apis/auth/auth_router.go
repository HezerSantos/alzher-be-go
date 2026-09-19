package auth

import (
	"github.com/HezerSantos/alzher-api/middleware"
	"github.com/gin-gonic/gin"
)

func ConnectAuthRouter(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	auth.POST("/login", middleware.RateLimit(5, 10), GetAuthTokenHandler)
	auth.POST("/signup", middleware.RateLimit(5, 10), CreateUserHandler)
	auth.POST("/logout", middleware.RateLimit(5, 10), LogoutUserHandler)
}

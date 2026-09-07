package main

import (
	"github.com/HezerSantos/alzher-api/apis/auth"
	"github.com/HezerSantos/alzher-api/apis/dashboard"
	"github.com/gin-gonic/gin"
)

func ConnectRouter(r *gin.RouterGroup) {
	dashboard.ConnectDashboardRouter(r)
	auth.ConnectAuthRouter(r)
}

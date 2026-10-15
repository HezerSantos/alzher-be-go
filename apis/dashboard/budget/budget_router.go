package budget

import (
	"github.com/HezerSantos/alzher-api/middleware"
	"github.com/gin-gonic/gin"
)

func ConnectBudgetRouter(r *gin.RouterGroup) {
	budget := r.Group("/budget")

	budget.POST("", middleware.RateLimit(5, 10), CreateBudgetHandler)
}

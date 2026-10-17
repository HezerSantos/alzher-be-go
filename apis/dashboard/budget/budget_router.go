package budget

import (
	"github.com/HezerSantos/alzher-api/middleware"
	"github.com/gin-gonic/gin"
)

func ConnectBudgetRouter(r *gin.RouterGroup) {
	budget := r.Group("/budget")

	budget.GET("", middleware.RateLimit(5, 10), GetAllBudgetsHandler)
	budget.GET("/:id", middleware.RateLimit(5, 10), GetBudgetByIDHandler)
	budget.POST("", middleware.RateLimit(5, 10), CreateBudgetHandler)
	budget.POST("/:id/activate", middleware.RateLimit(5, 10), ActivateBudgetSectionHandler)
	budget.DELETE("/:id", middleware.RateLimit(5, 10), DeleteBudgetByIDHandler)
	budget.PATCH("/:id", middleware.RateLimit(5, 10), UpdateBudgetHandler)
}

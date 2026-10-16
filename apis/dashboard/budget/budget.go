package budget

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/HezerSantos/alzher-api/common/api"
	"github.com/HezerSantos/alzher-api/common/errorfuncs"
	"github.com/HezerSantos/alzher-api/common/userinfo"
	"github.com/HezerSantos/alzher-api/services/railway"
	"github.com/HezerSantos/alzher-api/services/railway/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BudgetItems struct {
	CategoryName string  `json:"categoryName"`
	Amount       float64 `json:"amount"`
}

type BudgetSections struct {
	SectionName  string        `json:"sectionName"`
	IsDeductions bool          `json:"isDeductions"`
	BudgetItems  []BudgetItems `json:"budgetItems"`
}

type CreateBudgetRequestBody struct {
	BudgetName     string           `json:"budgetName"`
	GrossIncome    float64          `json:"grossIncome"`
	BudgetSections []BudgetSections `json:"budgetSections"`
}

func createNewBudgetMutation(budget *models.Budget) (*models.Budget, error) {
	err := railway.DB.Create(budget).Error

	if err != nil {
		return nil, err
	}
	return budget, nil
}

func CreateBudgetHandler(ginCtx *gin.Context) {
	crc, err := api.GetCallResultContainerContext(ginCtx.Request.Context())

	if err != nil {
		errorfuncs.NetworkError(ginCtx, err, nil)
		return
	}

	user, err := userinfo.GetUserContext(ginCtx.Request.Context())
	if err != nil {
		errorfuncs.UnauthorizedError(ginCtx)
		return
	}

	var requestBody CreateBudgetRequestBody

	if err := ginCtx.ShouldBindBodyWithJSON(&requestBody); err != nil {
		crc.Add("ShouldBindBodyWithJSON()", nil, http.StatusBadRequest, err)
		errorfuncs.BadRequestError(ginCtx, crc)
		return
	}

	var totalIsDeductions int

	for _, sect := range requestBody.BudgetSections {
		if sect.IsDeductions == true {
			totalIsDeductions += 1
		}
	}

	if totalIsDeductions > 1 {
		crc.Add("requestBody.BudgetSections", nil, http.StatusBadRequest, fmt.Errorf("BudgetSection.IsDeducitons Must Be Unique"))
		errorfuncs.BadRequestError(ginCtx, crc)
		return
	}

	if totalIsDeductions == 0 {
		crc.Add("requestBody.BudgetSections", nil, http.StatusBadRequest, fmt.Errorf("At Most One BudgetSection.IsDeducitons Must Be True"))
		errorfuncs.BadRequestError(ginCtx, crc)
		return
	}

	budgetId, err := uuid.NewV7()

	if err != nil {
		crc.Add("uuid.NewV7()", nil, http.StatusInternalServerError, err)
		errorfuncs.NetworkError(ginCtx, nil, crc)
		return
	}
	newBudget := models.Budget{}

	newBudget.Name = requestBody.BudgetName
	newBudget.ID = budgetId
	newBudget.GrossIncome = requestBody.GrossIncome
	newBudget.UserID = user.ID

	for _, section := range requestBody.BudgetSections {
		sectionId, err := uuid.NewV7()

		if err != nil {
			crc.Add("uuid.NewV7()", nil, http.StatusInternalServerError, err)
			errorfuncs.NetworkError(ginCtx, nil, crc)
			return
		}

		newBudgetSection := models.BudgetSection{}
		newBudgetSection.ID = sectionId
		newBudgetSection.BudgetID = budgetId
		newBudgetSection.Name = section.SectionName
		newBudgetSection.IsDeductions = section.IsDeductions

		for _, sectionItem := range section.BudgetItems {
			sectionItemId, err := uuid.NewV7()

			if err != nil {
				crc.Add("uuid.NewV7()", nil, http.StatusInternalServerError, err)
				errorfuncs.NetworkError(ginCtx, nil, crc)
				return
			}
			newBudgetSectionItem := models.BudgetSectionItem{}
			newBudgetSectionItem.ID = sectionItemId
			newBudgetSectionItem.BudgetSectionID = sectionId
			newBudgetSectionItem.CategoryName = sectionItem.CategoryName
			newBudgetSectionItem.Amount = sectionItem.Amount
			newBudgetSection.BudgetSectionItems = append(newBudgetSection.BudgetSectionItems, newBudgetSectionItem)
		}
		newBudget.BudgetSections = append(newBudget.BudgetSections, newBudgetSection)
	}

	createBudget, err := createNewBudgetMutation(&newBudget)

	if err != nil {
		crc.Add("createNewBudgetMutation()", nil, http.StatusInternalServerError, err)
		errorfuncs.NetworkError(ginCtx, nil, crc)
		return
	}

	crc.Add("createNewBudgetMutation()", createBudget, http.StatusCreated, nil)

	ginCtx.JSON(http.StatusOK, gin.H{
		"budget":      createBudget,
		"callResults": crc.CallResults,
	})
}

type ActivateBudgetSectionURI struct {
	ID uuid.UUID `uri:"id" binding:"required"`
}

func getBudgetByIDQuery(budgetId uuid.UUID) (*models.Budget, error) {
	var budget models.Budget

	err := railway.DB.Preload("BudgetSections.BudgetSectionItems").Model(&models.Budget{}).Where(`"id" = ?`, budgetId).First(&budget).Error

	if err != nil {
		return nil, err
	}

	return &budget, nil
}

func getActiveBudgetByQuery(userId uuid.UUID) (*models.Budget, error) {
	var budget models.Budget

	err := railway.DB.Model(&models.Budget{}).Where(`"user_id" = ?`, userId).Where(`"is_active" = ?`, true).First(&budget).Error

	if err != nil {
		return nil, err
	}

	return &budget, nil
}

func updateBudgetIsActiveByIDMutation(budgetId uuid.UUID, status bool) (int64, error) {
	result := railway.DB.Model(&models.Budget{}).
		Where("id = ?", budgetId).
		Update("is_active", status)

	return result.RowsAffected, result.Error
}

func updateBudgetIsActiveByIDAtomicMutation(willBeActive *models.Budget, willBeInactive *models.Budget) error {
	err := railway.DB.Model(&models.Budget{}).Transaction(func(tx *gorm.DB) error {
		if willBeInactive != nil {
			if err := tx.Model(&models.Budget{}).
				Where("id = ?", willBeInactive.ID).
				Update("is_active", false).Error; err != nil {
				return err
			}
		}

		// Second update
		if err := tx.Model(&models.Budget{}).
			Where("id = ?", willBeActive.ID).
			Update("is_active", true).Error; err != nil {
			return err // Automatically rolls back the first update
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func ActivateBudgetSectionHandler(ginCtx *gin.Context) {
	crc, err := api.GetCallResultContainerContext(ginCtx.Request.Context())

	if err != nil {
		errorfuncs.NetworkError(ginCtx, err, nil)
		return
	}

	user, err := userinfo.GetUserContext(ginCtx.Request.Context())
	if err != nil {
		errorfuncs.UnauthorizedError(ginCtx)
		return
	}

	var requestUri ActivateBudgetSectionURI

	if err := ginCtx.ShouldBindUri(&requestUri); err != nil {
		crc.Add("ShouldBindUri()", nil, http.StatusBadRequest, err)
		errorfuncs.BadRequestError(ginCtx, crc)
		return
	}

	budget, err := getBudgetByIDQuery(requestUri.ID)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		crc.Add("getBudgetByIDQuery()", nil, http.StatusNotFound, err)
		ginCtx.JSON(http.StatusNotFound, gin.H{
			"error":       "Budget Not Found",
			"callResults": crc.CallResults,
		})
		return
	}
	if err != nil {
		crc.Add("getBudgetByIDQuery()", nil, http.StatusInternalServerError, err)
		errorfuncs.NetworkError(ginCtx, nil, crc)
		return
	}

	crc.Add("getBudgetByIDQuery()", *budget, http.StatusOK, nil)

	if budget.UserID != user.ID {
		ginCtx.JSON(http.StatusForbidden, gin.H{
			"error": "Forbidden",
		})
		return
	}

	activeBudget, aberr := getActiveBudgetByQuery(requestUri.ID)

	if aberr != nil && !errors.Is(aberr, gorm.ErrRecordNotFound) {
		crc.Add("getActiveBudgetByQuery()", nil, http.StatusInternalServerError, aberr)
		errorfuncs.NetworkError(ginCtx, nil, crc)
		return
	}

	err = updateBudgetIsActiveByIDAtomicMutation(budget, activeBudget)

	if err != nil {
		crc.Add("updateBudgetIsActiveByIDAtomicMutation()", nil, http.StatusInternalServerError, err)
		errorfuncs.NetworkError(ginCtx, nil, crc)
		return
	}

	budget.IsActive = true
	if activeBudget != nil {
		activeBudget.IsActive = false
	}

	ginCtx.JSON(http.StatusOK, gin.H{
		"activeBudget":   budget,
		"inactiveBudget": activeBudget,
		"callResults":    crc.CallResults,
	})

}

type BudgetByIDURIRequest struct {
	ID uuid.UUID `uri:"id" binding:"required"`
}

func GetBudgetByIDHandler(ginCtx *gin.Context) {
	crc, err := api.GetCallResultContainerContext(ginCtx.Request.Context())

	if err != nil {
		errorfuncs.NetworkError(ginCtx, err, nil)
		return
	}

	user, err := userinfo.GetUserContext(ginCtx.Request.Context())
	if err != nil {
		errorfuncs.UnauthorizedError(ginCtx)
		return
	}

	var requestUri BudgetByIDURIRequest

	if err := ginCtx.ShouldBindUri(&requestUri); err != nil {
		crc.Add("ShouldBindUri()", nil, http.StatusBadRequest, err)
		errorfuncs.BadRequestError(ginCtx, crc)
		return
	}

	budget, err := getBudgetByIDQuery(requestUri.ID)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		crc.Add("getBudgetByIDQuery()", nil, http.StatusNotFound, err)
		ginCtx.JSON(http.StatusNotFound, gin.H{
			"error":       "Budget Not Found",
			"callResults": crc.CallResults,
		})
		return
	}
	if err != nil {
		crc.Add("getBudgetByIDQuery()", nil, http.StatusInternalServerError, err)
		errorfuncs.NetworkError(ginCtx, nil, crc)
		return
	}

	crc.Add("getBudgetByIDQuery()", *budget, http.StatusOK, nil)

	if budget.UserID != user.ID {
		ginCtx.JSON(http.StatusForbidden, gin.H{
			"error": "Forbidden",
		})
		return
	}

	ginCtx.JSON(http.StatusOK, gin.H{
		"budget": budget,
	})

}

func deleteBudgetByIDMutation(budgetId uuid.UUID) error {
	err := railway.DB.Delete(&models.Budget{}, budgetId).Error

	if err != nil {
		return err
	}

	return nil
}

func DeleteBudgetByIDHandler(ginCtx *gin.Context) {
	crc, err := api.GetCallResultContainerContext(ginCtx.Request.Context())

	if err != nil {
		errorfuncs.NetworkError(ginCtx, err, nil)
		return
	}

	user, err := userinfo.GetUserContext(ginCtx.Request.Context())
	if err != nil {
		errorfuncs.UnauthorizedError(ginCtx)
		return
	}

	var requestUri BudgetByIDURIRequest

	if err := ginCtx.ShouldBindUri(&requestUri); err != nil {
		crc.Add("ShouldBindUri()", nil, http.StatusBadRequest, err)
		errorfuncs.BadRequestError(ginCtx, crc)
		return
	}

	budget, err := getBudgetByIDQuery(requestUri.ID)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		crc.Add("getBudgetByIDQuery()", nil, http.StatusNotFound, err)
		ginCtx.JSON(http.StatusNotFound, gin.H{
			"error":       "Budget Not Found",
			"callResults": crc.CallResults,
		})
		return
	}
	if err != nil {
		crc.Add("getBudgetByIDQuery()", nil, http.StatusInternalServerError, err)
		errorfuncs.NetworkError(ginCtx, nil, crc)
		return
	}

	crc.Add("getBudgetByIDQuery()", *budget, http.StatusOK, nil)

	if budget.UserID != user.ID {
		ginCtx.JSON(http.StatusForbidden, gin.H{
			"error": "Forbidden",
		})
		return
	}

	err = deleteBudgetByIDMutation(budget.ID)

	if err != nil {
		crc.Add("deleteBudgetByIDMutation()", nil, http.StatusInternalServerError, err)
		errorfuncs.NetworkError(ginCtx, nil, crc)
		return
	}

	ginCtx.JSON(http.StatusOK, gin.H{
		"deletedBudget": budget,
	})

}

func getAllBudgetsQuery(userId uuid.UUID) ([]models.Budget, error) {
	var budgets []models.Budget

	err := railway.DB.Preload("BudgetSections.BudgetSectionItems").Where(`"user_id" = ?`, userId).Find(&budgets).Error

	if err != nil {
		return nil, err
	}

	return budgets, nil
}

func GetAllBudgetsHandler(ginCtx *gin.Context) {
	crc, err := api.GetCallResultContainerContext(ginCtx.Request.Context())

	if err != nil {
		errorfuncs.NetworkError(ginCtx, err, nil)
		return
	}

	user, err := userinfo.GetUserContext(ginCtx.Request.Context())
	if err != nil {
		errorfuncs.UnauthorizedError(ginCtx)
		return
	}

	budgets, err := getAllBudgetsQuery(user.ID)

	if err != nil {
		crc.Add("getAllBudgetsQuery()", nil, http.StatusInternalServerError, err)
		errorfuncs.NetworkError(ginCtx, nil, crc)
		return
	}

	crc.Add("getAllBudgetsQuery()", budgets, http.StatusOK, nil)

	ginCtx.JSON(http.StatusOK, gin.H{
		"budgets": budgets,
	})

}

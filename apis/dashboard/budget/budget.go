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
	CategoryName string  `json:"categoryName"  binding:"required"`
	Amount       float64 `json:"amount"  binding:"required"`
}

type BudgetSections struct {
	SectionName  string        `json:"sectionName" binding:"required"`
	IsDeductions bool          `json:"isDeductions"  binding:"required"`
	BudgetItems  []BudgetItems `json:"budgetItems"  binding:"required"`
}

type CreateBudgetRequestBody struct {
	BudgetName     string           `json:"budgetName" binding:"required"`
	GrossIncome    float64          `json:"grossIncome"  binding:"required"`
	BudgetSections []BudgetSections `json:"budgetSections"  binding:"required"`
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
	ID string `uri:"id" binding:"required"`
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

	requestUriUUID, err := uuid.Parse(requestUri.ID)

	if err != nil {
		crc.Add("uuid.Parse()", nil, http.StatusBadRequest, err)
		errorfuncs.BadRequestError(ginCtx, crc)
		return
	}

	budget, err := getBudgetByIDQuery(requestUriUUID)

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

	activeBudget, aberr := getActiveBudgetByQuery(user.ID)

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
	ID string `uri:"id" binding:"required"`
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

	requestUriUUID, err := uuid.Parse(requestUri.ID)

	if err != nil {
		crc.Add("uuid.Parse()", nil, http.StatusBadRequest, err)
		errorfuncs.BadRequestError(ginCtx, crc)
		return
	}
	budget, err := getBudgetByIDQuery(requestUriUUID)

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
		"budget":      budget,
		"callResults": crc.CallResults,
	})

}

func deleteBudgetByIDMutation(budgetID uuid.UUID) error {
	err := railway.DB.Delete(&models.Budget{}, budgetID).Error
	return err
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

	requestUriUUID, err := uuid.Parse(requestUri.ID)

	if err != nil {
		crc.Add("uuid.Parse()", nil, http.StatusBadRequest, err)
		errorfuncs.BadRequestError(ginCtx, crc)
		return
	}

	budget, err := getBudgetByIDQuery(requestUriUUID)

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
		"callResults":   crc.CallResults,
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
		"budgets":     budgets,
		"callResults": crc.CallResults,
	})

}

type UpdateBudgetRequestBody struct {
	Budget models.Budget `json:"budget" binding:"required"`
}

func updateBudgetSectionByIDMutation(
	crc *api.CallResultContainer,
	budgetID uuid.UUID,
	deletedBudgetSectionIDs []uuid.UUID,
	deletedBudgetSectionItemIDs []uuid.UUID,
	newBudgetSections []models.BudgetSection,
	newBudgetSectionItems map[uuid.UUID]models.BudgetSectionItem,
	updateBudgetSectionItemsMap map[uuid.UUID]interface{},
	updateBudgetInfoMap map[string]interface{},
	updateBudgetSectionMap map[uuid.UUID]map[string]interface{},
) error {
	err := railway.DB.Transaction(func(tx *gorm.DB) error {

		//Delete the budget Sections
		if len(deletedBudgetSectionIDs) != 0 {
			if err := tx.Where("id IN ?", deletedBudgetSectionIDs).Delete(&models.BudgetSection{}).Error; err != nil {
				return err
			}
			crc.Add("deleteBudgetSectionIDs()", deletedBudgetSectionIDs, http.StatusOK, nil)
		} else {
			crc.Add("deleteBudgetSectionIDs()", nil, http.StatusOK, nil)
		}

		//Delete the budget section items
		if len(deletedBudgetSectionItemIDs) != 0 {
			if err := tx.Where("id IN ?", deletedBudgetSectionItemIDs).Delete(&models.BudgetSectionItem{}).Error; err != nil {
				return err
			}
			crc.Add("deletedBudgetSectionItemIDs()", deletedBudgetSectionItemIDs, http.StatusOK, nil)
		} else {
			crc.Add("deletedBudgetSectionItemIDs()", nil, http.StatusOK, nil)
		}

		//Update the Base Budget Info e.g Gross Income, Name
		if len(updateBudgetInfoMap) != 0 {
			if err := tx.Model(&models.Budget{}).Where("id = ?", budgetID).Updates(updateBudgetInfoMap).Error; err != nil {
				return err
			}
			crc.Add("updateBudgetInfoMap()", updateBudgetInfoMap, http.StatusOK, nil)
		} else {
			crc.Add("updateBudgetInfoMap()", nil, http.StatusOK, nil)
		}

		//Update the Budget Section Info e.g Name
		for budgetSectionId, updateMap := range updateBudgetSectionMap {
			if err := tx.Model(&models.BudgetSection{}).Where("id = ?", budgetSectionId).Updates(updateMap).Error; err != nil {
				return err
			}
		}

		//Update the Budget Section Items
		for itemId, updateItemMap := range updateBudgetSectionItemsMap {
			if err := tx.Model(&models.BudgetSectionItem{}).Where("id = ?", itemId).Updates(updateItemMap).Error; err != nil {
				return err
			}
		}

		if len(updateBudgetSectionItemsMap) != 0 {
			crc.Add("updateBudgetSectionItemsMap()", updateBudgetSectionItemsMap, http.StatusOK, nil)
		} else {
			crc.Add("updateBudgetSectionItemsMap()", nil, http.StatusOK, nil)
		}

		//Insert New Budget Section Items
		for _, item := range newBudgetSectionItems {
			if err := tx.Model(&models.BudgetSectionItem{}).Create(&item).Error; err != nil {
				return err
			}
		}

		if len(newBudgetSectionItems) != 0 {
			crc.Add("newBudgetSectionItems()", newBudgetSectionItems, http.StatusOK, nil)
		} else {
			crc.Add("newBudgetSectionItems()", nil, http.StatusOK, nil)
		}

		if len(newBudgetSections) != 0 {
			if err := tx.Model(&models.BudgetSection{}).Create(&newBudgetSections).Error; err != nil {
				return err
			}
			crc.Add("newBudgetSections()", newBudgetSections, http.StatusOK, nil)
		} else {
			crc.Add("newBudgetSections()", nil, http.StatusOK, nil)
		}

		return nil
	})

	return err
}

func UpdateBudgetHandler(ginCtx *gin.Context) {

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

	requestUriUUID, err := uuid.Parse(requestUri.ID)

	if err != nil {
		crc.Add("uuid.Parse()", nil, http.StatusBadRequest, err)
		errorfuncs.BadRequestError(ginCtx, crc)
		return
	}

	var requestBody UpdateBudgetRequestBody

	if err := ginCtx.ShouldBindBodyWithJSON(&requestBody); err != nil {
		crc.Add("ShouldBindBodyWithJSON()", nil, http.StatusBadRequest, err)
		errorfuncs.BadRequestError(ginCtx, crc)
		return
	}

	if requestUriUUID != requestBody.Budget.ID {
		crc.Add("MatchError()", nil, http.StatusBadRequest, fmt.Errorf("Budget IDs Do Not Match"))
		errorfuncs.BadRequestError(ginCtx, crc)
		return
	}

	currentBudget, err := getBudgetByIDQuery(requestUriUUID)

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

	crc.Add("getBudgetByIDQuery()", *currentBudget, http.StatusOK, nil)

	if currentBudget.UserID != user.ID {
		ginCtx.JSON(http.StatusForbidden, gin.H{
			"error": "Forbidden",
		})
		return
	}

	updateBudgetInfoMap := map[string]interface{}{}

	if currentBudget.GrossIncome != requestBody.Budget.GrossIncome && requestBody.Budget.GrossIncome != 0 {
		updateBudgetInfoMap["gross_income"] = requestBody.Budget.GrossIncome
	}
	if currentBudget.Name != requestBody.Budget.Name {
		updateBudgetInfoMap["name"] = requestBody.Budget.Name
	}

	type CurrentBudgetInfo struct {
		CurrentBudgetSections map[uuid.UUID]models.BudgetSection //Sections sent from database
		RequestBudgetSections map[uuid.UUID]models.BudgetSection // Sections sent in the request
		NewBudgetSections     map[uuid.UUID]models.BudgetSection //Sections that are not in the Current
		DeletedBudgetSections map[uuid.UUID]models.BudgetSection //Sections that are in current but not in the request
		UpdateBudgetSections  map[uuid.UUID]models.BudgetSection //Sections in the current that have potential changes
	}

	budgetInfo := CurrentBudgetInfo{
		CurrentBudgetSections: map[uuid.UUID]models.BudgetSection{},
		RequestBudgetSections: map[uuid.UUID]models.BudgetSection{},
		NewBudgetSections:     map[uuid.UUID]models.BudgetSection{},
		DeletedBudgetSections: map[uuid.UUID]models.BudgetSection{},
		UpdateBudgetSections:  map[uuid.UUID]models.BudgetSection{},
	}

	var currentDeductionBudgetSectionID uuid.UUID
	updateBudgetSectionMap := map[uuid.UUID]map[string]interface{}{}
	//Loop to add the current budget stored in Postgres to the map
	for _, budgetSection := range currentBudget.BudgetSections {
		budgetInfo.CurrentBudgetSections[budgetSection.ID] = budgetSection
		if budgetSection.IsDeductions {
			currentDeductionBudgetSectionID = budgetSection.ID
		}
	}

	//Loop to add the updated budget NOT stored in Postgres to the map
	//AND
	//Check to find new and updated budget sections
	for _, budgetSection := range requestBody.Budget.BudgetSections {
		budgetInfo.RequestBudgetSections[budgetSection.ID] = budgetSection

		if budgetInCurrent, ok := budgetInfo.CurrentBudgetSections[budgetSection.ID]; !ok {
			if budgetSection.IsDeductions {
				crc.Add("isDeductionsCheck()", nil, http.StatusBadRequest, fmt.Errorf("Duplicate IsDeductions BudgetSection Detected"))
				ginCtx.JSON(http.StatusBadRequest, gin.H{
					"callResults": crc.CallResults,
				})
				return
			}
			budgetInfo.NewBudgetSections[budgetSection.ID] = budgetSection
		} else {
			//If the current IsDeductions potential updated budget section IsDeductions is turned false: Error
			if budgetInCurrent.ID == currentDeductionBudgetSectionID && !budgetSection.IsDeductions {
				crc.Add("isDeductionsCheck()", nil, http.StatusBadRequest, fmt.Errorf("Error overwriting current IsDeductions BudgetSection"))
				ginCtx.JSON(http.StatusBadRequest, gin.H{
					"callResults": crc.CallResults,
				})
				return
			}
			//If the potential updated budget section IsDeductions is turned true that is not current IsDeductions: Error
			if budgetInCurrent.ID != currentDeductionBudgetSectionID && budgetSection.IsDeductions {
				crc.Add("isDeductionsCheck()", nil, http.StatusBadRequest, fmt.Errorf("Error overwriting current IsDeductions BudgetSection"))
				ginCtx.JSON(http.StatusBadRequest, gin.H{
					"callResults": crc.CallResults,
				})
				return
			}
			budgetInfo.UpdateBudgetSections[budgetInCurrent.ID] = budgetSection
			if budgetInCurrent.Name != budgetSection.Name {
				updateBudgetSectionMap[budgetInCurrent.ID] = map[string]interface{}{"name": budgetSection.Name}
			}
		}
	}

	//Check to find deleted budget sections that are not deductions
	for _, budgetSection := range currentBudget.BudgetSections {
		if _, ok := budgetInfo.RequestBudgetSections[budgetSection.ID]; !ok {
			if budgetSection.IsDeductions {
				crc.Add("isDeductionsCheck()", nil, http.StatusBadRequest, fmt.Errorf("Error Deleting IsDeductions BudgetSection"))
				ginCtx.JSON(http.StatusBadRequest, gin.H{
					"callResults": crc.CallResults,
				})
				return
			}
			budgetInfo.DeletedBudgetSections[budgetSection.ID] = budgetSection
		}
	}

	//Process New Budget Sections To Prepare For Insertion
	var newBudgetSections []models.BudgetSection

	for _, budgetSection := range budgetInfo.NewBudgetSections {

		//IsDeductions is Unique. It should never be a new field as it should also never be deleted
		//Meaning it will always exist
		if budgetSection.IsDeductions {
			crc.Add("budgetSection.IsDeductions", nil, http.StatusBadRequest, fmt.Errorf("Duplicate IsDeductions Section Detected"))
			errorfuncs.BadRequestError(ginCtx, crc)
			return
		}
		newBudgetSectionUUID, err := uuid.NewV7()

		if err != nil {
			crc.Add("uuid.NewV7()", nil, http.StatusInternalServerError, err)
			errorfuncs.NetworkError(ginCtx, nil, crc)
			return
		}

		newBudgetSection := budgetSection
		newBudgetSection.ID = newBudgetSectionUUID   //Assert a new Budget Section ID
		newBudgetSection.BudgetID = currentBudget.ID //Assert the current Budget ID

		for i, item := range newBudgetSection.BudgetSectionItems {
			newBudgetSectionItemUUID, err := uuid.NewV7()

			if err != nil {
				crc.Add("uuid.NewV7()", nil, http.StatusInternalServerError, err)
				errorfuncs.NetworkError(ginCtx, nil, crc)
				return
			}
			newBudgetSectionItem := item
			newBudgetSectionItem.ID = newBudgetSectionItemUUID
			newBudgetSectionItem.BudgetSectionID = newBudgetSectionUUID
			newBudgetSection.BudgetSectionItems[i] = newBudgetSectionItem
		}
		newBudgetSections = append(newBudgetSections, newBudgetSection)

	}

	updatedBudgetItems := map[uuid.UUID]interface{}{}
	deletedBudgetItems := map[uuid.UUID]models.BudgetSectionItem{}
	newBudgetItems := map[uuid.UUID]models.BudgetSectionItem{}

	for budgetSecionID, budgetSection := range budgetInfo.UpdateBudgetSections {
		staleBudgetSection, _ := budgetInfo.CurrentBudgetSections[budgetSecionID]

		//Account for new items in the budget section
		//Account for deleted items in the budget section
		//Account for same items but different values in the budget section

		incomingBudgetSectionItems := map[uuid.UUID]models.BudgetSectionItem{}
		staleBudgetSectionItems := map[uuid.UUID]models.BudgetSectionItem{}

		for _, item := range budgetSection.BudgetSectionItems {
			incomingBudgetSectionItems[item.ID] = item
		}

		for _, item := range staleBudgetSection.BudgetSectionItems {
			staleBudgetSectionItems[item.ID] = item
		}

		//deleted and validation check for potential updates
		for itemId, item := range staleBudgetSectionItems {
			if incomingItem, ok := incomingBudgetSectionItems[itemId]; !ok {
				deletedBudgetItems[itemId] = item
			} else {
				tempItem := map[string]interface{}{}
				//It needs to be updated and value checked
				if item.CategoryName != incomingItem.CategoryName {
					tempItem["category_name"] = incomingItem.CategoryName
				}
				if item.Amount != incomingItem.Amount {
					if incomingItem.Amount == 0 {
						crc.Add("incomingItem.AmountCheck()()", nil, http.StatusBadRequest, fmt.Errorf("Error Parsing 0 Amount Value"))
						errorfuncs.BadRequestError(ginCtx, crc)
						return
					}
					tempItem["amount"] = incomingItem.Amount
				}

				if len(tempItem) != 0 {
					//item.ID Asserts the item ID of the exisitng item
					updatedBudgetItems[item.ID] = tempItem
				}
			}
		}

		for itemId, item := range incomingBudgetSectionItems {
			if _, ok := staleBudgetSectionItems[itemId]; !ok {
				newBudgetSectionItemUUID, err := uuid.NewV7()

				if err != nil {
					crc.Add("uuid.NewV7()", nil, http.StatusInternalServerError, err)
					errorfuncs.NetworkError(ginCtx, nil, crc)
					return
				}
				tempItemCopy := item
				tempItemCopy.ID = newBudgetSectionItemUUID
				tempItemCopy.BudgetSectionID = budgetSecionID //Assert the budget id
				newBudgetItems[itemId] = tempItemCopy
			}
		}
	}

	deletedBudgetSectionIDs := make([]uuid.UUID, len(budgetInfo.DeletedBudgetSections))
	deletedBudgetSectionItemIDs := make([]uuid.UUID, len(deletedBudgetItems))

	for id, _ := range budgetInfo.DeletedBudgetSections {
		deletedBudgetSectionIDs = append(deletedBudgetSectionIDs, id)
	}
	for id, _ := range deletedBudgetItems {
		deletedBudgetSectionItemIDs = append(deletedBudgetSectionItemIDs, id)
	}

	if len(deletedBudgetSectionIDs)+
		len(deletedBudgetSectionItemIDs)+
		len(newBudgetSections)+
		len(newBudgetItems)+
		len(updatedBudgetItems)+
		len(updateBudgetInfoMap)+
		len(updateBudgetSectionMap) == 0 {
		ginCtx.JSON(http.StatusOK, gin.H{
			"updatedBudget": nil,
			"staleBudget":   currentBudget,
			"callResults":   crc.CallResults,
		})
		return
	}

	err = updateBudgetSectionByIDMutation(
		crc,
		currentBudget.ID,
		deletedBudgetSectionIDs,
		deletedBudgetSectionItemIDs,
		newBudgetSections,
		newBudgetItems,
		updatedBudgetItems,
		updateBudgetInfoMap,
		updateBudgetSectionMap,
	)

	if err != nil {
		crc.Add("updateBudgetSectionByIDMutation()", nil, http.StatusInternalServerError, err)
		errorfuncs.NetworkError(ginCtx, nil, crc)
		return
	}

	updatedBudget, err := getBudgetByIDQuery(requestUriUUID)

	if err != nil {
		crc.Add("getBudgetByIDQuery()", nil, http.StatusInternalServerError, err)
		errorfuncs.NetworkError(ginCtx, nil, crc)
		return
	}

	crc.Add("getBudgetByIDQuery()", *updatedBudget, http.StatusOK, nil)

	ginCtx.JSON(http.StatusOK, gin.H{
		"updatedBudget": updatedBudget,
		"staleBudget":   currentBudget,
		"callResults":   crc.CallResults,
	})
}

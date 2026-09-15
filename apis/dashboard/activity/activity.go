package activity

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/HezerSantos/alzher-api/common/api"
	"github.com/HezerSantos/alzher-api/common/constants"
	"github.com/HezerSantos/alzher-api/common/errorfuncs"
	"github.com/HezerSantos/alzher-api/common/userinfo"
	"github.com/HezerSantos/alzher-api/services/railway"
	"github.com/HezerSantos/alzher-api/services/railway/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RequestParams struct {
	PageSize       int    `form:"pageSize"`
	Page           int    `form:"page"`
	CategoryFilter string `form:"categoryFilter"`
	KeyWord        string `form:"keyWord"`
}

func buildQuery(id uuid.UUID, params RequestParams) *gorm.DB {
	query := railway.DB.Model(&models.Transaction{}).Where(`"userId" = ?`, id)

	if params.CategoryFilter != "" {
		query = query.Where(`"category" = ?`, params.CategoryFilter)
	}

	if params.KeyWord != "" {
		query = query.Where(`"description" ILIKE ?`, "%"+params.KeyWord+"%")
	}

	return query
}
func queryTransactions(id uuid.UUID, params RequestParams) ([]models.Transaction, error) {
	var transactions []models.Transaction

	query := buildQuery(id, params)
	limit := params.PageSize

	if limit == 0 {
		limit = 10
	}

	offset := (params.Page - 1) * limit
	err := query.Order(`amount DESC`).Limit(limit).Offset(offset).Find(&transactions).Error

	if err != nil {
		return nil, err
	}

	return transactions, err
}

func queryMaxPages(id uuid.UUID, params RequestParams) (*int, error) {
	query := buildQuery(id, params)

	var count int

	err := query.Select("COUNT(*) AS count").Scan(&count).Error

	if err != nil {
		return nil, err
	}

	return &count, nil
}

func GetActivityHandler(ginCtx *gin.Context) {
	crc, err := api.GetCallResultContainerContext(ginCtx.Request.Context())

	if err != nil {
		errorfuncs.NetworkError(ginCtx, err)
		return
	}

	user, err := userinfo.GetUserContext(ginCtx.Request.Context())

	if err != nil {
		errorfuncs.UnauthorizedError(ginCtx)
		return
	}

	var requestParams RequestParams

	if err = ginCtx.ShouldBindQuery(&requestParams); err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if requestParams.Page < 1 {
		requestParams.Page = 1
	}

	if requestParams.PageSize < 1 {
		requestParams.PageSize = 10
	}

	var transactions []models.Transaction
	var maxPages int
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		transactionResult, err := queryTransactions(user.ID, requestParams)
		if err != nil {
			crc.Add("Railway: queryTransactions()", nil, http.StatusInternalServerError, err)
			return
		}

		transactions = transactionResult
		crc.Add("Railway: queryTransactions()", transactionResult, http.StatusOK, nil)
	}()
	go func() {
		defer wg.Done()
		maxPagesResult, err := queryMaxPages(user.ID, requestParams)
		if err != nil {
			crc.Add("Railway: queryMaxPages()", nil, http.StatusInternalServerError, err)
			return
		}
		maxPages = *maxPagesResult
		crc.Add("Railway: queryMaxPages()", maxPagesResult, http.StatusOK, nil)
	}()

	wg.Wait()

	if crc.HasError() {
		ginCtx.JSON(http.StatusInternalServerError, gin.H{
			"callResults": crc.CallResults,
		})
		return
	}

	type TransactionItem struct {
		TransactionID     uuid.UUID `json:"transactionId"`
		Category          string    `json:"category"`
		Description       string    `json:"description"`
		TransactionDate   string    `json:"transactionDate"`
		TransactionAmount string    `json:"transactionAmount"`
	}

	mappedTransactions := make([]TransactionItem, len(transactions))

	if len(transactions) >= 1 {
		for i, t := range transactions {
			transaction := TransactionItem{
				TransactionID:     t.ID,
				Category:          t.Category,
				Description:       t.Description,
				TransactionDate:   fmt.Sprintf("%d/%d/%d", constants.MONTH_ORDER[t.Month], t.Day, t.Year),
				TransactionAmount: fmt.Sprint(math.Round(t.Amount*100) / 100),
			}
			mappedTransactions[i] = transaction
		}
	}

	var previousPageFlag bool
	var nextPageFlag bool
	var transactionData []TransactionItem

	if len(transactions) == 0 {
		previousPageFlag = false
	} else {
		previousPageFlag = requestParams.Page > 1
	}
	if len(transactions) == 0 {
		nextPageFlag = false
	} else {
		nextPageFlag = requestParams.Page < (maxPages / requestParams.PageSize)
	}

	if len(mappedTransactions) > 0 {
		transactionData = mappedTransactions
	}

	ginCtx.JSON(http.StatusOK, gin.H{
		"transactionData":  transactionData,
		"previousPageFlag": previousPageFlag,
		"nextPageFlag":     nextPageFlag,
		"callResults":      crc.CallResults,
	})
}

type DeleteRequestUri struct {
	TransactionID string `uri:"id" binding:"required"`
}

func queryTransactionByID(transactionID uuid.UUID) (*models.Transaction, error) {
	var transaction models.Transaction

	err := railway.DB.Model(&models.Transaction{}).Where(`"id" = ?`, transactionID).Find(&transaction).Error

	if err != nil {
		return nil, err
	}

	if transaction.ID == uuid.Nil {
		return nil, nil
	}

	return &transaction, nil
}

func mutateTransactionByID(transactionID uuid.UUID) (bool, error, *gorm.DB) {
	result := railway.DB.Delete(models.Transaction{}, transactionID)

	if result.Error != nil {
		return false, result.Error, result
	}

	if result.RowsAffected == 0 {
		return false, nil, result
	}

	return true, nil, result
}

func DeleteActivityByIDHandler(ginCtx *gin.Context) {

	crc, err := api.GetCallResultContainerContext(ginCtx.Request.Context())

	if err != nil {
		errorfuncs.NetworkError(ginCtx, err)
		return
	}

	user, err := userinfo.GetUserContext(ginCtx.Request.Context())

	if err != nil {
		errorfuncs.UnauthorizedError(ginCtx)
		return
	}

	var requestParams DeleteRequestUri

	if err := ginCtx.ShouldBindUri(&requestParams); err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	parsedTransactionId, err := uuid.Parse(requestParams.TransactionID)

	if err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	transaction, err := queryTransactionByID(parsedTransactionId)

	if err != nil {
		crc.Add("Railway: queryTransactionByID()", nil, http.StatusInternalServerError, err)

		ginCtx.JSON(http.StatusInternalServerError, gin.H{
			"callResults": crc.CallResults,
		})

		return
	}
	crc.Add("Railway: queryTransactionByID()", transaction, http.StatusOK, nil)

	if transaction == nil {
		ginCtx.JSON(http.StatusNotFound, gin.H{
			"transaction": &transaction,
			"callResults": crc.CallResults,
		})
		return
	}

	if transaction.UserID != user.ID {
		ginCtx.JSON(http.StatusForbidden, gin.H{
			"error": "Unauthorized Transaction Permissions",
		})
		return
	}

	deleteOk, err, res := mutateTransactionByID(parsedTransactionId)

	if err != nil {
		crc.Add("Railway: mutateTransactionByID()", nil, http.StatusInternalServerError, err)
		ginCtx.JSON(http.StatusInternalServerError, gin.H{
			"callResults": crc.CallResults,
		})
		return
	}

	if !deleteOk {
		crc.Add("Railway: mutateTransactionByID()", res.RowsAffected, http.StatusInternalServerError, nil)
		ginCtx.JSON(http.StatusNotFound, gin.H{
			"callResults": crc.CallResults,
		})
		return
	}
	crc.Add("Railway: mutateTransactionByID()", res.RowsAffected, http.StatusOK, nil)

	ginCtx.JSON(http.StatusOK, gin.H{
		"transaction": transaction,
		"callResults": crc.CallResults,
	})

}

type PutRequestBody struct {
	Category        string  `json:"category" binding:"omitempty,oneof=Dining Merchandise Entertainment Grocery Transportation Subscriptions Bills"`
	Description     string  `json:"description"`
	Amount          float64 `json:"amount"`
	TransactionDate string  `json:"transactionDate" binding:"dateformat"`
}

type PutRequestUri struct {
	TransactionID string `uri:"id" binding:"required"`
}

func updateTransactionByID(transaction *models.Transaction, updates map[string]interface{}) (*int64, error) {
	result := railway.DB.Model(transaction).Updates(updates)

	if result.Error != nil {
		return nil, result.Error
	}

	return &result.RowsAffected, nil
}
func PatchActivityByIDHandler(ginCtx *gin.Context) {

	crc, err := api.GetCallResultContainerContext(ginCtx.Request.Context())

	if err != nil {
		errorfuncs.NetworkError(ginCtx, err)
		return
	}

	user, err := userinfo.GetUserContext(ginCtx.Request.Context())

	if err != nil {
		errorfuncs.UnauthorizedError(ginCtx)
		return
	}

	var requestBody PutRequestBody
	var requestUri PutRequestUri

	if err := ginCtx.ShouldBindUri(&requestUri); err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	if err := ginCtx.ShouldBindJSON(&requestBody); err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	parsedTransactionId, err := uuid.Parse(requestUri.TransactionID)

	if err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	transaction, err := queryTransactionByID(parsedTransactionId)

	if err != nil {
		crc.Add("Railway: queryTransactionByID()", nil, http.StatusInternalServerError, err)

		ginCtx.JSON(http.StatusInternalServerError, gin.H{
			"callResults": crc.CallResults,
		})

		return
	}

	crc.Add("Railway: queryTransactionByID()", transaction, http.StatusOK, nil)

	if transaction == nil {
		ginCtx.JSON(http.StatusNotFound, gin.H{
			"transaction": &transaction,
			"callResults": crc.CallResults,
		})
		return
	}

	if transaction.UserID != user.ID {
		ginCtx.JSON(http.StatusForbidden, gin.H{
			"error": "Unauthorized Transaction Permissions",
		})
		return
	}

	updatedTransactionMap := map[string]interface{}{}

	if requestBody.Category != "" && requestBody.Category != transaction.Category {
		updatedTransactionMap["category"] = requestBody.Category
	}
	if requestBody.Description != "" {
		updatedTransactionMap["description"] = requestBody.Description
	}
	if requestBody.Amount != 0 {
		updatedTransactionMap["amount"] = requestBody.Amount
	}
	if requestBody.TransactionDate != "" {
		splitTransactionDate := strings.Split(requestBody.TransactionDate, "/")

		newMonth := splitTransactionDate[0]
		newDay := splitTransactionDate[1]
		newYear := splitTransactionDate[2]

		numDay, _ := strconv.Atoi(newDay)
		numMonth, _ := strconv.Atoi(newMonth)
		numYear, _ := strconv.Atoi(newYear)

		updatedTransactionMap["day"] = numDay
		updatedTransactionMap["year"] = numYear

		var stringMonth string

		for month, order := range constants.MONTH_ORDER {
			if numMonth == order {
				stringMonth = month
				break
			}
		}

		updatedTransactionMap["month"] = stringMonth
	}

	if len(updatedTransactionMap) == 0 {
		ginCtx.JSON(http.StatusOK, gin.H{
			"transaction":        transaction,
			"updatedTransaction": nil,
			"updatedMap":         updatedTransactionMap,
			"callResults":        crc.CallResults,
		})
		return
	}

	rowsAffected, err := updateTransactionByID(transaction, updatedTransactionMap)

	if err != nil {
		crc.Add("Railway: updateTransactionByID()", nil, http.StatusInternalServerError, err)

		ginCtx.JSON(http.StatusInternalServerError, gin.H{
			"callResults": crc.CallResults,
		})
		return
	}

	crc.Add("Railway: updateTransactionByID()", gin.H{"Rows Affected": *rowsAffected}, http.StatusOK, nil)

	ginCtx.JSON(http.StatusOK, gin.H{
		"transaction": transaction,
		"updatedMap":  updatedTransactionMap,
		"callResults": crc.CallResults,
	})
}

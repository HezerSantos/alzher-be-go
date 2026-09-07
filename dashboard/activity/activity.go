package activity

import (
	"fmt"
	"math"
	"net/http"
	"sync"

	"github.com/HezerSantos/alzher-api/common/api"
	callresult "github.com/HezerSantos/alzher-api/common/api/models"
	"github.com/HezerSantos/alzher-api/common/constants"
	"github.com/HezerSantos/alzher-api/common/errorfuncs"
	userinfo "github.com/HezerSantos/alzher-api/common/userInfo"
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
	var callResults []callresult.CallResult
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		transactionResult, err := queryTransactions(user.ID, requestParams)
		if err != nil {
			api.MakeCallResults(&callResults, "Railway: queryTransactions()", nil, http.StatusInternalServerError, err)
			return
		}

		transactions = transactionResult
		api.MakeCallResults(&callResults, "Railway: queryTransactions()", transactionResult, http.StatusOK, nil)
	}()
	go func() {
		defer wg.Done()
		maxPagesResult, err := queryMaxPages(user.ID, requestParams)
		if err != nil {
			api.MakeCallResults(&callResults, "Railway: queryMaxPages()", nil, http.StatusInternalServerError, err)
			return
		}
		maxPages = *maxPagesResult
		api.MakeCallResults(&callResults, "Railway: queryMaxPages()", maxPagesResult, http.StatusOK, nil)
	}()

	wg.Wait()

	for _, cr := range callResults {
		if cr.Error != nil {
			ginCtx.JSON(http.StatusInternalServerError, gin.H{
				"callResults": callResults,
			})
			return
		}
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
		"callResults":      callResults,
	})
}

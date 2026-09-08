package overview

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"sync"

	"github.com/HezerSantos/alzher-api/common/api"
	"github.com/HezerSantos/alzher-api/common/api/types"
	"github.com/HezerSantos/alzher-api/common/constants"
	"github.com/HezerSantos/alzher-api/common/errorfuncs"
	"github.com/HezerSantos/alzher-api/common/userinfo"
	"github.com/HezerSantos/alzher-api/services/railway"
	"github.com/HezerSantos/alzher-api/services/railway/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var SEMESTER_MAP = map[int][]string{
	1: {"Jan", "Feb", "Mar", "Apr", "May", "Jun"},
	2: {"Jul", "Aug", "Sep", "Oct", "Nov", "Dec"},
}

type RequestParams struct {
	Year     int `form:"year" binding:"required"`
	Semester int `form:"semester" binding:"required,oneof=1 2"`
}

type Stats struct {
	Total float64
	Count int64
}

type MonthItems struct {
	Month           string `json:"month"`
	Year            string `json:"year"`
	HighestCategory string `json:"highestCategory"`
	LowestCategory  string `json:"lowestCategory"`
	TotalSpent      string `json:"totalSpent"`
}

func queryUserAggregate(id uuid.UUID, queryYear int) (Stats, error) {
	var stats Stats
	err := railway.DB.
		Model(&models.Transaction{}).
		Select(`
			COALESCE(SUM("amount"), 0) AS total,
			COUNT(*) AS count
		`).
		Where(`"userId" = ?`, id).
		Where(`"year" = ?`, queryYear).
		Scan(&stats).Error

	if err != nil {
		return Stats{}, err
	}
	return stats, nil
}

func queryDistinctMonths(id uuid.UUID, queryYear int) ([]string, error) {

	var months []string

	err := railway.DB.
		Model(&models.Transaction{}).
		Where(`"userId" = ?`, id).
		Where(`"year" = ?`, queryYear).
		Distinct(`"month"`).
		Pluck(`"month"`, &months).Error

	if err != nil {
		return []string{}, err
	}

	return months, nil
}

type CategoryData struct {
	Category          string
	Amount            float64
	TotalTransactions int64
}

type CategoryOverview struct {
	Name              string  `json:"name"`
	Amount            float64 `json:"amount"`
	TotalTransactions int64   `json:"totalTransactions"`
	Percent           float64 `json:"percent"`
}

type TransactionMapData struct {
	Amount   float64
	Month    string
	Category string
}

func queryTransactionMap(id uuid.UUID, queryYear int, selectedSemester []string) (map[string][]TransactionMapData, error) {
	transactionsMap := map[string][]TransactionMapData{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	errorSlice := make([]error, len(selectedSemester))

	for i, month := range selectedSemester {
		wg.Add(1)
		go func(month string, i int) {
			defer wg.Done()
			var transactions []TransactionMapData
			err := railway.DB.
				Model(&models.Transaction{}).
				Select(`"amount", "month", "category"`).
				Where(`"userId" = ?`, id).
				Where(`"year" = ?`, queryYear).
				Where(`"month" = ?`, month).
				Find(&transactions).Error
			if err != nil {
				errorSlice[i] = err
			}
			mu.Lock()
			if transactions != nil {
				transactionsMap[month] = transactions
			}
			mu.Unlock()
		}(month, i)
	}

	wg.Wait()
	for _, err := range errorSlice {
		if err != nil {
			return nil, fmt.Errorf("error fetching transaction map: %w", err)
		}
	}
	return transactionsMap, nil
}

func queryCategoryOverview(id uuid.UUID, queryYear int, userAggregate Stats) ([]CategoryOverview, error) {
	var categoryData []CategoryData

	err := railway.DB.
		Model(&models.Transaction{}).
		Select(`
        "category",
        SUM("amount") AS amount,
        COUNT(*) AS "totalTransactions"
    `).
		Where(`"userId" = ?`, id).
		Where(`"year" = ?`, queryYear).
		Group(`"category"`).
		Order(`amount DESC`).
		Scan(&categoryData).Error

	if err != nil {
		return []CategoryOverview{}, err
	}
	categoryOverview := make([]CategoryOverview, len(categoryData))

	for i, category := range categoryData {
		categoryOverview[i] = CategoryOverview{
			Name:              category.Category,
			Amount:            math.Round(category.Amount*100) / 100,
			TotalTransactions: category.TotalTransactions,
			Percent:           (category.Amount / userAggregate.Total) * 100,
		}
	}

	return categoryOverview, nil
}
func GetDashboardOverviewHandler(ginCtx *gin.Context) {

	user, err := userinfo.GetUserContext(ginCtx.Request.Context())
	if err != nil {
		errorfuncs.UnauthorizedError(ginCtx)
		return
	}
	var RequestParams RequestParams

	if err := ginCtx.ShouldBindQuery(&RequestParams); err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	queryYear := RequestParams.Year
	selectedSemester := SEMESTER_MAP[RequestParams.Semester]

	var callResults []types.CallResult

	var years []int
	err = railway.DB.
		Model(&models.Transaction{}).
		Where(`"userId" = ?`, user.ID).
		Distinct(`"year"`).
		Pluck(`"year"`, &years).Error

	if err != nil {
		api.MakeCallResults(&callResults, "Railway: queryYears()", nil, http.StatusInternalServerError, err)

		ginCtx.JSON(http.StatusInternalServerError, gin.H{
			"callResults": callResults,
		})
		return
	}
	api.MakeCallResults(&callResults, "Railway: queryYears()", years, http.StatusOK, nil)

	if len(years) == 0 {
		ginCtx.JSON(http.StatusNoContent, gin.H{"chartData": nil, "callResults": callResults})
		return
	}

	match := false
	for _, year := range years {
		if year == queryYear {
			match = true
		}
	}

	if match == false {
		sort.Ints(years)
		queryYear = years[len(years)-1]
	}

	var userAggregate Stats
	var distinctMonths []string
	var transactionMap map[string][]TransactionMapData
	var categoryOverview []CategoryOverview
	var wg sync.WaitGroup

	wg.Add(3)

	go func() {
		defer wg.Done()
		stats, err := queryUserAggregate(user.ID, queryYear)

		if err != nil {
			api.MakeCallResults(&callResults, "Railway: queryUserAggregate()", nil, http.StatusInternalServerError, err)
			return
		}
		api.MakeCallResults(&callResults, "Railway: queryUserAggregate()", stats, http.StatusOK, nil)

		categoryOverviewResult, err := queryCategoryOverview(user.ID, queryYear, stats)

		if err != nil {
			api.MakeCallResults(&callResults, "Railway: queryCategoryOverview()", nil, http.StatusInternalServerError, err)
			return
		}
		api.MakeCallResults(&callResults, "Railway: queryCategoryOverview()", categoryOverviewResult, http.StatusOK, nil)

		userAggregate = stats
		categoryOverview = categoryOverviewResult

	}()

	go func() {
		defer wg.Done()
		months, err := queryDistinctMonths(user.ID, queryYear)
		if err != nil {
			api.MakeCallResults(&callResults, "Railway: queryDistinctMonths()", nil, http.StatusInternalServerError, err)
			return
		}
		distinctMonths = months
		api.MakeCallResults(&callResults, "Railway: queryDistinctMonths()", months, http.StatusOK, nil)

	}()

	go func() {
		defer wg.Done()
		transactionMapResult, err := queryTransactionMap(user.ID, queryYear, selectedSemester)
		if err != nil {
			api.MakeCallResults(&callResults, "Railway: queryTransactionMap()", nil, http.StatusInternalServerError, err)

			return
		}
		transactionMap = transactionMapResult
		api.MakeCallResults(&callResults, "Railway: queryTransactionMap()", transactionMapResult, http.StatusOK, nil)

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

	categoryTransactionMap := map[string]map[string]float64{}

	for m, transactionList := range transactionMap {
		categoryMap := map[string]float64{}

		for _, t := range transactionList {

			if categoryTotal, ok := categoryMap[t.Category]; !ok {
				categoryMap[t.Category] = t.Amount
			} else {
				categoryMap[t.Category] = categoryTotal + t.Amount
			}
		}

		categoryTransactionMap[m] = categoryMap
	}

	monthItems := []MonthItems{}

	for m, categoryMap := range categoryTransactionMap {
		var highestCategory, lowestCategory string
		var highestAmount, lowestAmount float64
		var totalSpent float64
		for category, amount := range categoryMap {
			if amount > highestAmount {
				highestAmount = amount
				highestCategory = category
			}

			if amount < lowestAmount || lowestAmount == 0 {
				lowestAmount = amount
				lowestCategory = category
			}
			totalSpent += amount
		}

		monthItem := MonthItems{
			Month:           m,
			Year:            fmt.Sprint(queryYear),
			HighestCategory: highestCategory,
			LowestCategory:  lowestCategory,
			TotalSpent:      fmt.Sprintf("%.2f", totalSpent),
		}

		monthItems = append(monthItems, monthItem)
	}
	sort.Slice(monthItems, func(i, j int) bool {
		return constants.MONTH_ORDER[monthItems[i].Month] < constants.MONTH_ORDER[monthItems[j].Month]
	})

	chartData := []map[string]any{}

	for _, mi := range monthItems {
		value, err := strconv.ParseFloat(mi.TotalSpent, 64)

		if err != nil {
			continue
		}

		tmpMap := map[string]any{}

		tmpMap["month"] = mi.Month
		tmpMap[fmt.Sprint(queryYear)] = math.Round(value*100) / 100

		chartData = append(chartData, tmpMap)
	}

	var peakMonth string
	var peakMonthAmount float64
	for _, cd := range chartData {
		monthTotal, ok := cd[fmt.Sprint(queryYear)]

		if !ok {
			continue
		}

		assertedMonthTotal, ok := monthTotal.(float64)
		if !ok {
			continue
		}

		monthName, ok := cd["month"]
		if !ok {
			continue
		}

		assertedMonthName, ok := monthName.(string)
		if !ok {
			continue
		}

		if assertedMonthTotal > peakMonthAmount {
			peakMonthAmount = assertedMonthTotal
			peakMonth = assertedMonthName
		}
	}

	categoryTotalMap := map[string]float64{}
	for _, categoryMap := range categoryTransactionMap {
		for c, amount := range categoryMap {
			if categoryTotal, ok := categoryTotalMap[c]; !ok {
				categoryTotalMap[c] = amount
			} else {
				categoryTotalMap[c] = categoryTotal + amount
			}
		}
	}

	var highestCategory string
	var highestCategoryAmount float64

	for category, amount := range categoryTotalMap {
		if amount > highestCategoryAmount {
			highestCategory = category
			highestCategoryAmount = amount
		}
	}

	overviewDetailsItems := []gin.H{
		{
			"header": "Total Spent",
			"price":  fmt.Sprintf("%.2f", userAggregate.Total),
			"details": []struct {
				Heading string `json:"heading"`
				Value   string `json:"value"`
			}{
				{
					Heading: "Peak Month",
					Value:   peakMonth,
				},
				{
					Heading: "Largest Expense",
					Value:   highestCategory,
				},
			},
		},
		{
			"header": "Monthly Average",
			"price":  fmt.Sprintf("%.2f", userAggregate.Total/float64(len(distinctMonths))),
		},
		{
			"header": "Total Transactions",
			"price":  userAggregate.Count,
		},
	}

	var chartDataOverride *[]map[string]any
	var monthItemsOverride *[]MonthItems

	if len(chartData) != 0 {
		chartDataOverride = &chartData
	}

	if len(monthItems) != 0 {
		monthItemsOverride = &monthItems
	}
	ginCtx.JSON(http.StatusOK, gin.H{
		"year":                 fmt.Sprint(queryYear),
		"semester":             RequestParams.Semester,
		"overviewDetailsItems": overviewDetailsItems,
		"chartData":            chartDataOverride,
		"monthItems":           monthItemsOverride,
		"yearList":             years,
		"categoryOverview":     categoryOverview,
		"callResults":          callResults,
	})
}

package overview

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"sync"

	userinfo "github.com/HezerSantos/alzher-api/services/common/userInfo"
	"github.com/HezerSantos/alzher-api/services/railway"
	"github.com/HezerSantos/alzher-api/services/railway/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var SEMESTER_MAP = map[int][]string{
	1: {"Jan", "Feb", "Mar", "Apr", "May", "Jun"},
	2: {"Jul", "Aug", "Sep", "Oct", "Nov", "Dec"},
}

var MONTH_ORDER = map[string]int{
	"Jan": 1,
	"Feb": 2,
	"Mar": 3,
	"Apr": 4,
	"May": 5,
	"Jun": 6,
	"Jul": 7,
	"Aug": 8,
	"Sep": 9,
	"Oct": 10,
	"Nov": 11,
	"Dec": 12,
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

func queryTransactionMap(id uuid.UUID, queryYear int, selectedSemester []string) (map[string][]models.Transaction, error) {
	transactionsMap := map[string][]models.Transaction{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	errorSlice := make([]error, len(selectedSemester))

	for i, month := range selectedSemester {
		wg.Add(1)
		go func(month string, i int) {
			defer wg.Done()
			var transactions []models.Transaction
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
			return map[string][]models.Transaction{}, fmt.Errorf("error fetching transaction map: %w", err)
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
		ginCtx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var RequestParams RequestParams

	if err := ginCtx.ShouldBindQuery(&RequestParams); err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	queryYear := RequestParams.Year
	selectedSemester := SEMESTER_MAP[RequestParams.Semester]

	var years []int
	err = railway.DB.
		Model(&models.Transaction{}).
		Where(`"userId" = ?`, user.ID).
		Distinct(`"year"`).
		Pluck(`"year"`, &years).Error

	if err != nil {
		ginCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query distinct years"})
		return
	}

	if len(years) == 0 {
		ginCtx.JSON(http.StatusNoContent, gin.H{"chartData": nil})
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
	var transactionMap map[string][]models.Transaction
	var categoryOverview []CategoryOverview
	errorSlice := make([]error, 3)
	var wg sync.WaitGroup

	wg.Add(3)

	go func() {
		defer wg.Done()
		stats, err := queryUserAggregate(user.ID, queryYear)

		if err != nil {
			errorSlice[1] = err
			return
		}
		categoryOverviewResult, err := queryCategoryOverview(user.ID, queryYear, stats)

		if err != nil {
			errorSlice[1] = err
			return
		}
		userAggregate = stats
		categoryOverview = categoryOverviewResult

	}()

	go func() {
		defer wg.Done()
		months, err := queryDistinctMonths(user.ID, queryYear)
		if err != nil {
			errorSlice[2] = err
			return
		}
		distinctMonths = months

	}()

	go func() {
		defer wg.Done()
		transactionMapResult, err := queryTransactionMap(user.ID, queryYear, selectedSemester)
		if err != nil {
			errorSlice[3] = err
			return
		}
		transactionMap = transactionMapResult
	}()

	wg.Wait()

	for _, e := range errorSlice {
		if e != nil {
			ginCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Error Fetching Transaction Data"})
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
		return MONTH_ORDER[monthItems[i].Month] < MONTH_ORDER[monthItems[j].Month]
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
	ginCtx.JSON(http.StatusAccepted, gin.H{
		"year":                 fmt.Sprint(queryYear),
		"semester":             RequestParams.Semester,
		"overviewDetailsItems": overviewDetailsItems,
		"chartData":            chartDataOverride,
		"monthItems":           monthItemsOverride,
		"yearList":             years,
		"categoryOverview":     categoryOverview,
	})
}

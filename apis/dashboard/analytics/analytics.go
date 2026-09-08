package analytics

import (
	"fmt"
	"math"
	"net/http"
	"slices"
	"sort"
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

type Stats struct {
	TotalSpent float64
}

type CategoryCount struct {
	Category string
	Count    int64
}

type CategorySum struct {
	Category string
	Total    float64
}

type YearlySums struct {
	Year  int     `json:"year"`
	Count int     `json:"count,omitempty"`
	Total float64 `json:"total"`
}

type MonthlySums struct {
	Month string
	Total float64
}

type DailySums struct {
	Day   int     `json:"dateOfMonth"`
	Count int     `json:"count,omitempty"`
	Total float64 `json:"dailyAverage"`
}

type TotalTransactions struct {
	Count int
}

type DashboardAnalyticsInfoType struct {
	Header             string   `json:"header"`
	AmountSpent        *float64 `json:"amountSpent"`
	PrimarySubHeader   *string  `json:"primarySubHeader"`
	PrimarySubValue    *string  `json:"primarySubValue"`
	SecondarySubHeader *string  `json:"secondarySubHeader"`
	SecondarySubValue  *string  `json:"secondarySubValue"`
}

type MonthlyYearlySums struct {
	Month string
	Year  int
	Total float64
}

func queryTotalSpent(id uuid.UUID) (float64, error) {
	var stats Stats
	err := railway.DB.
		Model(&models.Transaction{}).
		Select(`
			SUM(amount) as totalSpent
		`).
		Where(`"userId" = ?`, id).
		Row().
		Scan(&stats.TotalSpent)
	if err != nil {
		return 0, err
	}

	return stats.TotalSpent, nil
}

func queryMostFrequentCategory(id uuid.UUID) (string, error) {
	var result []CategoryCount

	err := railway.DB.Model(&models.Transaction{}).
		Select(`"category", COUNT(*) AS count`).
		Where(`"userId" = ?`, id).
		Group(`"category"`).
		Order(`count DESC`).
		Limit(1).
		Scan(&result).Error

	if err != nil {
		return "", err
	}
	if len(result) == 0 {
		return "", nil
	}
	return result[0].Category, nil
}

func queryHighestCategory(id uuid.UUID) (string, []CategorySum, error) {
	var result []CategorySum

	err := railway.DB.
		Model(&models.Transaction{}).
		Select(`
			"category",
			SUM(amount) AS total
		`).
		Where(`"userId" = ?`, id).
		Group("category").
		Order(`total DESC`).
		Scan(&result).Error

	if err != nil {
		return "", nil, err
	}
	if len(result) == 0 {
		return "", nil, nil
	}
	return result[0].Category, result, nil
}

func queryYearlySums(id uuid.UUID) ([]YearlySums, error) {
	var result []YearlySums

	err := railway.DB.
		Model(&models.Transaction{}).
		Select(`
			"year",
			COUNT(*) AS count,
			SUM(amount) as total
		`).
		Where(`"userId" = ?`, id).
		Group("year").
		Order("year DESC").
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return result, nil
}

func queryMonthlySums(id uuid.UUID) ([]MonthlySums, error) {
	var result []MonthlySums

	err := railway.DB.
		Model(&models.Transaction{}).
		Select(`
			"month",
			SUM(amount) as total
		`).
		Where(`"userId" = ?`, id).
		Group("month").
		Order("total DESC").
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return result, nil
}

func queryDailySums(id uuid.UUID) ([]DailySums, error) {
	var result []DailySums

	err := railway.DB.
		Model(&models.Transaction{}).
		Select(`
			"day",
			COUNT(*) AS count,
			SUM(amount) as total
		`).
		Where(`"userId" = ?`, id).
		Group("day").
		Order("total DESC").
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return result, nil
}

func queryTotalTransactions(id uuid.UUID) (int, error) {
	var totalTransactions TotalTransactions

	err := railway.DB.
		Model(&models.Transaction{}).
		Select(`
			COUNT(*) AS count
		`).
		Where(`"userId" = ?`, id).
		Scan(&totalTransactions).Error

	if err != nil {
		return 0, err
	}

	return totalTransactions.Count, nil
}

func queryYearlyMonthlyData(id uuid.UUID) ([]MonthlyYearlySums, error) {
	var result []MonthlyYearlySums

	err := railway.DB.
		Model(&models.Transaction{}).
		Select(`
			"year",
			"month",
			SUM(amount) as Total
		`).
		Where(`"userId" = ?`, id).
		Group(`"year", "month"`).
		Order("year DESC").
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return result, nil
}

func GetAnalyticsHandler(ginCtx *gin.Context) {

	user, err := userinfo.GetUserContext(ginCtx.Request.Context())

	if err != nil {
		errorfuncs.UnauthorizedError(ginCtx)
		return
	}

	var totalSpent float64
	var mostFrequentCategory string
	var highestCategory string
	var categoryExpense []CategorySum
	var yearlySums []YearlySums
	var monthlySums []MonthlySums
	var dailySums []DailySums
	var totalTransactionCount int
	var callResults []types.CallResult
	var wg sync.WaitGroup

	wg.Add(7)
	go func() {
		defer wg.Done()
		totalSpentResult, err := queryTotalSpent(user.ID)
		if err != nil {
			api.MakeCallResults(&callResults, "Railway: queryTotalSpent()", nil, http.StatusInternalServerError, err)
			return
		}

		totalSpent = totalSpentResult
		api.MakeCallResults(&callResults, "Railway: queryTotalSpent()", totalSpentResult, http.StatusOK, nil)
	}()
	go func() {
		defer wg.Done()
		mostFrequentCategoryResult, err := queryMostFrequentCategory(user.ID)
		if err != nil {
			api.MakeCallResults(&callResults, "Railway: queryMostFrequentCategory()", nil, http.StatusInternalServerError, err)
			return
		}
		mostFrequentCategory = mostFrequentCategoryResult
		api.MakeCallResults(&callResults, "Railway: queryMostFrequentCategory()", mostFrequentCategoryResult, http.StatusOK, nil)

	}()
	go func() {
		defer wg.Done()
		highestCategoryResult, categorySums, err := queryHighestCategory(user.ID)
		if err != nil {
			api.MakeCallResults(&callResults, "Railway: queryHighestCategory()", nil, http.StatusInternalServerError, err)
			return
		}
		highestCategory = highestCategoryResult
		categoryExpense = categorySums
		api.MakeCallResults(&callResults, "Railway: queryHighestCategory()", highestCategoryResult, http.StatusOK, nil)
		api.MakeCallResults(&callResults, "Railway: queryHighestCategory()", categorySums, http.StatusOK, nil)

	}()
	go func() {
		defer wg.Done()
		yearlySumsResult, err := queryYearlySums(user.ID)
		if err != nil {
			api.MakeCallResults(&callResults, "Railway: queryYearlySums()", nil, http.StatusInternalServerError, err)
			return
		}
		yearlySums = yearlySumsResult
		api.MakeCallResults(&callResults, "Railway: queryYearlySums()", yearlySumsResult, http.StatusOK, nil)

	}()
	go func() {
		defer wg.Done()
		monthlySumsResult, err := queryMonthlySums(user.ID)
		if err != nil {
			api.MakeCallResults(&callResults, "Railway: queryMonthlySums()", nil, http.StatusInternalServerError, err)
			return
		}
		monthlySums = monthlySumsResult
		api.MakeCallResults(&callResults, "Railway: queryMonthlySums()", monthlySumsResult, http.StatusOK, nil)

	}()
	go func() {
		defer wg.Done()
		dailySumsResult, err := queryDailySums(user.ID)
		if err != nil {
			api.MakeCallResults(&callResults, "Railway: queryDailySums()", nil, http.StatusInternalServerError, err)
			return
		}
		dailySums = dailySumsResult
		api.MakeCallResults(&callResults, "Railway: queryDailySums()", dailySumsResult, http.StatusOK, nil)

	}()
	go func() {
		defer wg.Done()
		totalTransactionCountResult, err := queryTotalTransactions(user.ID)
		if err != nil {
			api.MakeCallResults(&callResults, "Railway: queryTotalTransactions()", nil, http.StatusInternalServerError, err)
			return
		}
		totalTransactionCount = totalTransactionCountResult
		api.MakeCallResults(&callResults, "Railway: queryTotalTransactions()", totalTransactionCountResult, http.StatusOK, nil)

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

	if totalSpent == 0 {
		ginCtx.JSON(http.StatusNoContent, gin.H{
			"dashboardAnalyticsInfo": nil,
			"yearlyData":             nil,
			"categoryData":           nil,
			"overviewData":           nil,
			"scatterData":            nil,
		})
		return
	}

	var yearTotal float64

	for _, year := range yearlySums {
		yearTotal += year.Total
	}

	yearlyAverage := yearTotal / float64(len(yearlySums))

	monthAverage := yearlyAverage / float64(len(monthlySums))

	highestDay := dailySums[0].Day
	lowestDay := dailySums[len(dailySums)-1].Day

	frequentCategory := "Frequent Category"
	largestCategory := "Largest Expense"
	peakMonth := "Peak Month"
	highestDayLabel := "Highest Day"
	hom := fmt.Sprintf("%d of the month", highestDay)
	lowestDayLabel := "Lowest Day"
	lom := fmt.Sprintf("%d of the month", lowestDay)
	convertedTotal := float64(totalTransactionCount)

	dashboardAnalyticsInfo := []DashboardAnalyticsInfoType{
		{
			Header:             "Total Spent",
			AmountSpent:        &totalSpent,
			PrimarySubHeader:   &frequentCategory,
			PrimarySubValue:    &mostFrequentCategory,
			SecondarySubHeader: &largestCategory,
			SecondarySubValue:  &highestCategory,
		},
		{
			Header:           "Yearly Average",
			AmountSpent:      &yearlyAverage,
			PrimarySubHeader: &peakMonth,
			PrimarySubValue:  &monthlySums[0].Month,
		},
		{
			Header:             "Monthly Average",
			AmountSpent:        &monthAverage,
			PrimarySubHeader:   &highestDayLabel,
			PrimarySubValue:    &hom,
			SecondarySubHeader: &lowestDayLabel,
			SecondarySubValue:  &lom,
		},
		{
			Header:      "Total Transactions",
			AmountSpent: &convertedTotal,
		},
	}

	yearlyLineChart := make([]YearlySums, len(yearlySums))

	for i, y := range yearlySums {

		yearlyLineChart[i] = YearlySums{Year: y.Year, Total: math.Round(y.Total*100) / 100}
	}

	monthlyYearlyData, err := queryYearlyMonthlyData(user.ID)

	if err != nil {
		api.MakeCallResults(&callResults, "Railway: queryYearlyMonthlyData()", nil, http.StatusInternalServerError, err)
		ginCtx.JSON(http.StatusInternalServerError, gin.H{
			"callResults": callResults,
		})
		return
	}
	api.MakeCallResults(&callResults, "Railway: queryYearlyMonthlyData()", monthlyYearlyData, http.StatusOK, nil)

	sort.Slice(monthlyYearlyData, func(i, j int) bool {
		return constants.MONTH_ORDER[monthlyYearlyData[i].Month] < constants.MONTH_ORDER[monthlyYearlyData[j].Month]
	})

	type month string
	type year int

	cleanedMonthlyYearlyData := map[month]map[year]float64{}

	for _, myData := range monthlyYearlyData {
		if yearMap, ok := cleanedMonthlyYearlyData[month(myData.Month)]; !ok {
			newYearMap := map[year]float64{
				year(myData.Year): myData.Total,
			}
			cleanedMonthlyYearlyData[month(myData.Month)] = newYearMap
		} else {
			if _, ok := yearMap[year(myData.Year)]; !ok {
				yearMap[year(myData.Year)] = myData.Total
			}
		}
	}

	type ChartData map[string]any

	monthlyBarChartData := make([]ChartData, 12)

	for m, i := range constants.MONTH_ORDER {
		yearMap, ok := cleanedMonthlyYearlyData[month(m)]
		if !ok {
			continue
		}

		years := make([]year, 0, len(yearMap))
		for y := range yearMap {
			years = append(years, y)
		}

		// Highest/latest years first
		sort.Slice(years, func(i, j int) bool {
			return years[i] > years[j]
		})

		data := ChartData{
			"month": m,
		}

		for _, y := range years[:min(2, len(years))] {
			data[fmt.Sprint(y)] = math.Round(yearMap[y]*100) / 100
		}

		monthlyBarChartData[i-1] = data
	}

	scatterData := make([]DailySums, len(dailySums))

	for i, d := range dailySums {
		scatterData[i] = DailySums{
			Day:   d.Day,
			Total: math.Round(d.Total/float64(d.Count)*100) / 100,
		}
	}

	uniqueYears := map[int]struct{}{}
	uniqueMonths := map[string]struct{}{}
	uniqueDays := map[int]struct{}{}

	for _, year := range yearlySums {
		if _, ok := uniqueYears[year.Year]; !ok {
			uniqueYears[year.Year] = struct{}{}
		}
	}
	for _, month := range monthlySums {
		if _, ok := uniqueMonths[month.Month]; !ok {
			uniqueMonths[month.Month] = struct{}{}
		}
	}
	for _, day := range dailySums {
		if _, ok := uniqueDays[day.Day]; !ok {
			uniqueDays[day.Day] = struct{}{}
		}
	}

	categoryChartData := []gin.H{}

	for _, category := range categoryExpense {
		newData := gin.H{
			"category":        category.Category,
			"Yearly Average":  math.Round(category.Total/float64(len(uniqueYears))*100) / 100,
			"Monthly Average": math.Round(category.Total/float64(len(uniqueMonths))*100) / 100,
			"Daily Average":   math.Round(category.Total/float64(len(uniqueDays))*100) / 100,
			"total":           math.Round(category.Total*100) / 100,
		}
		categoryChartData = append(categoryChartData, newData)
	}
	slices.Reverse(yearlyLineChart)

	ginCtx.JSON(http.StatusOK, gin.H{
		"dashboardAnalyticsInfo": dashboardAnalyticsInfo,
		"yearlyData":             yearlyLineChart,
		"categoryData":           categoryChartData,
		"overviewData":           monthlyBarChartData,
		"scatterData":            scatterData,
		"callResults":            callResults,
	})
}

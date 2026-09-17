package alzherml

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/HezerSantos/alzher-api/common/api"
	"github.com/HezerSantos/alzher-api/common/constants"
	"github.com/HezerSantos/alzher-api/services/common/ai"
)

type AlzherMLResult struct {
	TransactionArray []Transaction `json:"transactionArray"`
}

type Transaction struct {
	Month       string  `json:"month"`
	Day         int     `json:"day"`
	Year        int     `json:"year"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Category    string  `json:"category"`
}

func processDate(date string) (*string, *string, error) {
	splitDate := strings.Split(date, "/")

	if len(splitDate) != 3 {
		return nil, nil, fmt.Errorf("Date Not of Type D/M/YYYY:")
	}

	month := splitDate[0]
	day := splitDate[1]
	year := splitDate[2]

	intMonth, err := strconv.Atoi(month)

	if err != nil {
		return nil, nil, err
	}

	wordMonth := constants.MONTHS[intMonth]

	monthDay := fmt.Sprintf("%s %s", wordMonth, day)

	return &monthDay, &year, nil
}

func formatTransactions(transactions []ai.Transaction) ([][]interface{}, error) {
	var formattedTransactions = make([][]interface{}, len(transactions))

	for i, t := range transactions {
		ft := make([]interface{}, 4)

		monthDay, year, err := processDate(t.PostDate)

		if err != nil {
			return nil, err
		}

		ft[0] = monthDay
		ft[1] = year
		ft[2] = t.Description
		ft[3] = t.Amount

		formattedTransactions[i] = ft
	}

	return formattedTransactions, nil
}

type AlzherMLRequest struct {
	Transactions [][]interface{} `json:"transactions"`
}

func FetchTransactionCategories(ctx context.Context, transactions []ai.Transaction, crc *api.CallResultContainer) ([]Transaction, error) {

	var ALZHER_ML_URL = os.Getenv("ALZHER_ML_URL")
	var ALZHER_ML_API_KEY = os.Getenv("ALZHER_ML_API_KEY")

	if ALZHER_ML_URL == "" {
		return nil, fmt.Errorf("Alzher ML Url Not Configured")
	}

	if ALZHER_ML_API_KEY == "" {
		return nil, fmt.Errorf("Alzher ML API KEY Not Configured")
	}

	formattedTransactions, err := formatTransactions(transactions)

	if err != nil {
		return nil, err
	}

	newRequest := AlzherMLRequest{
		Transactions: formattedTransactions,
	}

	jsonBytes, err := json.Marshal(newRequest)

	if err != nil {
		return nil, err
	}

	bytesReader := bytes.NewReader(jsonBytes)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ALZHER_ML_URL, bytesReader)

	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", ALZHER_ML_API_KEY)

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)

	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 400 {
		crc.SetStatus(res.StatusCode)
		crc.Add("AlzherML: FetchTransactionCategories()", string(bodyBytes), res.StatusCode, fmt.Errorf("Alzher ML Failure"))
		return nil, fmt.Errorf("Alzher ML Failure")
	}

	var AlzherMLResult AlzherMLResult

	err = json.Unmarshal(bodyBytes, &AlzherMLResult)

	if err != nil {
		return nil, err
	}

	return AlzherMLResult.TransactionArray, nil
}

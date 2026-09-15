package alzherml

import (
	"context"
	"net/http"
	"strings"

	"github.com/HezerSantos/alzher-api/common/constants"
	"github.com/HezerSantos/alzher-api/services/common/ai"
)

type AlzherMLResult struct {
	TransactionArray []transaction `json:"transactionArray"`
}

type transaction struct {
	Month       string  `json:"month"`
	Day         int     `json:"day"`
	Year        int     `json:"year"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Category    string  `json:"category"`
}

func processDate(date string) (string, string){
	splitDate := strings.Split(date, "/")

	
	return fmt.Sprintf("%s %s", splitDate[])
}
func formatTransactions(transactions []ai.Transaction) {
	 var formattedTransactions [][]interface{}

	for i, t := range transactions {
		ft := make([]interface{}, 4)

		ft[2] = t.Description
		ft[3] = t.Amount
	}
}
func FetchTransactionCategories(ctx context.Context, transactions []ai.Transaction) (AlzherMLResult, error) {

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "", )
}

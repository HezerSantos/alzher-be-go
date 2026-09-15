package ai

type TransactionResult struct {
	Transactions []Transaction `json:"transactions"`
}

type Transaction struct {
	PostDate    string  `json:"postDate"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
}

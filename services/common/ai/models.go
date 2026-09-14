package ai

type Transaction struct {
	Transactions []struct {
		PostDate    string  `json:"postDate"`
		Description string  `json:"description"`
		Amount      float64 `json:"amount"`
	} `json:"transactions"`
}

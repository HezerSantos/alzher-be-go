package models

type CallResult struct {
	Source string  `json:"source"`
	Result *string `json:"result"`
	Status int     `json:"status"`
	Error  *string `json:"error"`
}

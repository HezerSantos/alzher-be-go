package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Format string `json:"format"`
	Stream bool   `json:"stream"`
}

type OllamaResponse struct {
	Response string `json:"response"`
}

type Transaction struct {
	PostDate    string  `json:"postDate"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
}

func AskOllama(transactionString string) ([]Transaction, error) {
	var OLLAMA_URL = os.Getenv("OLLAMA_URL")

	if OLLAMA_URL == "" {
		return nil, fmt.Errorf("Ollama URL Not Configured")
	}

	promptWithTransactions := promptHelper
	promptWithTransactions += transactionString
	schema := map[string]any{
		"type": "array",
		"items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"postDate": map[string]any{
					"type": "string",
				},
				"description": map[string]any{
					"type": "string",
				},
				"amount": map[string]any{
					"type": "number",
				},
			},
			"required": []string{
				"postDate",
				"description",
				"amount",
			},
		},
	}

	jsonSchema, err := json.Marshal(schema)

	if err != nil {
		return nil, err
	}

	body := OllamaRequest{
		Model:  "llama3.2",
		Prompt: promptWithTransactions,
		Format: string(jsonSchema),
		Stream: false,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	res, err := http.Post(
		OLLAMA_URL,
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var result OllamaResponse
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	var transactions []Transaction

	err = json.Unmarshal([]byte(result.Response), &transactions)

	if err != nil {
		return nil, err
	}

	return transactions, nil
}

package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/HezerSantos/alzher-api/common/api"
	"github.com/HezerSantos/alzher-api/common/api/types"
	"github.com/HezerSantos/alzher-api/services/common/ai"
)

type OllamaRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	Format  any            `json:"format"`
	Stream  bool           `json:"stream"`
	Options map[string]any `json:"options"`
}

type OllamaResponse struct {
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	DoneReason         string `json:"done_reason"`
	EvalCount          int    `json:"eval_count"`
	EvalDuration       int64  `json:"eval_duration"`
	PromptEvalCount    int    `json:"prompt_eval_count"`
	PromptEvalDuration int64  `json:"prompt_eval_duration"`
}

func AskOllama(transactionString string) (*ai.Transaction, []types.CallResult) {
	var OLLAMA_URL = os.Getenv("OLLAMA_URL")
	var callResults []types.CallResult

	if OLLAMA_URL == "" {
		api.MakeCallResults(&callResults, "Ollama: os.Getenv()", nil, http.StatusInternalServerError, fmt.Errorf("Ollama URL Not Configured"))

		return nil, callResults
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"transactions": map[string]any{
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
					"required": []any{
						"postDate",
						"description",
						"amount",
					},
				},
			},
		},
		"required": []any{
			"transactions",
		},
	}

	body := OllamaRequest{
		Model:  "qwen2.5:1.5b",
		Prompt: ai.ReturnFinancialPromptInstructions(transactionString),
		Format: schema,
		Stream: false,
		Options: map[string]any{
			"num_thread":  8,
			"num_batch":   2048,
			"temperature": 0,
			"num_ctx":     4096,
		},
	}

	data, err := json.Marshal(body)
	if err != nil {
		api.MakeCallResults(&callResults, "Ollama: json.Marshal()", nil, http.StatusInternalServerError, err)
		return nil, callResults
	}

	res, err := http.Post(
		OLLAMA_URL,
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		api.MakeCallResults(&callResults, "Ollama: http.Post()", nil, http.StatusInternalServerError, err)
		return nil, callResults
	}
	defer res.Body.Close()

	var result OllamaResponse
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		api.MakeCallResults(&callResults, "Ollama: json.NewDecoder().Decode()", nil, http.StatusInternalServerError, fmt.Errorf("Error decoding ollama response: %w", err))
		return nil, callResults
	}

	api.MakeCallResults(&callResults, "Ollama: json.NewDecoder().Decode()", result, http.StatusOK, nil)

	var transactions ai.Transaction

	err = json.Unmarshal([]byte(result.Response), &transactions)

	if err != nil {
		api.MakeCallResults(&callResults, "Ollama: json.NewDecoder().Decode()", nil, http.StatusInternalServerError, fmt.Errorf("Error Unmarshalling ollama response: %w", err))
		return nil, callResults
	}

	return &transactions, callResults
}

package groq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/HezerSantos/alzher-api/common/api"
	"github.com/HezerSantos/alzher-api/services/common/ai"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func AskGroq(ctx context.Context, crc *api.CallResultContainer, input string) (*ai.Transaction, error) {
	GROQ_API_KEY := os.Getenv("GROQ_API_KEY")
	GROQ_URL := os.Getenv("GROQ_URL")

	if GROQ_API_KEY == "" {
		crc.Add("Groq: os.Getenv()", nil, http.StatusInternalServerError, fmt.Errorf("GROQ API KEY Not Configure"))
		return nil, fmt.Errorf("GROQ API KEY Not Configure")
	}
	if GROQ_URL == "" {
		crc.Add("Groq: os.Getenv()", nil, http.StatusInternalServerError, fmt.Errorf("GROQ URL Not Configure"))
		return nil, fmt.Errorf("GROQ URL Not Configure")
	}

	client := openai.NewClient(
		option.WithAPIKey(GROQ_API_KEY),
		option.WithBaseURL(GROQ_URL),
	)

	prompt := ai.ReturnFinancialPromptInstructions(input)
	resp, err := client.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Model: "openai/gpt-oss-20b",
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(prompt),
			},
			Temperature: openai.Float(0.0),
			MaxTokens:   openai.Int(4096),
		},
		option.WithJSONSet("reasoning_effort", "low"),
	)

	if err != nil {
		var apiErr *openai.Error
		if errors.As(err, &apiErr) {
			crc.Add("Groq: New()", nil, apiErr.StatusCode, err)
			return nil, err
		}
		crc.Add("Groq: New()", nil, http.StatusInternalServerError, err)

		return nil, err
	}

	crc.Add("Groq: New()", resp, http.StatusOK, nil)

	if len(resp.Choices) == 0 {
		return nil, nil
	}

	var transactions ai.Transaction

	if err := json.Unmarshal(
		[]byte(resp.Choices[0].Message.Content),
		&transactions,
	); err != nil {
		crc.Add("Groq: json.Unmarshal()", nil, http.StatusInternalServerError, err)
		return nil, err
	}

	return &transactions, nil
}

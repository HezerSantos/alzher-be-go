package api

import (
	"encoding/json"
	"net/http"

	"github.com/HezerSantos/alzher-api/common/api/types"
)

func MakeJson(jsonSource any, source string) (*string, *types.CallResult) {
	jsonBytes, err := json.Marshal(jsonSource)
	if err != nil {
		errorString := err.Error()
		callResult := types.CallResult{
			Source: source,
			Result: nil,
			Status: http.StatusInternalServerError,
			Error:  &errorString,
		}
		return nil, &callResult
	}

	jsonString := string(jsonBytes)

	return &jsonString, nil

}

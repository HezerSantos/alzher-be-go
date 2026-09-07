package api

import (
	"encoding/json"
	"net/http"

	"github.com/HezerSantos/alzher-api/common/api/models"
)

func MakeJson(jsonSource any, source string) (*string, *models.CallResult) {
	jsonBytes, err := json.Marshal(jsonSource)
	if err != nil {
		errorString := err.Error()
		callResult := models.CallResult{
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

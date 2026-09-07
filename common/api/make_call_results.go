package api

import (
	"github.com/HezerSantos/alzher-api/common/api/models"
)

// This function takes an empty call results slice and appends a call result struct based on parameters
func MakeCallResults(callResults *[]models.CallResult, source string, result any, status int, err error) {
	if err != nil {
		errorString := err.Error()
		callResult := models.CallResult{
			Source: source,
			Result: nil,
			Status: status,
			Error:  &errorString,
		}
		*callResults = append(*callResults, callResult)
	} else {
		jsonString, cr := MakeJson(result, source)

		if cr != nil {
			*callResults = append(*callResults, *cr)
		}

		callResult := models.CallResult{
			Source: source,
			Result: jsonString,
			Status: status,
			Error:  nil,
		}
		*callResults = append(*callResults, callResult)
	}
}

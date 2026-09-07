package api

import "github.com/HezerSantos/alzher-api/common/api/types"

// This function takes an empty call results slice and appends a call result struct based on parameters
func MakeCallResults(callResults *[]types.CallResult, source string, result any, status int, err error) {
	if err != nil {
		errorString := err.Error()
		callResult := types.CallResult{
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

		callResult := types.CallResult{
			Source: source,
			Result: jsonString,
			Status: status,
			Error:  nil,
		}
		*callResults = append(*callResults, callResult)
	}
}

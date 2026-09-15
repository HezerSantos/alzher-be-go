package api

import (
	"context"
	"fmt"
	"sync"

	"github.com/HezerSantos/alzher-api/common/api/types"
)

type CallResultContextType string

var CallResultContextKey CallResultContextType

type CallResultContainer struct {
	CallResults []types.CallResult
	mu          sync.Mutex
}

func (crc *CallResultContainer) HasError() bool {
	for _, cr := range crc.CallResults {
		if cr.Error != nil {
			return true
		}
	}
	return false
}

func (crc *CallResultContainer) Add(source string, result any, status int, err error) {
	crc.mu.Lock()
	MakeCallResults(&crc.CallResults, source, result, status, err)
	crc.mu.Unlock()
}

func GetCallResultContainerContext(ctx context.Context) (*CallResultContainer, error) {
	crc := ctx.Value(CallResultContextKey)

	if parsecCrc, ok := crc.(*CallResultContainer); ok != true {
		return nil, fmt.Errorf("CallResultContainer Not Defined")
	} else {
		return parsecCrc, nil
	}
}

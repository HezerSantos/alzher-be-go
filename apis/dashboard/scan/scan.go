package scan

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"

	"github.com/HezerSantos/alzher-api/common/api"
	"github.com/HezerSantos/alzher-api/common/api/types"
	"github.com/HezerSantos/alzher-api/services/ollama"
	"github.com/gin-gonic/gin"
	"github.com/ledongthuc/pdf"
)

func processFile(file *multipart.FileHeader) (*ollama.Transaction, []types.CallResult) {
	var callResults []types.CallResult

	f, e := file.Open()
	if e != nil {
		api.MakeCallResults(&callResults, "Ollama:  file.Open()", nil, http.StatusInternalServerError, e)

		return nil, callResults
	}
	defer f.Close()

	p, err := pdf.NewReader(f, file.Size)

	if err != nil {
		api.MakeCallResults(&callResults, "Ollama:  pdf.NewReader()", nil, http.StatusInternalServerError, e)

		return nil, callResults
	}

	var text strings.Builder

	for pageNum := 1; pageNum <= p.NumPage(); pageNum++ {
		page := p.Page(pageNum)

		pageText, err := page.GetPlainText(nil)

		if err != nil {
			api.MakeCallResults(&callResults, "Ollama:  page.GetPlainText()", nil, http.StatusInternalServerError, e)

			return nil, callResults
		}
		text.WriteString(pageText)
	}

	normalized := ollama.NormalizeStatementText(text.String())
	transactions, callResultsResponse := ollama.AskOllama(normalized)

	callResults = append(callResults, callResultsResponse...)
	for _, err := range callResultsResponse {
		if err.Error != nil {
			return nil, callResults
		}
	}

	return transactions, callResults
}

func PostDashboardDocument(ginCtx *gin.Context) {
	// user, err := userinfo.GetUserContext(ginCtx.Request.Context())

	// if err != nil {
	// 	errorfuncs.UnauthorizedError(ginCtx)
	// 	return
	// }

	form, err := ginCtx.MultipartForm()

	if err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	files := form.File["files"]

	var callResults []types.CallResult
	var transactions []ollama.Transaction

	var mu sync.Mutex

	var wg sync.WaitGroup
	for _, file := range files {
		wg.Add(1)
		go func(file *multipart.FileHeader) {
			defer wg.Done()

			transactionResult, callResultsResponse := processFile(file)

			mu.Lock()
			defer mu.Unlock()
			for _, err := range callResultsResponse {
				if err.Error != nil {
					callResults = append(callResults, callResultsResponse...)
					return
				}
			}
			api.MakeCallResults(&callResults, fmt.Sprintf("Ollama: File-%s", file.Filename), transactionResult, http.StatusOK, nil)
			transactions = append(transactions, *transactionResult)

		}(file)
	}

	wg.Wait()

	for _, cr := range callResults {
		if cr.Error != nil {
			ginCtx.JSON(http.StatusInternalServerError, gin.H{
				"callResults": callResults,
			})
			return
		}
	}

	ginCtx.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		"callResults":  callResults,
	})
}

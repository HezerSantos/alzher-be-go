package scan

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"

	"github.com/HezerSantos/alzher-api/common/api"
	"github.com/HezerSantos/alzher-api/common/api/types"
	"github.com/HezerSantos/alzher-api/services/common/ai"
	"github.com/HezerSantos/alzher-api/services/groq"
	"github.com/HezerSantos/alzher-api/services/ollama"
	"github.com/gen2brain/go-fitz"
	"github.com/gin-gonic/gin"
)

func processFile(file *multipart.FileHeader) (*ai.Transaction, []types.CallResult) {
	var callResults []types.CallResult

	f, err := file.Open()
	if err != nil {
		api.MakeCallResults(&callResults, "Groq: file.Open()", nil, http.StatusInternalServerError, err)
		return nil, callResults
	}
	defer f.Close()

	contents, err := io.ReadAll(f)
	if err != nil {
		api.MakeCallResults(&callResults, "Groq: io.ReadAll()", nil, http.StatusInternalServerError, err)
		return nil, callResults
	}

	// Uses MuPDF engine in-memory — handles Chase, Capital One, and encrypted streams without panicking
	doc, err := fitz.NewFromMemory(contents)
	if err != nil {
		api.MakeCallResults(&callResults, "Groq: fitz.NewFromMemory()", nil, http.StatusInternalServerError, err)
		return nil, callResults
	}
	defer doc.Close()

	var textBuilder strings.Builder
	for n := 0; n < doc.NumPage(); n++ {
		pageText, err := doc.Text(n)
		if err != nil {
			api.MakeCallResults(&callResults, "Groq: doc.Text()", nil, http.StatusInternalServerError, err)
			return nil, callResults
		}
		textBuilder.WriteString(pageText)
	}

	normalized := ollama.NormalizeStatementText(textBuilder.String())
	transactions, err, callResultsResponse := groq.AskGroq(normalized)

	if err != nil {
		api.MakeCallResults(&callResults, "Groq: processFile()", nil, http.StatusInternalServerError, err)
		return nil, callResults
	}

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
	var transactions []ai.Transaction

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
			if transactionResult != nil {
				api.MakeCallResults(&callResults, fmt.Sprintf("Groq: File-%s", file.Filename), transactionResult, http.StatusOK, nil)
				transactions = append(transactions, *transactionResult)
			} else {
				// Record an error if transactionResult is nil despite no explicit call error
				api.MakeCallResults(&callResults, fmt.Sprintf("Groq: File-%s", file.Filename), nil, http.StatusInternalServerError, fmt.Errorf("transaction extraction returned nil result"))
			}

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

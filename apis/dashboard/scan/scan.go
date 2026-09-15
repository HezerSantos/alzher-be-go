package scan

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"

	"github.com/HezerSantos/alzher-api/common/api"
	"github.com/HezerSantos/alzher-api/common/errorfuncs"
	"github.com/HezerSantos/alzher-api/services/common/ai"
	"github.com/HezerSantos/alzher-api/services/groq"
	"github.com/HezerSantos/alzher-api/services/ollama"
	"github.com/gen2brain/go-fitz"
	"github.com/gin-gonic/gin"
)

// func hash(data []byte) string {
// 	hash := sha256.Sum256(data)
// 	return hex.EncodeToString(hash[:])
// }

func processFile(crc *api.CallResultContainer, file *multipart.FileHeader) (*ai.Transaction, error) {

	f, err := file.Open()
	if err != nil {
		crc.Add("processFile(): file.Open()", nil, http.StatusInternalServerError, err)
		return nil, err
	}
	defer f.Close()

	contents, err := io.ReadAll(f)
	if err != nil {
		crc.Add("processFile(): io.ReadAll()", nil, http.StatusInternalServerError, err)

		return nil, err
	}

	// Uses MuPDF engine in-memory — handles Chase, Capital One, and encrypted streams without panicking
	doc, err := fitz.NewFromMemory(contents)
	if err != nil {
		crc.Add("processFile(): fitz.NewFromMemory()", nil, http.StatusInternalServerError, err)

		return nil, err
	}
	defer doc.Close()

	var textBuilder strings.Builder
	for n := 0; n < doc.NumPage(); n++ {
		pageText, err := doc.Text(n)
		if err != nil {
			crc.Add("processFile(): doc.Text()", nil, http.StatusInternalServerError, err)

			return nil, err
		}
		textBuilder.WriteString(pageText)
	}

	normalized := ollama.NormalizeStatementText(textBuilder.String())
	transactions, err := groq.AskGroq(crc, normalized)

	if err != nil {
		crc.Add("processFile(): processFile()", nil, http.StatusInternalServerError, err)
		return nil, err
	}

	return transactions, nil
}

func PostDashboardDocument(ginCtx *gin.Context) {

	crc, err := api.GetCallResultContainerContext(ginCtx.Request.Context())

	if err != nil {
		errorfuncs.NetworkError(ginCtx, err)
		return
	}
	form, err := ginCtx.MultipartForm()

	if err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	files := form.File["files"]

	var transactions []ai.Transaction

	var mu sync.Mutex

	var wg sync.WaitGroup
	for _, file := range files {
		wg.Add(1)
		go func(file *multipart.FileHeader) {
			defer wg.Done()

			transactionResult, err := processFile(crc, file)

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				return
			}
			if transactionResult != nil {
				crc.Add(fmt.Sprintf("Groq: File-%s", file.Filename), transactionResult, http.StatusOK, nil)
				transactions = append(transactions, *transactionResult)
			} else {
				crc.Add(fmt.Sprintf("Groq: File-%s", file.Filename), nil, http.StatusInternalServerError, fmt.Errorf("transaction extraction returned nil result"))
			}

		}(file)
	}

	wg.Wait()

	if crc.HasError() {
		ginCtx.JSON(http.StatusInternalServerError, gin.H{
			"callResults": crc.CallResults,
		})
		return
	}

	ginCtx.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		"callResults":  crc.CallResults,
	})
}

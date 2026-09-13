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

func processFile(i int, file *multipart.FileHeader) ([]ollama.Transaction, error) {
	f, e := file.Open()
	if e != nil {
		return nil, e
	}
	defer f.Close()

	p, err := pdf.NewReader(f, file.Size)

	if err != nil {
		return nil, err
	}

	var text strings.Builder

	for pageNum := 1; pageNum <= p.NumPage(); pageNum++ {
		page := p.Page(pageNum)

		pageText, err := page.GetPlainText(nil)

		if err != nil {
			return nil, err
		}
		text.WriteString(pageText)
	}

	transactions, err := ollama.AskOllama(text.String())

	if err != nil {
		return nil, err
	}

	return transactions, nil
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
	for i, file := range files {
		wg.Add(1)
		go func(i int, file *multipart.FileHeader) {
			defer wg.Done()

			transactionResult, err := processFile(i, file)

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				api.MakeCallResults(&callResults, fmt.Sprintf("Ollama: File-%s", file.Filename), nil, http.StatusInternalServerError, err)
				return
			}
			api.MakeCallResults(&callResults, fmt.Sprintf("Ollama: File-%s", file.Filename), transactionResult, http.StatusInternalServerError, nil)
			transactions = append(transactions, transactionResult...)

		}(i, file)
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
}

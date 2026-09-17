package scan

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"

	"github.com/HezerSantos/alzher-api/common/api"
	"github.com/HezerSantos/alzher-api/common/errorfuncs"
	"github.com/HezerSantos/alzher-api/common/userinfo"
	"github.com/HezerSantos/alzher-api/services/common/ai"
	"github.com/HezerSantos/alzher-api/services/groq"
	"github.com/HezerSantos/alzher-api/services/ollama"
	"github.com/HezerSantos/alzher-api/services/railway"
	"github.com/HezerSantos/alzher-api/services/railway/models"
	"github.com/gen2brain/go-fitz"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	transactionCache   = map[string][]ai.Transaction{}
	transactionCacheMu sync.Mutex
)

func hash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func queryStatementHash(userId uuid.UUID, fileHash string) (*models.Statements, error) {
	var statement *models.Statements

	err := railway.DB.Model(&models.Statements{}).
		Where(`"userId" = ?`, userId).
		Where(`"statementId" = ?`, fileHash).
		First(&statement).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return statement, nil
}

func processFile(ctx context.Context, cancel context.CancelFunc, crc *api.CallResultContainer, file *multipart.FileHeader, userId uuid.UUID) ([]ai.Transaction, *string, error) {

	f, err := file.Open()
	if err != nil {
		crc.Add("processFile(): file.Open()", nil, http.StatusInternalServerError, err)
		return nil, nil, err
	}
	defer f.Close()

	contents, err := io.ReadAll(f)
	if err != nil {
		crc.Add("processFile(): io.ReadAll()", nil, http.StatusInternalServerError, err)

		return nil, nil, err
	}

	fileHash := hash(contents)

	transactionCacheMu.Lock()

	if cacheHit, ok := transactionCache[fileHash]; ok == true {
		transactionCacheMu.Unlock()
		crc.Add("processFile(): Cache Hit", cacheHit, http.StatusOK, nil)
		return cacheHit, &fileHash, nil
	}
	transactionCacheMu.Unlock()

	statement, err := queryStatementHash(userId, fileHash)

	if err != nil {
		crc.Add("processFile(): queryStatementHash()", nil, http.StatusInternalServerError, err)
		return nil, nil, err
	}

	crc.Add("processFile(): queryStatementHash()", statement, http.StatusOK, nil)

	if statement != nil {
		cancel()
		crc.Add("processFile(): queryStatementHash()", statement, http.StatusConflict, fmt.Errorf("Statement Already Exists"))
		crc.SetStatus(http.StatusConflict)
		return nil, nil, fmt.Errorf("Statement Already Exists")
	}

	// Uses MuPDF engine in-memory — handles Chase, Capital One, and encrypted streams without panicking
	doc, err := fitz.NewFromMemory(contents)
	if err != nil {
		crc.Add("processFile(): fitz.NewFromMemory()", nil, http.StatusInternalServerError, err)

		return nil, nil, err
	}
	defer doc.Close()

	var textBuilder strings.Builder
	for n := 0; n < doc.NumPage(); n++ {
		pageText, err := doc.Text(n)
		if err != nil {
			crc.Add("processFile(): doc.Text()", nil, http.StatusInternalServerError, err)

			return nil, nil, err
		}
		textBuilder.WriteString(pageText)
	}

	normalized := ollama.NormalizeStatementText(textBuilder.String())
	fmt.Println(normalized)
	return nil, nil, nil
	transactions, err := groq.AskGroq(ctx, crc, normalized)

	if err != nil {
		crc.Add("processFile(): processFile()", nil, http.StatusInternalServerError, err)
		return nil, nil, err
	}

	if transactions == nil {
		return nil, nil, nil
	}

	transactionCacheMu.Lock()
	transactionCache[fileHash] = transactions
	transactionCacheMu.Unlock()

	return transactions, &fileHash, nil
}

func PostDashboardDocument(ginCtx *gin.Context) {

	crc, err := api.GetCallResultContainerContext(ginCtx.Request.Context())

	if err != nil {
		errorfuncs.NetworkError(ginCtx, err)
		return
	}

	user, err := userinfo.GetUserContext(ginCtx.Request.Context())

	if err != nil {
		errorfuncs.UnauthorizedError(ginCtx)
	}

	form, err := ginCtx.MultipartForm()

	if err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	files := form.File["files"]

	var transactions = make(map[string][]ai.Transaction)

	var mu sync.Mutex

	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, file := range files {
		wg.Add(1)
		go func(file *multipart.FileHeader) {
			select {
			case <-ctx.Done():
				return
			default:
			}
			defer wg.Done()

			transactionResult, fileHash, err := processFile(ctx, cancel, crc, file, user.ID)

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				return
			}
			if transactionResult != nil && fileHash != nil {
				crc.Add(fmt.Sprintf("Groq: File-%s", file.Filename), transactionResult, http.StatusOK, nil)
				transactions[*fileHash] = transactionResult
			} else {
				crc.Add(fmt.Sprintf("Groq: File-%s", file.Filename), nil, http.StatusInternalServerError, fmt.Errorf("transaction extraction returned nil result"))
			}

		}(file)
	}

	wg.Wait()

	if crc.HasError() {
		ginCtx.JSON(crc.Status(), gin.H{
			"callResults": crc.CallResults,
		})
		return
	}

	// var predictedTransactions []alzherml.Transaction
	// var successHashes []string
	// for h, t := range transactions {
	// 	wg.Add(1)
	// 	go func(hash string, t []ai.Transaction) {
	// 		defer wg.Done()
	// 		predicted, err := alzherml.FetchTransactionCategories(ctx, t, crc)

	// 		if err != nil {
	// 			crc.Add("AlzherML: FetchTransactionCategories()", nil, http.StatusInternalServerError, err)
	// 			return
	// 		}

	// 		crc.Add("AlzherML: FetchTransactionCategories()", predicted, http.StatusOK, nil)
	// 		mu.Lock()
	// 		predictedTransactions = append(predictedTransactions, predicted...)
	// 		successHashes = append(successHashes, hash)
	// 		mu.Unlock()
	// 	}(h, t)
	// }

	// wg.Wait()

	//TODO: Add query to post success hashes
	//TOOD: Add query to post success transactions

	ginCtx.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		// "predictedTransactions": predictedTransactions,
		"callResults": crc.CallResults,
	})
}

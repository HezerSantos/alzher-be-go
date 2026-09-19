package errorfuncs

import (
	"fmt"
	"net/http"

	"github.com/HezerSantos/alzher-api/common/api"
	"github.com/gin-gonic/gin"
)

type JsonResponseType struct {
	Msg  string
	Code string
}

type JsonError struct {
	Message string
	Status  int
	Json    JsonResponseType
}

func ErrorHelper(c *gin.Context, j JsonError) {
	fmt.Printf("\t%s\n", j.Message)
	c.JSON(j.Status, gin.H{
		"msg":  j.Json.Msg,
		"code": j.Json.Code,
	})
}

func NetworkError(ginCtx *gin.Context, err error, crc *api.CallResultContainer) {
	if err != nil {
		ginCtx.JSON(http.StatusInternalServerError, gin.H{
			"message": "Internal Server Error",
			"error":   err.Error(),
		})
	} else {
		ginCtx.JSON(http.StatusInternalServerError, gin.H{
			"message":     "Internal Server Error",
			"callResults": crc.CallResults,
		})
	}
}

func UnauthorizedError(ginCtx *gin.Context) {
	ginCtx.JSON(http.StatusInternalServerError, gin.H{
		"message": "Unauthorized",
	})
}

func BadRequestError(ginCtx *gin.Context, crc *api.CallResultContainer) {
	ginCtx.JSON(http.StatusInternalServerError, gin.H{
		"message":     "Bad Request",
		"callResults": crc.CallResults,
	})
}

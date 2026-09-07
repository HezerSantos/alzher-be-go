package errorfuncs

import (
	"fmt"

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

func NetworkError(c *gin.Context, error error) {
	fmt.Printf("\t%s\n\t%s", "Internal Server Error", error)
	c.JSON(500, gin.H{
		"msg":  "Internal Server Error",
		"code": "INVALID_SERVER",
	})
}

func UnauthorizedError(c *gin.Context) {
	ErrorHelper(c,
		JsonError{
			Message: "Unauthorized",
			Status:  401,
			Json:    JsonResponseType{Code: "INVALID_ACCESS", Msg: "Unauthorized"},
		},
	)
}

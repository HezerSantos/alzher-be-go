package main

import (
	"regexp"

	"github.com/HezerSantos/alzher-api/services/railway"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
)

var dateRegex = regexp.MustCompile(
	`^(?:[1-9]|1[0-2])/(?:[1-9]|[12][0-9]|3[01])/[0-9]{4}$`,
)

func validateDate(fl validator.FieldLevel) bool {
	value := fl.Field().String()

	if value == "" {
		return true
	}

	return dateRegex.MatchString(value)
}

func main() {
	godotenv.Load()

	err := railway.ConnectDatabase()

	if err != nil {
		panic(err.Error())
	}

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("dateformat", validateDate)
	}

	r := gin.Default()

	api := r.Group("/api")
	ConnectRouter(api)

	r.Run(":8080")
}

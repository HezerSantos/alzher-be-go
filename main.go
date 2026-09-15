package main

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/HezerSantos/alzher-api/middleware"
	"github.com/HezerSantos/alzher-api/services/railway"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
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

	client := openai.NewClient(
		option.WithAPIKey(os.Getenv("GROQ_API_KEY")),
		option.WithBaseURL(os.Getenv("GROQ_URL")),
	)

	models, err := client.Models.List(context.Background())
	if err != nil {
		fmt.Printf("Error fetching models: %v\n", err)
		return
	}

	fmt.Println("Available models for your API key:")
	for _, model := range models.Data {
		fmt.Println("-", model.ID)
	}
	err = railway.ConnectDatabase()

	if err != nil {
		panic(err.Error())
	}

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("dateformat", validateDate)
	}

	r := gin.Default()

	api := r.Group("/api", middleware.AttatchContext())
	ConnectRouter(api)

	r.Run(":8080")
}

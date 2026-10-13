package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/HezerSantos/alzher-api/common/cors"
	"github.com/HezerSantos/alzher-api/middleware"
	"github.com/HezerSantos/alzher-api/services/railway"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

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

	r := gin.Default()

	api := r.Group("/api", middleware.AttatchContext())
	ConnectRouter(api)

	if err := http.ListenAndServe(":8080", cors.CorsHandler.Handler(r)); err != nil {
		panic(err)
	}
}

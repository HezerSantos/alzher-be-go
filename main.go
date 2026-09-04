package main

import (
	"github.com/HezerSantos/alzher-api/services/railway"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	err := railway.ConnectDatabase()

	if err != nil {
		panic(err.Error())
	}

	r := gin.Default()

	api := r.Group("/api")
	ConnectRouter(api)

	r.Run(":8080")
}

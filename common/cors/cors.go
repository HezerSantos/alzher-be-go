package cors

import (
	"net/http"

	"github.com/rs/cors"
)

var CorsHandler = cors.New(cors.Options{
	AllowedOrigins: []string{
		"http://localhost:5173",
	},
	AllowedMethods: []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
	},
	AllowedHeaders: []string{
		"Origin",
		"Content-Type",
		"Authorization",
	},
	AllowCredentials: true,
})

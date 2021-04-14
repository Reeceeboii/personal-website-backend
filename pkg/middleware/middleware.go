package middleware

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// Simply output a logging of each incoming request
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		log.Println("-> INCOMING", request.Method, "REQUEST TO", request.RequestURI)
		// Call the next handler
		next.ServeHTTP(writer, request)
	})
}

// Enable CORS headers on all responses
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// Insert CORS header into ResponseWriter
		writer.Header().Set("Access-Control-Allow-Origin", "*")
		// Call the next handler
		next.ServeHTTP(writer, request)
	})
}

// Handle cache control headers
func CacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// Insert cache control header into ResponseWriter
		writer.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", int(time.Hour.Seconds())))
		// Call the next handler
		next.ServeHTTP(writer, request)
	})
}

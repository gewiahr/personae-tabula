package apiHttp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func HttpWrapper(f APIEndpointFunc) APIHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		response := f(ctx, r)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(response.Code)

		var err error
		if response.Error != nil {
			err = json.NewEncoder(w).Encode(response.Error)
		} else if response.Payload != nil {
			err = json.NewEncoder(w).Encode(response.Payload)
		}

		if err != nil {
			err = json.NewEncoder(w).Encode(&APIError{
				Error:   err.Error(),
				Message: fmt.Sprintf("Error parsing %s %s response", r.Method, r.URL.Path),
			})
		}

		log.Printf("[%s %s] %s", r.Method, r.URL.Path, r.Pattern)
	}
}

// Middleware для логирования
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//start := time.Now()

		// Логируем запрос
		//log.Printf("[%s] %s %s", r.Method, r.URL.Path, time.Since(start))

		// Добавляем заголовки CORS для разработки
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

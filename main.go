package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type URLPair struct {
	Original  string `json:"original"`
	ShortCode string `json:"short_code"`
}

var urlMap = make(map[string]string)
var logger *zap.Logger

// loggingMiddleware logs the start and end of each request
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		logger.Info("Request started",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
		)

		next(w, r)

		duration := time.Since(start)
		logger.Info("Request finished",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Duration("duration", duration),
		)
	}
}

func main() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	http.HandleFunc("/shorten", loggingMiddleware(shortenHandler))
	http.HandleFunc("/", loggingMiddleware(redirectHandler))

	logger.Info("Server starting", zap.String("address", "http://localhost:8080"))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Fatal("Server failed to start", zap.Error(err))
	}
}

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var urlPair URLPair
	if err := json.NewDecoder(r.Body).Decode(&urlPair); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	shortCode := generateShortCode()
	urlMap[shortCode] = urlPair.Original

	response := map[string]string{
		"short_code": shortCode,
		"short_url":  "http://localhost:8080/" + shortCode,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	shortCode := r.URL.Path[1:]

	originalURL, exists := urlMap[shortCode]
	if !exists {
		http.Error(w, "Short code not found", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
}

// generateShortCode generates a random short code for the URL
// it uses a combination of lowercase and uppercase letters and numbers
// and returns a 6 character string
func generateShortCode() string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	shortCode := make([]byte, 6)
	for i := range shortCode {
		shortCode[i] = chars[rand.Intn(len(chars))]
	}
	return string(shortCode)
}
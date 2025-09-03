package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Kairum-Labs/should"
)

func TestGenerateShortCode(t *testing.T) {
	t.Run("should generate 6 character code", func(t *testing.T) {
		code := generateShortCode()
		should.BeEqual(t, len(code), 6, should.WithMessage("Short code should be exactly 6 characters"))
	})

	t.Run("should generate alphanumeric characters", func(t *testing.T) {
		code := generateShortCode()
		validChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		
		for _, char := range code {
			should.ContainSubstring(t, validChars, string(char), should.WithMessage("Code should only contain alphanumeric characters"))
		}
	})

	t.Run("should generate different codes on multiple calls", func(t *testing.T) {
		code1 := generateShortCode()
		code2 := generateShortCode()
		code3 := generateShortCode()
		
		should.NotBeEqual(t, code1, code2, should.WithMessage("Consecutive codes should be different"))
		should.NotBeEqual(t, code2, code3, should.WithMessage("Consecutive codes should be different"))
		should.NotBeEqual(t, code1, code3, should.WithMessage("Non-consecutive codes should be different"))
	})
}

func TestShortenHandler(t *testing.T) {
	t.Run("should return method not allowed for non-POST requests", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shorten", nil)
		w := httptest.NewRecorder()
		
		shortenHandler(w, req)
		
		should.BeEqual(t, w.Code, http.StatusMethodNotAllowed, should.WithMessage("Should return 405 for non-POST requests"))
		should.BeEqual(t, strings.TrimSpace(w.Body.String()), "Method not allowed")
	})

	t.Run("should return bad request for invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader("invalid json"))
		w := httptest.NewRecorder()
		
		shortenHandler(w, req)
		
		should.BeEqual(t, w.Code, http.StatusBadRequest, should.WithMessage("Should return 400 for invalid JSON"))
		should.BeEqual(t, strings.TrimSpace(w.Body.String()), "Invalid request body")
	})

	t.Run("should create short URL successfully", func(t *testing.T) {
		// Clear the urlMap for clean test
		urlMap = make(map[string]string)
		
		urlPair := URLPair{Original: "https://example.com/very/long/url"}
		jsonData, _ := json.Marshal(urlPair)
		
		req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBuffer(jsonData))
		w := httptest.NewRecorder()
		
		shortenHandler(w, req)
		
		should.BeEqual(t, w.Code, http.StatusOK, should.WithMessage("Should return 200 for successful creation"))
		should.BeEqual(t, w.Header().Get("Content-Type"), "application/json", should.WithMessage("Should set correct content type"))
		
		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		should.BeNil(t, err, should.WithMessage("Response should be valid JSON"))
		
		should.ContainKey(t, response, "short_code", should.WithMessage("Response should contain short_code"))
		should.ContainKey(t, response, "short_url", should.WithMessage("Response should contain short_url"))
		should.BeEqual(t, len(response["short_code"]), 6, should.WithMessage("Short code should be 6 characters"))
		should.StartsWith(t, response["short_url"], "http://localhost:8080/", should.WithMessage("Short URL should start with localhost"))
		should.EndsWith(t, response["short_url"], response["short_code"], should.WithMessage("Short URL should end with short code"))
	})

	t.Run("should store URL in map", func(t *testing.T) {
		// Clear the urlMap for clean test
		urlMap = make(map[string]string)
		
		originalURL := "https://google.com"
		urlPair := URLPair{Original: originalURL}
		jsonData, _ := json.Marshal(urlPair)
		
		req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBuffer(jsonData))
		w := httptest.NewRecorder()
		
		shortenHandler(w, req)
		
		var response map[string]string
		json.Unmarshal(w.Body.Bytes(), &response)
		
		shortCode := response["short_code"]
		should.ContainKey(t, urlMap, shortCode, should.WithMessage("URL should be stored in map"))
		should.BeEqual(t, urlMap[shortCode], originalURL, should.WithMessage("Stored URL should match original"))
	})
}

func TestRedirectHandler(t *testing.T) {
	t.Run("should return not found for non-existent short code", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
		w := httptest.NewRecorder()
		
		redirectHandler(w, req)
		
		should.BeEqual(t, w.Code, http.StatusNotFound, should.WithMessage("Should return 404 for non-existent code"))
		should.BeEqual(t, strings.TrimSpace(w.Body.String()), "Short code not found")
	})

	t.Run("should redirect to original URL for valid short code", func(t *testing.T) {
		// Clear and populate urlMap for test
		urlMap = make(map[string]string)
		shortCode := "abc123"
		originalURL := "https://example.com"
		urlMap[shortCode] = originalURL
		
		req := httptest.NewRequest(http.MethodGet, "/"+shortCode, nil)
		w := httptest.NewRecorder()
		
		redirectHandler(w, req)
		
		should.BeEqual(t, w.Code, http.StatusTemporaryRedirect, should.WithMessage("Should return 307 for redirect"))
		should.BeEqual(t, w.Header().Get("Location"), originalURL, should.WithMessage("Should redirect to original URL"))
	})

	t.Run("should handle root path correctly", func(t *testing.T) {
		// Clear and populate urlMap for test
		urlMap = make(map[string]string)
		shortCode := "xyz789"
		originalURL := "https://google.com"
		urlMap[shortCode] = originalURL
		
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		
		redirectHandler(w, req)
		
		should.BeEqual(t, w.Code, http.StatusNotFound, should.WithMessage("Root path should return 404"))
	})
}

func TestURLPairStruct(t *testing.T) {
	t.Run("should marshal and unmarshal correctly", func(t *testing.T) {
		original := URLPair{
			Original:  "https://example.com",
			ShortCode: "abc123",
		}
		
		jsonData, err := json.Marshal(original)
		should.BeNil(t, err, should.WithMessage("Should marshal without error"))
		
		var unmarshaled URLPair
		err = json.Unmarshal(jsonData, &unmarshaled)
		should.BeNil(t, err, should.WithMessage("Should unmarshal without error"))
		
		should.BeEqual(t, unmarshaled.Original, original.Original, should.WithMessage("Original URL should match"))
		should.BeEqual(t, unmarshaled.ShortCode, original.ShortCode, should.WithMessage("Short code should match"))
	})
}

func TestIntegration(t *testing.T) {
	t.Run("should create and redirect successfully", func(t *testing.T) {
		// Clear the urlMap for clean test
		urlMap = make(map[string]string)
		
		// Step 1: Create short URL
		originalURL := "https://github.com"
		urlPair := URLPair{Original: originalURL}
		jsonData, _ := json.Marshal(urlPair)
		
		req1 := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBuffer(jsonData))
		w1 := httptest.NewRecorder()
		shortenHandler(w1, req1)
		
		should.BeEqual(t, w1.Code, http.StatusOK, should.WithMessage("Shorten should succeed"))
		
		var response map[string]string
		json.Unmarshal(w1.Body.Bytes(), &response)
		shortCode := response["short_code"]
		
		should.NotBeEmpty(t, shortCode, should.WithMessage("Short code should not be empty"))
		should.ContainKey(t, urlMap, shortCode, should.WithMessage("URL should be stored in map"))
		
		// Step 2: Test redirect
		req2 := httptest.NewRequest(http.MethodGet, "/"+shortCode, nil)
		w2 := httptest.NewRecorder()
		redirectHandler(w2, req2)
		
		should.BeEqual(t, w2.Code, http.StatusTemporaryRedirect, should.WithMessage("Redirect should succeed"))
		should.BeEqual(t, w2.Header().Get("Location"), originalURL, should.WithMessage("Should redirect to original URL"))
	})
} 
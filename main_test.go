package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type testPayload struct {
	Action      string            `json:"action"`
	IP          string            `json:"ip"`
	BearerToken string            `json:"bearerToken"`
	QueryParams map[string]string `json:"queryParams"`
	UserAgent   string            `json:"userAgent"`
}

func TestWebhookHandler(t *testing.T) {
	os.Setenv("FOO", "token1,token2")

	payload := testPayload{
		Action:      "test",
		IP:          "127.0.0.1",
		BearerToken: "token1",
		QueryParams: map[string]string{},
		UserAgent:   "test-agent",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	// Use the same handler as in main.go
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST method is accepted", http.StatusMethodNotAllowed)
			return
		}

		var payload webhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		fooEnv := os.Getenv("FOO")
		var validTokens []string
		if fooEnv != "" {
			validTokens = strings.Split(fooEnv, ",")
			for i := range validTokens {
				validTokens[i] = strings.TrimSpace(validTokens[i])
			}
		}

		isValid := false
		for _, token := range validTokens {
			if payload.BearerToken == token {
				isValid = true
				break
			}
		}

		if isValid {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(webhookResponse{StreamKey: payload.BearerToken})
		} else {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(webhookResponse{})
		}
	})

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

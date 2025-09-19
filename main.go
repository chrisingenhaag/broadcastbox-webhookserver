package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
)

type webhookPayload struct {
	Action      string            `json:"action"`
	IP          string            `json:"ip"`
	BearerToken string            `json:"bearerToken"`
	QueryParams map[string]string `json:"queryParams"`
	UserAgent   string            `json:"userAgent"`
}

type webhookResponse struct {
	StreamKey string `json:"streamKey"`
}

func main() {
	// Read WEBHOOK_ENABLED_STREAMKEYS from environment and split by comma
	enabledStreamKeys := os.Getenv("WEBHOOK_ENABLED_STREAMKEYS")
	var validTokens []string
	if enabledStreamKeys != "" {
		validTokens = strings.Split(enabledStreamKeys, ",")
		for i := range validTokens {
			validTokens[i] = strings.TrimSpace(validTokens[i])
		}
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST method is accepted", http.StatusMethodNotAllowed)
			return
		}

		var payload webhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Check if payload.BearerToken is in validTokens
		isValid := false
		for _, token := range validTokens {
			if payload.BearerToken == token {
				isValid = true
				break
			}
		}

		if isValid {
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(webhookResponse{StreamKey: payload.BearerToken}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		} else {
			log.Printf("Invalid token attempt from IP %s with token %s\n", payload.IP, payload.BearerToken)
			w.WriteHeader(http.StatusForbidden)
			if err := json.NewEncoder(w).Encode(webhookResponse{}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	})

	log.Println("Server listening on port 8000")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}

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

func parseStreamKeys(env string) map[string]string {
	m := make(map[string]string)
	// Read WEBHOOK_ENABLED_STREAMKEYS from environment and parse as bearerToken:streamKey tuples
	enabledStreamKeys := env
	if enabledStreamKeys != "" {
		pairs := strings.Split(enabledStreamKeys, ",")
		for _, pair := range pairs {
			pair = strings.TrimSpace(pair)
			if pair == "" {
				continue
			}
			parts := strings.SplitN(pair, ":", 2)
			if len(parts) == 2 {
				bearer := strings.TrimSpace(parts[0])
				streamKey := strings.TrimSpace(parts[1])
				m[bearer] = streamKey
			}
		}
	}
	return m
}

func webhookHandler(tokenToStreamKey map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST method is accepted", http.StatusMethodNotAllowed)
			return
		}

		var payload webhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		switch payload.Action {
		case "whip-connect":
			streamKey, isValid := tokenToStreamKey[payload.BearerToken]
			if isValid {
				w.WriteHeader(http.StatusOK)
				if err := json.NewEncoder(w).Encode(webhookResponse{StreamKey: streamKey}); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			} else {
				log.Printf("Invalid whip-connect token attempt from IP %s with bearerToken %s\n", payload.IP, payload.BearerToken)
				w.WriteHeader(http.StatusForbidden)
				if err := json.NewEncoder(w).Encode(webhookResponse{}); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			}
		case "whep-connect":
			found := false
			for _, v := range tokenToStreamKey {
				if v == payload.BearerToken {
					found = true
					break
				}
			}
			if found {
				w.WriteHeader(http.StatusOK)
				if err := json.NewEncoder(w).Encode(webhookResponse{StreamKey: payload.BearerToken}); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				return
			}
			log.Printf("Invalid whep-connect token attempt from IP %s with streamKey %s\n", payload.IP, payload.BearerToken)
			w.WriteHeader(http.StatusForbidden)
			if err := json.NewEncoder(w).Encode(webhookResponse{}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		default:
			http.Error(w, "Invalid action", http.StatusBadRequest)
		}
	}
}

func main() {
	tokenToStreamKey := parseStreamKeys(os.Getenv("WEBHOOK_ENABLED_STREAMKEYS"))
	http.HandleFunc("/", webhookHandler(tokenToStreamKey))
	log.Println("Server listening on port 8000")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}

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
	// Read WEBHOOK_ENABLED_STREAMKEYS from environment and parse as bearerToken:streamKey tuples
	enabledStreamKeys := os.Getenv("WEBHOOK_ENABLED_STREAMKEYS")
	tokenToStreamKey := make(map[string]string)
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
				tokenToStreamKey[bearer] = streamKey
			}
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

		switch payload.Action {
		case "whip-connect":
			// Standard logic: BearerToken must be a key in the map
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
			// Accept if BearerToken is a value in the map (streamKey)
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
	})

	log.Println("Server listening on port 8000")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}

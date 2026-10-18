package main

import (
	"encoding/json"
	"errors"
	"fmt"
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

func parseStreamKeys(env string) (map[string]string, error) {
	m := make(map[string]string)
	// Parse comma-separated bearerToken:streamKey tuples
	for _, pair := range strings.Split(env, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		bearer, streamKey, ok := strings.Cut(pair, ":")
		bearer, streamKey = strings.TrimSpace(bearer), strings.TrimSpace(streamKey)
		if !ok || bearer == "" || streamKey == "" {
			return nil, fmt.Errorf("invalid pair %q: want bearerToken:streamKey", pair)
		}
		m[bearer] = streamKey
	}
	if len(m) == 0 {
		return nil, errors.New("no bearerToken:streamKey pairs configured")
	}
	return m, nil
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
	tokenToStreamKey, err := parseStreamKeys(os.Getenv("WEBHOOK_ENABLED_STREAMKEYS"))
	if err != nil {
		log.Fatalf("Invalid WEBHOOK_ENABLED_STREAMKEYS: %s\n", err)
	}
	http.HandleFunc("/", webhookHandler(tokenToStreamKey))
	log.Println("Server listening on port 8000")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}

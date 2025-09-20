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
	os.Setenv("WEBHOOK_ENABLED_STREAMKEYS", "token1:streamkey1,token2:streamkey2")

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

		switch payload.Action {
		case "whip-connect":
			streamKey, isValid := tokenToStreamKey[payload.BearerToken]
			if isValid {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(webhookResponse{StreamKey: streamKey})
			} else {
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(webhookResponse{})
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
				json.NewEncoder(w).Encode(webhookResponse{StreamKey: payload.BearerToken})
			} else {
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(webhookResponse{})
			}
		default:
			http.Error(w, "Invalid action", http.StatusBadRequest)
		}
	})

	// whip-connect: BearerToken as key
	payload := testPayload{
		Action:      "whip-connect",
		IP:          "127.0.0.1",
		BearerToken: "token1",
		QueryParams: map[string]string{},
		UserAgent:   "test-agent",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("whip-connect: expected status 200, got %d", rec.Code)
	}
	var resp webhookResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("whip-connect: failed to decode response: %v", err)
	}
	if resp.StreamKey != "streamkey1" {
		t.Errorf("whip-connect: expected streamKey 'streamkey1', got '%s'", resp.StreamKey)
	}

	// whep-connect: BearerToken as value (streamKey)
	payload = testPayload{
		Action:      "whep-connect",
		IP:          "127.0.0.1",
		BearerToken: "streamkey2",
		QueryParams: map[string]string{},
		UserAgent:   "test-agent",
	}
	body, _ = json.Marshal(payload)
	req = httptest.NewRequest("POST", "/", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("whep-connect: expected status 200, got %d", rec.Code)
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("whep-connect: failed to decode response: %v", err)
	}
	if resp.StreamKey != "streamkey2" {
		t.Errorf("whep-connect: expected streamKey 'streamkey2', got '%s'", resp.StreamKey)
	}

	// whep-connect: BearerToken not found
	payload = testPayload{
		Action:      "whep-connect",
		IP:          "127.0.0.1",
		BearerToken: "notfound",
		QueryParams: map[string]string{},
		UserAgent:   "test-agent",
	}
	body, _ = json.Marshal(payload)
	req = httptest.NewRequest("POST", "/", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("whep-connect (notfound): expected status 403, got %d", rec.Code)
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("whep-connect (notfound): failed to decode response: %v", err)
	}
	if resp.StreamKey != "" {
		t.Errorf("whep-connect (notfound): expected empty streamKey, got '%s'", resp.StreamKey)
	}
}

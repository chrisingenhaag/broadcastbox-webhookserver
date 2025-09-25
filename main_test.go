package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	tokenToStreamKey := parseStreamKeys("token1:streamkey1,token2:streamkey2")
	handler := webhookHandler(tokenToStreamKey)

	tests := []struct {
		name          string
		payload       testPayload
		wantStatus    int
		wantStreamKey string
	}{
		{
			name: "whip-connect: valid token",
			payload: testPayload{
				Action:      "whip-connect",
				IP:          "127.0.0.1",
				BearerToken: "token1",
				QueryParams: map[string]string{},
				UserAgent:   "test-agent",
			},
			wantStatus:    http.StatusOK,
			wantStreamKey: "streamkey1",
		},
		{
			name: "whep-connect: valid streamkey",
			payload: testPayload{
				Action:      "whep-connect",
				IP:          "127.0.0.1",
				BearerToken: "streamkey2",
				QueryParams: map[string]string{},
				UserAgent:   "test-agent",
			},
			wantStatus:    http.StatusOK,
			wantStreamKey: "streamkey2",
		},
		{
			name: "whip-connect: not found",
			payload: testPayload{
				Action:      "whip-connect",
				IP:          "127.0.0.1",
				BearerToken: "notfound",
				QueryParams: map[string]string{},
				UserAgent:   "test-agent",
			},
			wantStatus:    http.StatusForbidden,
			wantStreamKey: "",
		},
		{
			name: "whep-connect: not found",
			payload: testPayload{
				Action:      "whep-connect",
				IP:          "127.0.0.1",
				BearerToken: "notfound",
				QueryParams: map[string]string{},
				UserAgent:   "test-agent",
			},
			wantStatus:    http.StatusForbidden,
			wantStreamKey: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.payload)
			req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}
			var resp webhookResponse
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if resp.StreamKey != tc.wantStreamKey {
				t.Errorf("expected streamKey '%s', got '%s'", tc.wantStreamKey, resp.StreamKey)
			}
		})
	}
}

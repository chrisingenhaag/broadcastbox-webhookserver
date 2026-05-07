package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
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
		{
			// Public stream names must not be usable as ingest tokens.
			name: "whip-connect: streamKey rejected as bearerToken",
			payload: testPayload{
				Action:      "whip-connect",
				IP:          "127.0.0.1",
				BearerToken: "streamkey1",
				QueryParams: map[string]string{},
				UserAgent:   "test-agent",
			},
			wantStatus:    http.StatusForbidden,
			wantStreamKey: "",
		},
		{
			// Secret ingest tokens must not authenticate viewers.
			name: "whep-connect: ingest token rejected as bearerToken",
			payload: testPayload{
				Action:      "whep-connect",
				IP:          "127.0.0.1",
				BearerToken: "token1",
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

	t.Run("non-POST method rejected", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
		}
	})

	t.Run("invalid JSON rejected", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", strings.NewReader("not json"))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("unknown action rejected", func(t *testing.T) {
		body, _ := json.Marshal(testPayload{
			Action:      "bogus-action",
			BearerToken: "token1",
		})
		req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}

func TestParseStreamKeys(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want map[string]string
	}{
		{
			name: "empty input",
			in:   "",
			want: map[string]string{},
		},
		{
			name: "single pair",
			in:   "token1:streamkey1",
			want: map[string]string{"token1": "streamkey1"},
		},
		{
			name: "multiple pairs",
			in:   "token1:streamkey1,token2:streamkey2",
			want: map[string]string{"token1": "streamkey1", "token2": "streamkey2"},
		},
		{
			name: "whitespace around pairs is trimmed",
			in:   "  token1 : streamkey1 , token2:streamkey2  ",
			want: map[string]string{"token1": "streamkey1", "token2": "streamkey2"},
		},
		{
			name: "trailing comma is ignored",
			in:   "token1:streamkey1,",
			want: map[string]string{"token1": "streamkey1"},
		},
		{
			name: "pair without colon is skipped",
			in:   "token1:streamkey1,malformed",
			want: map[string]string{"token1": "streamkey1"},
		},
		{
			// SplitN(_, ":", 2) keeps anything past the first colon in the value.
			name: "second colon kept in stream key",
			in:   "token1:stream:key:1",
			want: map[string]string{"token1": "stream:key:1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseStreamKeys(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("parseStreamKeys(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

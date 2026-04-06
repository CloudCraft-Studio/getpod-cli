package jira

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetup_ValidConfig(t *testing.T) {
	p := &Plugin{}
	cfg := map[string]string{
		"base_url":  "https://example.atlassian.net",
		"email":     "test@example.com",
		"api_token": "test-token",
	}

	if err := p.Setup(cfg); err != nil {
		t.Fatalf("Setup() failed: %v", err)
	}

	if p.client == nil {
		t.Fatal("client not initialized")
	}

	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("test@example.com:test-token"))
	if p.client.authHeader != expectedAuth {
		t.Errorf("authHeader mismatch: got %q, want %q", p.client.authHeader, expectedAuth)
	}
}

func TestSetup_MissingFields(t *testing.T) {
	tests := []struct {
		name string
		cfg  map[string]string
		want string
	}{
		{
			name: "missing base_url",
			cfg:  map[string]string{"email": "test@example.com", "api_token": "token"},
			want: "base_url",
		},
		{
			name: "missing email",
			cfg:  map[string]string{"base_url": "https://example.atlassian.net", "api_token": "token"},
			want: "email",
		},
		{
			name: "missing api_token",
			cfg:  map[string]string{"base_url": "https://example.atlassian.net", "email": "test@example.com"},
			want: "api_token",
		},
		{
			name: "empty config",
			cfg:  map[string]string{},
			want: "base_url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Plugin{}
			err := p.Setup(tt.cfg)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestValidate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/myself" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth == "" {
			t.Error("missing Authorization header")
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(JiraUser{
			AccountID:   "test-account-id",
			Email:       "test@example.com",
			DisplayName: "Test User",
		}); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	p := &Plugin{}
	cfg := map[string]string{
		"base_url":  server.URL,
		"email":     "test@example.com",
		"api_token": "test-token",
	}

	if err := p.Setup(cfg); err != nil {
		t.Fatalf("Setup() failed: %v", err)
	}

	if err := p.Validate(); err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}
}

func TestValidate_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		if err := json.NewEncoder(w).Encode(JiraError{
			ErrorMessages: []string{"Invalid credentials"},
		}); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	p := &Plugin{}
	cfg := map[string]string{
		"base_url":  server.URL,
		"email":     "test@example.com",
		"api_token": "invalid-token",
	}

	if err := p.Setup(cfg); err != nil {
		t.Fatalf("Setup() failed: %v", err)
	}

	err := p.Validate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestValidate_NotInitialized(t *testing.T) {
	p := &Plugin{}
	err := p.Validate()
	if err == nil {
		t.Fatal("expected error for uninitialized plugin")
	}
}

func TestClient_HandleErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    string
	}{
		{
			name:       "401 unauthorized",
			statusCode: http.StatusUnauthorized,
			wantErr:    "invalid credentials",
		},
		{
			name:       "403 forbidden",
			statusCode: http.StatusForbidden,
			wantErr:    "insufficient permissions",
		},
		{
			name:       "404 not found",
			statusCode: http.StatusNotFound,
			wantErr:    "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewClient(server.URL, "test@example.com", "token")
			var out any
			err := client.get(context.Background(), "/test", &out)

			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

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

func TestFetchIssues_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !containsString(r.URL.Path, "/rest/api/3/search") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := JiraSearchResponse{
			Issues: []JiraIssue{
				{
					Key:  "LULO-1234",
					Self: r.Host + "/rest/api/3/issue/12345",
					Fields: JiraIssueFields{
						Summary: "Test issue",
						Status: JiraStatus{
							Name: "In Progress",
						},
						Priority: JiraPriority{
							Name: "High",
						},
						Updated: "2026-04-05T10:00:00Z",
					},
				},
			},
			Total:      1,
			StartAt:    0,
			MaxResults: 50,
		}

		if err := json.NewEncoder(w).Encode(resp); err != nil {
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

	issues, err := p.FetchIssues(context.Background(), IssueFilter{
		AssignedToMe: true,
	})

	if err != nil {
		t.Fatalf("FetchIssues() failed: %v", err)
	}

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]
	if issue.Key != "LULO-1234" {
		t.Errorf("Key mismatch: got %q, want %q", issue.Key, "LULO-1234")
	}
	if issue.Title != "Test issue" {
		t.Errorf("Title mismatch: got %q, want %q", issue.Title, "Test issue")
	}
	if issue.Status != "in-progress" {
		t.Errorf("Status mismatch: got %q, want %q", issue.Status, "in-progress")
	}
	if issue.Priority != "high" {
		t.Errorf("Priority mismatch: got %q, want %q", issue.Priority, "high")
	}
	if issue.Source != "jira" {
		t.Errorf("Source mismatch: got %q, want %q", issue.Source, "jira")
	}
}

func TestFetchIssues_EmptyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := JiraSearchResponse{
			Issues:     []JiraIssue{},
			Total:      0,
			StartAt:    0,
			MaxResults: 50,
		}

		if err := json.NewEncoder(w).Encode(resp); err != nil {
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

	issues, err := p.FetchIssues(context.Background(), IssueFilter{
		AssignedToMe: true,
	})

	if err != nil {
		t.Fatalf("FetchIssues() failed: %v", err)
	}

	if len(issues) != 0 {
		t.Fatalf("expected 0 issues, got %d", len(issues))
	}
}

func TestFetchIssues_NetworkError(t *testing.T) {
	p := &Plugin{}
	cfg := map[string]string{
		"base_url":  "http://localhost:99999",
		"email":     "test@example.com",
		"api_token": "test-token",
	}

	if err := p.Setup(cfg); err != nil {
		t.Fatalf("Setup() failed: %v", err)
	}

	_, err := p.FetchIssues(context.Background(), IssueFilter{
		AssignedToMe: true,
	})

	if err == nil {
		t.Fatal("expected error for network failure, got nil")
	}
}

func TestNormalizeStatus(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Open", "todo"},
		{"To Do", "todo"},
		{"Backlog", "todo"},
		{"In Progress", "in-progress"},
		{"In Development", "in-progress"},
		{"Code Review", "in-review"},
		{"In Review", "in-review"},
		{"PR Open", "in-review"},
		{"Done", "done"},
		{"Closed", "done"},
		{"Resolved", "done"},
		{"Blocked", "blocked"},
		{"Custom State", "todo"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeStatus(tt.input)
			if got != tt.want {
				t.Errorf("normalizeStatus(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizePriority(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Highest", "urgent"},
		{"Blocker", "urgent"},
		{"High", "high"},
		{"Medium", "medium"},
		{"Low", "low"},
		{"Lowest", "low"},
		{"Custom Priority", "medium"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizePriority(tt.input)
			if got != tt.want {
				t.Errorf("normalizePriority(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsString(s[1:], substr)))
}

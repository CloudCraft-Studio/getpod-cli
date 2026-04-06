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

func TestGetIssue_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !containsString(r.URL.Path, "/rest/api/3/issue/LULO-1234") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := JiraIssueDetail{
			Key: "LULO-1234",
			Fields: JiraIssueDetailFields{
				Summary: "Test issue with details",
				Status: JiraStatus{
					Name: "In Progress",
				},
				Priority: JiraPriority{
					Name: "High",
				},
				Updated:     "2026-04-05T10:00:00Z",
				Description: map[string]any{"type": "doc", "content": []any{map[string]any{"type": "paragraph", "content": []any{map[string]any{"type": "text", "text": "Test description"}}}}},
				Comment: struct {
					Comments []JiraComment `json:"comments"`
				}{
					Comments: []JiraComment{
						{
							Author: struct {
								DisplayName string `json:"displayName"`
							}{DisplayName: "John Doe"},
							Body:    map[string]any{"type": "doc", "content": []any{map[string]any{"type": "paragraph", "content": []any{map[string]any{"type": "text", "text": "First comment"}}}}},
							Created: "2026-04-05T09:00:00Z",
						},
					},
				},
				Subtasks: []any{map[string]any{}, map[string]any{}},
				Labels:   []string{"backend", "api"},
			},
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

	detail, err := p.GetIssue(context.Background(), "LULO-1234")
	if err != nil {
		t.Fatalf("GetIssue() failed: %v", err)
	}

	if detail.Key != "LULO-1234" {
		t.Errorf("Key mismatch: got %q, want %q", detail.Key, "LULO-1234")
	}
	if detail.Description != "Test description" {
		t.Errorf("Description mismatch: got %q, want %q", detail.Description, "Test description")
	}
	if len(detail.Comments) != 1 {
		t.Errorf("Comments count mismatch: got %d, want 1", len(detail.Comments))
	}
	if detail.Subtasks != 2 {
		t.Errorf("Subtasks count mismatch: got %d, want 2", detail.Subtasks)
	}
	if len(detail.Labels) != 2 {
		t.Errorf("Labels count mismatch: got %d, want 2", len(detail.Labels))
	}
}

func TestGetIssue_MalformedADF(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := JiraIssueDetail{
			Key: "LULO-999",
			Fields: JiraIssueDetailFields{
				Summary: "Issue with malformed ADF",
				Status: JiraStatus{
					Name: "Open",
				},
				Priority: JiraPriority{
					Name: "Medium",
				},
				Updated:     "2026-04-05T10:00:00Z",
				Description: "not a valid ADF object",
				Comment: struct {
					Comments []JiraComment `json:"comments"`
				}{
					Comments: []JiraComment{},
				},
				Subtasks: []any{},
				Labels:   []string{},
			},
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

	detail, err := p.GetIssue(context.Background(), "LULO-999")
	if err != nil {
		t.Fatalf("GetIssue() should not fail on malformed ADF: %v", err)
	}

	if detail.Description != "" {
		t.Errorf("Expected empty description for malformed ADF, got: %q", detail.Description)
	}
}

func TestGetTransitions_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !containsString(r.URL.Path, "/rest/api/3/issue/LULO-1234/transitions") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := JiraTransitionsResponse{
			Transitions: []JiraTransition{
				{
					ID:   "31",
					Name: "In Progress",
					To: JiraStatus{
						Name: "In Progress",
					},
				},
				{
					ID:   "41",
					Name: "Done",
					To: JiraStatus{
						Name: "Done",
					},
				},
			},
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

	transitions, err := p.GetTransitions(context.Background(), "LULO-1234")
	if err != nil {
		t.Fatalf("GetTransitions() failed: %v", err)
	}

	if len(transitions) != 2 {
		t.Fatalf("expected 2 transitions, got %d", len(transitions))
	}

	if transitions[0].ID != "31" {
		t.Errorf("Transition ID mismatch: got %q, want %q", transitions[0].ID, "31")
	}
	if transitions[0].To != "in-progress" {
		t.Errorf("Transition To mismatch: got %q, want %q", transitions[0].To, "in-progress")
	}
}

func TestTransitionIssue_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !containsString(r.URL.Path, "/rest/api/3/issue/LULO-1234/transitions") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Validar body
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}

		transition, ok := body["transition"].(map[string]any)
		if !ok || transition["id"] != "31" {
			t.Errorf("invalid body: %+v", body)
		}

		// HTTP 204 No Content
		w.WriteHeader(http.StatusNoContent)
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

	err := p.TransitionIssue(context.Background(), "LULO-1234", "31")
	if err != nil {
		t.Fatalf("TransitionIssue() failed: %v", err)
	}
}

func TestTransitionIssue_InvalidID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		resp := JiraError{
			ErrorMessages: []string{"Transition ID not found"},
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

	err := p.TransitionIssue(context.Background(), "LULO-1234", "999")
	if err == nil {
		t.Fatal("expected error for invalid transition ID, got nil")
	}
}

func TestExtractPlainText(t *testing.T) {
	tests := []struct {
		name string
		adf  any
		want string
	}{
		{
			name: "simple paragraph",
			adf: map[string]any{
				"type": "doc",
				"content": []any{
					map[string]any{
						"type": "paragraph",
						"content": []any{
							map[string]any{"type": "text", "text": "Hello world"},
						},
					},
				},
			},
			want: "Hello world",
		},
		{
			name: "multiple paragraphs",
			adf: map[string]any{
				"type": "doc",
				"content": []any{
					map[string]any{
						"type": "paragraph",
						"content": []any{
							map[string]any{"type": "text", "text": "First paragraph"},
						},
					},
					map[string]any{
						"type": "paragraph",
						"content": []any{
							map[string]any{"type": "text", "text": "Second paragraph"},
						},
					},
				},
			},
			want: "First paragraph\nSecond paragraph",
		},
		{
			name: "nil ADF",
			adf:  nil,
			want: "",
		},
		{
			name: "invalid ADF",
			adf:  "not a map",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPlainText(tt.adf)
			if got != tt.want {
				t.Errorf("extractPlainText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsString(s[1:], substr)))
}

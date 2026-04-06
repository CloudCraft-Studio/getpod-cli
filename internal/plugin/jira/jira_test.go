package jira

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CloudCraft-Studio/getpod-cli/internal/plugin"
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

func TestGetIssueDetail_Success(t *testing.T) {
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

	detail, err := p.GetIssueDetail(context.Background(), "LULO-1234")
	if err != nil {
		t.Fatalf("GetIssueDetail() failed: %v", err)
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

func TestGetIssueDetail_MalformedADF(t *testing.T) {
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

	detail, err := p.GetIssueDetail(context.Background(), "LULO-999")
	if err != nil {
		t.Fatalf("GetIssueDetail() should not fail on malformed ADF: %v", err)
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

// TestAddComment_Success tests adding a comment without context
func TestAddComment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !containsString(r.URL.Path, "/rest/api/3/issue/LULO-1234/comment") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Validate request body contains ADF
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}

		bodyField, ok := body["body"].(map[string]any)
		if !ok {
			t.Errorf("expected body field to be a map, got %T", body["body"])
		}

		// Validate ADF structure
		if bodyField["type"] != "doc" {
			t.Errorf("expected type 'doc', got %v", bodyField["type"])
		}
		if bodyField["version"] != float64(1) {
			t.Errorf("expected version 1, got %v", bodyField["version"])
		}

		// HTTP 201 Created
		w.WriteHeader(http.StatusCreated)
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

	err := p.AddComment(context.Background(), "LULO-1234", "This is a test comment", nil)
	if err != nil {
		t.Fatalf("AddComment() failed: %v", err)
	}
}

// TestAddComment_WithContext tests adding a comment with GetPod context
func TestAddComment_WithContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !containsString(r.URL.Path, "/rest/api/3/issue/LULO-5678/comment") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Validate request body contains ADF with context
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}

		bodyADF, ok := body["body"].(map[string]any)
		if !ok {
			t.Errorf("expected body field to be a map")
		}

		// Get the text content from ADF
		content, ok := bodyADF["content"].([]any)
		if !ok || len(content) == 0 {
			t.Errorf("expected content array in ADF")
			return
		}

		paragraph, ok := content[0].(map[string]any)
		if !ok {
			t.Errorf("expected first content item to be a paragraph")
			return
		}

		paragraphContent, ok := paragraph["content"].([]any)
		if !ok || len(paragraphContent) == 0 {
			t.Errorf("expected paragraph content")
			return
		}

		textNode, ok := paragraphContent[0].(map[string]any)
		if !ok {
			t.Errorf("expected text node")
			return
		}

		text, ok := textNode["text"].(string)
		if !ok {
			t.Errorf("expected text field in node")
			return
		}

		// Validate that context is included in the text
		if !containsString(text, "GetPod context") {
			t.Errorf("expected 'GetPod context' in comment text, got: %s", text)
		}
		if !containsString(text, "Workspace: core-services") {
			t.Errorf("expected 'Workspace: core-services' in comment, got: %s", text)
		}
		if !containsString(text, "Env: qa") {
			t.Errorf("expected 'Env: qa' in comment, got: %s", text)
		}

		// HTTP 201 Created
		w.WriteHeader(http.StatusCreated)
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

	gpCtx := &plugin.CommentContext{
		Workspace:   "core-services",
		Environment: "qa",
		Branch:      "feature/lulo-5678",
		Repos:       []string{"backend-core", "infra-terraform"},
	}

	err := p.AddComment(context.Background(), "LULO-5678", "Test comment with context", gpCtx)
	if err != nil {
		t.Fatalf("AddComment() failed: %v", err)
	}
}

// TestAddComment_IssueNotFound tests error handling for non-existent issue
func TestAddComment_IssueNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		resp := JiraError{
			ErrorMessages: []string{"Issue not found"},
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

	err := p.AddComment(context.Background(), "NONEXISTENT-999", "This should fail", nil)
	if err == nil {
		t.Fatal("expected error for non-existent issue, got nil")
	}

	if !containsString(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

// TestBuildADFComment tests the ADF comment builder
func TestBuildADFComment(t *testing.T) {
	adf := buildADFComment("Hello world")

	// Validate structure
	if adf["type"] != "doc" {
		t.Errorf("expected type 'doc', got %v", adf["type"])
	}
	if adf["version"] != 1 {
		t.Errorf("expected version 1, got %v", adf["version"])
	}

	// Validate content
	content, ok := adf["content"].([]map[string]any)
	if !ok || len(content) == 0 {
		t.Fatal("expected non-empty content array")
	}

	paragraph := content[0]
	if paragraph["type"] != "paragraph" {
		t.Errorf("expected first item to be a paragraph")
	}

	paragraphContent, ok := paragraph["content"].([]map[string]any)
	if !ok || len(paragraphContent) == 0 {
		t.Fatal("expected paragraph content")
	}

	textNode := paragraphContent[0]
	if textNode["type"] != "text" {
		t.Errorf("expected text node type")
	}
	if textNode["text"] != "Hello world" {
		t.Errorf("expected text 'Hello world', got %v", textNode["text"])
	}
}

// TestBuildCommentBody tests the comment body builder with context
func TestBuildCommentBody(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		gpCtx   *plugin.CommentContext
		wantCtx bool
	}{
		{
			name:    "no context",
			body:    "Just a comment",
			gpCtx:   nil,
			wantCtx: false,
		},
		{
			name: "with context",
			body: "Working on this",
			gpCtx: &plugin.CommentContext{
				Workspace:   "prod",
				Environment: "staging",
				Repos:       []string{"api", "web"},
			},
			wantCtx: true,
		},
		{
			name:    "empty context",
			body:    "Another comment",
			gpCtx:   &plugin.CommentContext{},
			wantCtx: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildCommentBody(tt.body, tt.gpCtx)

			if !containsString(result, tt.body) {
				t.Errorf("expected comment body to contain '%s', got '%s'", tt.body, result)
			}

			if tt.wantCtx {
				if !containsString(result, "GetPod context") {
					t.Errorf("expected GetPod context block in result")
				}
				if !containsString(result, "Workspace:") {
					t.Errorf("expected Workspace in context")
				}
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsString(s[1:], substr)))
}

// TestChangeStatus_Success tests successfully changing issue status
func TestChangeStatus_Success(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// First call: GetTransitions
		if callCount == 1 && containsString(r.URL.Path, "/transitions") && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			resp := JiraTransitionsResponse{
				Transitions: []JiraTransition{
					{
						ID:   "21",
						Name: "To Do",
						To: JiraStatus{
							Name: "To Do",
						},
					},
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
			return
		}

		// Second call: TransitionIssue
		if callCount == 2 && containsString(r.URL.Path, "/transitions") && r.Method == http.MethodPost {
			// Validate body
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("failed to decode body: %v", err)
			}

			transition, ok := body["transition"].(map[string]any)
			if !ok || transition["id"] != "31" {
				t.Errorf("expected transition ID '31' in body, got: %+v", body)
			}

			// HTTP 204 No Content
			w.WriteHeader(http.StatusNoContent)
			return
		}

		t.Errorf("unexpected call %d: %s %s", callCount, r.Method, r.URL.Path)
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

	err := p.ChangeStatus(context.Background(), "LULO-1234", "in-progress")
	if err != nil {
		t.Fatalf("ChangeStatus() failed: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
}

// TestChangeStatus_NoMatchingTransition tests error when no transition matches desired status
func TestChangeStatus_NoMatchingTransition(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if containsString(r.URL.Path, "/transitions") && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			resp := JiraTransitionsResponse{
				Transitions: []JiraTransition{
					{
						ID:   "21",
						Name: "To Do",
						To: JiraStatus{
							Name: "To Do",
						},
					},
					{
						ID:   "31",
						Name: "In Progress",
						To: JiraStatus{
							Name: "In Progress",
						},
					},
				},
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Errorf("failed to encode response: %v", err)
			}
			return
		}

		t.Errorf("unexpected call: %s %s", r.Method, r.URL.Path)
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

	// Try to change to "done" which is not in available transitions
	err := p.ChangeStatus(context.Background(), "LULO-1234", "done")
	if err == nil {
		t.Fatal("expected error for unavailable status, got nil")
	}

	if !containsString(err.Error(), "no transition found") {
		t.Errorf("expected 'no transition found' error, got: %v", err)
	}
}

// TestChangeStatus_GetTransitionsError tests error handling when GetTransitions fails
func TestChangeStatus_GetTransitionsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		resp := JiraError{
			ErrorMessages: []string{"Issue not found"},
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

	err := p.ChangeStatus(context.Background(), "NONEXISTENT-999", "in-progress")
	if err == nil {
		t.Fatal("expected error when GetTransitions fails, got nil")
	}

	if !containsString(err.Error(), "getting transitions") {
		t.Errorf("expected 'getting transitions' error, got: %v", err)
	}
}

// TestAvailableStatusNames tests the helper function
func TestAvailableStatusNames(t *testing.T) {
	transitions := []Transition{
		{ID: "21", Name: "To Do", To: "todo"},
		{ID: "31", Name: "In Progress", To: "in-progress"},
		{ID: "41", Name: "Done", To: "done"},
	}

	names := availableStatusNames(transitions)

	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}

	expected := []string{"todo", "in-progress", "done"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("names[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

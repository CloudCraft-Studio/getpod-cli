package jira

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	authHeader string
	httpClient *http.Client
}

func NewClient(baseURL, email, apiToken string) *Client {
	encoded := base64.StdEncoding.EncodeToString([]byte(email + ":" + apiToken))
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		authHeader: "Basic " + encoded,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("making request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return c.handleError(resp.StatusCode, body)
	}

	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}
	}

	return nil
}

func (c *Client) handleError(statusCode int, body []byte) error {
	var jiraErr JiraError
	if err := json.Unmarshal(body, &jiraErr); err == nil {
		if len(jiraErr.ErrorMessages) > 0 {
			return fmt.Errorf("jira error: %s", strings.Join(jiraErr.ErrorMessages, "; "))
		}
	}

	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("invalid credentials: authentication failed")
	case http.StatusForbidden:
		return fmt.Errorf("insufficient permissions: access denied")
	case http.StatusNotFound:
		return fmt.Errorf("resource not found")
	default:
		return fmt.Errorf("HTTP %d: %s", statusCode, string(body))
	}
}

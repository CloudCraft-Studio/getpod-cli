package jira

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/CloudCraft-Studio/getpod-cli/internal/plugin"
)

type Plugin struct {
	client *Client
}

// IssueFilter define criterios para filtrar issues en Jira
type IssueFilter struct {
	AssignedToMe   bool
	StatusCategory string // "in-progress", "todo", "done"
	MaxResults     int    // default 50
}

func (p *Plugin) Name() string    { return "jira" }
func (p *Plugin) Version() string { return "0.1.0" }

func (p *Plugin) Setup(cfg map[string]string) error {
	baseURL, ok := cfg["base_url"]
	if !ok || baseURL == "" {
		return fmt.Errorf("missing required config: base_url")
	}

	email, ok := cfg["email"]
	if !ok || email == "" {
		return fmt.Errorf("missing required config: email")
	}

	apiToken, ok := cfg["api_token"]
	if !ok || apiToken == "" {
		return fmt.Errorf("missing required config: api_token")
	}

	p.client = NewClient(baseURL, email, apiToken)
	return nil
}

func (p *Plugin) Validate() error {
	if p.client == nil {
		return fmt.Errorf("plugin not initialized: call Setup() first")
	}

	ctx := context.Background()
	var user JiraUser
	if err := p.client.get(ctx, "/rest/api/3/myself", &user); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) CollectMetrics(ctx context.Context, since time.Time) ([]plugin.Metric, error) {
	return nil, nil
}

func (p *Plugin) DeriveSkills(ctx context.Context) ([]plugin.Skill, error) {
	return nil, nil
}

func (p *Plugin) Commands() []*cobra.Command {
	return []*cobra.Command{
		issuesCmd(p),
	}
}

// FetchIssues consulta issues desde Jira con filtros y los normaliza
func (p *Plugin) FetchIssues(ctx context.Context, filter IssueFilter) ([]plugin.NormalizedIssue, error) {
	if p.client == nil {
		return nil, fmt.Errorf("plugin not initialized: call Setup() first")
	}

	// Construir JQL
	jql := buildJQL(filter)

	// Preparar parámetros de query
	maxResults := filter.MaxResults
	if maxResults == 0 {
		maxResults = 50
	}

	// Construir URL con parámetros
	path := fmt.Sprintf("/rest/api/3/search?jql=%s&fields=summary,status,priority,assignee,updated,issuetype,description&maxResults=%d&startAt=0",
		url.QueryEscape(jql),
		maxResults,
	)

	// Hacer request
	var resp JiraSearchResponse
	if err := p.client.get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("fetching issues: %w", err)
	}

	// Normalizar issues
	normalized := make([]plugin.NormalizedIssue, 0, len(resp.Issues))
	for _, issue := range resp.Issues {
		ni, err := normalizeIssue(issue, p.client.baseURL)
		if err != nil {
			// Log error pero continuar con otros issues
			continue
		}
		normalized = append(normalized, ni)
	}

	return normalized, nil
}

// buildJQL construye la query JQL según el filtro
func buildJQL(filter IssueFilter) string {
	parts := []string{}

	if filter.AssignedToMe {
		parts = append(parts, "assignee = currentUser()")
	}

	if filter.StatusCategory != "" {
		switch filter.StatusCategory {
		case "todo":
			parts = append(parts, "statusCategory = \"To Do\"")
		case "in-progress":
			parts = append(parts, "statusCategory = \"In Progress\"")
		case "done":
			parts = append(parts, "statusCategory = Done")
		}
	} else {
		// Default: excluir issues completados
		parts = append(parts, "statusCategory != Done")
	}

	jql := strings.Join(parts, " AND ")
	jql += " ORDER BY updated DESC"

	return jql
}

// normalizeIssue convierte un JiraIssue al modelo normalizado
func normalizeIssue(issue JiraIssue, baseURL string) (plugin.NormalizedIssue, error) {
	updatedAt, err := parseJiraTime(issue.Fields.Updated)
	if err != nil {
		return plugin.NormalizedIssue{}, fmt.Errorf("parsing updated time: %w", err)
	}

	// Construir URL del issue
	issueURL := fmt.Sprintf("%s/browse/%s", baseURL, issue.Key)

	return plugin.NormalizedIssue{
		Key:       issue.Key,
		Title:     issue.Fields.Summary,
		Status:    normalizeStatus(issue.Fields.Status.Name),
		Priority:  normalizePriority(issue.Fields.Priority.Name),
		UpdatedAt: updatedAt,
		Source:    "jira",
		URL:       issueURL,
	}, nil
}

// normalizeStatus mapea el estado de Jira a los estados normalizados
func normalizeStatus(jiraStatus string) string {
	switch strings.ToLower(jiraStatus) {
	case "open", "to do", "backlog":
		return "todo"
	case "in progress", "in development":
		return "in-progress"
	case "code review", "in review", "pr open":
		return "in-review"
	case "done", "closed", "resolved":
		return "done"
	case "blocked":
		return "blocked"
	default:
		return "todo"
	}
}

// normalizePriority mapea la prioridad de Jira a las prioridades normalizadas
func normalizePriority(jiraPriority string) string {
	switch strings.ToLower(jiraPriority) {
	case "highest", "blocker":
		return "urgent"
	case "high":
		return "high"
	case "medium":
		return "medium"
	case "low", "lowest":
		return "low"
	default:
		return "medium"
	}
}

// issuesCmd crea el comando CLI "getpod jira issues"
func issuesCmd(p *Plugin) *cobra.Command {
	return &cobra.Command{
		Use:   "issues",
		Short: "List issues assigned to me",
		Long:  "Fetch and display issues from Jira assigned to the current user",
		RunE: func(cmd *cobra.Command, args []string) error {
			issues, err := p.FetchIssues(cmd.Context(), IssueFilter{
				AssignedToMe: true,
			})
			if err != nil {
				return fmt.Errorf("fetching issues: %w", err)
			}

			if len(issues) == 0 {
				fmt.Println("No issues found")
				return nil
			}

			// Imprimir tabla
			fmt.Printf("%-15s %-50s %-15s %-10s %-20s\n", "KEY", "TITLE", "STATUS", "PRIORITY", "UPDATED")
			fmt.Println(strings.Repeat("-", 110))

			for _, issue := range issues {
				title := issue.Title
				if len(title) > 50 {
					title = title[:47] + "..."
				}

				fmt.Printf("%-15s %-50s %-15s %-10s %-20s\n",
					issue.Key,
					title,
					issue.Status,
					issue.Priority,
					issue.UpdatedAt.Format("2006-01-02 15:04"),
				)
			}

			fmt.Printf("\nTotal: %d issues\n", len(issues))
			return nil
		},
	}
}

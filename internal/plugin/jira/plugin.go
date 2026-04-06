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
		viewCmd(p),
	}
}

// GetIssue obtiene el detalle completo de un issue por su key
func (p *Plugin) GetIssue(ctx context.Context, key string) (*NormalizedIssueDetail, error) {
	if p.client == nil {
		return nil, fmt.Errorf("plugin not initialized: call Setup() first")
	}

	// Campos necesarios según el spec
	fields := "summary,status,priority,description,comment,subtasks,labels,issuetype,updated"
	path := fmt.Sprintf("/rest/api/3/issue/%s?fields=%s", key, fields)

	var issue JiraIssueDetail
	if err := p.client.get(ctx, path, &issue); err != nil {
		return nil, fmt.Errorf("fetching issue %s: %w", key, err)
	}

	// Parse updated time
	updatedAt, err := parseJiraTime(issue.Fields.Updated)
	if err != nil {
		return nil, fmt.Errorf("parsing updated time: %w", err)
	}

	// Extraer texto plano de description
	description := extractPlainText(issue.Fields.Description)

	// Normalizar comments (últimos 5)
	comments := make([]NormalizedComment, 0)
	commentCount := len(issue.Fields.Comment.Comments)
	startIdx := 0
	if commentCount > 5 {
		startIdx = commentCount - 5
	}

	for i := startIdx; i < commentCount; i++ {
		c := issue.Fields.Comment.Comments[i]
		createdAt, err := parseJiraTime(c.Created)
		if err != nil {
			continue // Skip malformed comments
		}

		comments = append(comments, NormalizedComment{
			Author:    c.Author.DisplayName,
			Body:      extractPlainText(c.Body),
			CreatedAt: createdAt,
		})
	}

	// Construir URL del issue
	issueURL := fmt.Sprintf("%s/browse/%s", p.client.baseURL, issue.Key)

	return &NormalizedIssueDetail{
		Key:         issue.Key,
		Title:       issue.Fields.Summary,
		Status:      normalizeStatus(issue.Fields.Status.Name),
		Priority:    normalizePriority(issue.Fields.Priority.Name),
		UpdatedAt:   updatedAt,
		Source:      "jira",
		URL:         issueURL,
		Description: description,
		Comments:    comments,
		Subtasks:    len(issue.Fields.Subtasks),
		Labels:      issue.Fields.Labels,
	}, nil
}

// GetTransitions obtiene las transiciones disponibles para un issue
func (p *Plugin) GetTransitions(ctx context.Context, key string) ([]Transition, error) {
	if p.client == nil {
		return nil, fmt.Errorf("plugin not initialized: call Setup() first")
	}

	path := fmt.Sprintf("/rest/api/3/issue/%s/transitions", key)

	var resp JiraTransitionsResponse
	if err := p.client.get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("fetching transitions for %s: %w", key, err)
	}

	transitions := make([]Transition, 0, len(resp.Transitions))
	for _, t := range resp.Transitions {
		transitions = append(transitions, Transition{
			ID:   t.ID,
			Name: t.Name,
			To:   normalizeStatus(t.To.Name),
		})
	}

	return transitions, nil
}

// TransitionIssue ejecuta un cambio de estado en un issue
func (p *Plugin) TransitionIssue(ctx context.Context, key, transitionID string) error {
	if p.client == nil {
		return fmt.Errorf("plugin not initialized: call Setup() first")
	}

	path := fmt.Sprintf("/rest/api/3/issue/%s/transitions", key)
	body := map[string]any{
		"transition": map[string]string{
			"id": transitionID,
		},
	}

	if err := p.client.post(ctx, path, body); err != nil {
		return fmt.Errorf("transitioning issue %s: %w", key, err)
	}

	return nil
}

// extractPlainText extrae texto plano de un Atlassian Document Format (ADF)
// Recorre recursivamente el árbol ADF y extrae nodos type="text"
func extractPlainText(adf any) string {
	if adf == nil {
		return ""
	}

	// Convertir a map
	adfMap, ok := adf.(map[string]any)
	if !ok {
		return ""
	}

	var paragraphs []string

	// Función recursiva para procesar nodos
	var processNode func(node map[string]any) string
	processNode = func(node map[string]any) string {
		if node == nil {
			return ""
		}

		nodeType, _ := node["type"].(string)

		// Si es un nodo de texto, retornar el contenido
		if nodeType == "text" {
			if text, ok := node["text"].(string); ok {
				return text
			}
			return ""
		}

		// Si es un párrafo, procesar su contenido como una unidad
		if nodeType == "paragraph" {
			var paragraphText strings.Builder
			if content, ok := node["content"].([]any); ok {
				for _, item := range content {
					if itemMap, ok := item.(map[string]any); ok {
						paragraphText.WriteString(processNode(itemMap))
					}
				}
			}
			return paragraphText.String()
		}

		// Para otros nodos con contenido, procesar recursivamente
		if content, ok := node["content"].([]any); ok {
			for _, item := range content {
				if itemMap, ok := item.(map[string]any); ok {
					itemType, _ := itemMap["type"].(string)
					if itemType == "paragraph" {
						// Acumular párrafos
						p := processNode(itemMap)
						if p != "" {
							paragraphs = append(paragraphs, p)
						}
					} else {
						return processNode(itemMap)
					}
				}
			}
		}

		return ""
	}

	processNode(adfMap)
	return strings.Join(paragraphs, "\n")
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

// viewCmd crea el comando CLI "getpod jira view"
func viewCmd(p *Plugin) *cobra.Command {
	return &cobra.Command{
		Use:   "view [key]",
		Short: "View issue details",
		Long:  "Display detailed information about a specific Jira issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			detail, err := p.GetIssue(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("fetching issue: %w", err)
			}

			// Renderizar con lipgloss (importar al inicio del archivo)
			// Por ahora, output simple sin lipgloss para evitar import circular
			fmt.Printf("\n%s: %s\n\n", detail.Key, detail.Title)
			fmt.Printf("Status: %s\n", detail.Status)
			fmt.Printf("Priority: %s\n", detail.Priority)
			fmt.Printf("Updated: %s\n", detail.UpdatedAt.Format("2006-01-02 15:04"))
			fmt.Printf("URL: %s\n", detail.URL)

			if len(detail.Labels) > 0 {
				fmt.Printf("Labels: %s\n", strings.Join(detail.Labels, ", "))
			}

			if detail.Subtasks > 0 {
				fmt.Printf("Subtasks: %d\n", detail.Subtasks)
			}

			fmt.Println("\n--- Description ---")
			if detail.Description != "" {
				fmt.Println(detail.Description)
			} else {
				fmt.Println("(empty)")
			}

			if len(detail.Comments) > 0 {
				fmt.Printf("\n--- Comments (%d) ---\n", len(detail.Comments))
				for _, c := range detail.Comments {
					fmt.Printf("\n%s (%s):\n", c.Author, c.CreatedAt.Format("2006-01-02 15:04"))
					fmt.Println(c.Body)
				}
			}

			fmt.Println()
			return nil
		},
	}
}

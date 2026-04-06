package jira

import (
	"fmt"
	"time"
)

type JiraUser struct {
	AccountID   string `json:"accountId"`
	Email       string `json:"emailAddress"`
	DisplayName string `json:"displayName"`
}

type JiraError struct {
	ErrorMessages []string          `json:"errorMessages"`
	Errors        map[string]string `json:"errors"`
}

// JiraSearchResponse representa la respuesta de /rest/api/3/search y /rest/api/3/search/jql
type JiraSearchResponse struct {
	Issues        []JiraIssue `json:"issues"`
	Total         int         `json:"total"`         // solo en /search (deprecated)
	StartAt       int         `json:"startAt"`       // solo en /search (deprecated)
	MaxResults    int         `json:"maxResults"`    // solo en /search (deprecated)
	NextPageToken string      `json:"nextPageToken"` // solo en /search/jql (nuevo)
	IsLast        bool        `json:"isLast"`        // solo en /search/jql (nuevo)
}

// JiraIssue representa un issue individual en la respuesta de Jira
type JiraIssue struct {
	Key    string          `json:"key"`
	Self   string          `json:"self"`
	Fields JiraIssueFields `json:"fields"`
}

// JiraIssueFields contiene los campos específicos del issue
type JiraIssueFields struct {
	Summary     string        `json:"summary"`
	Status      JiraStatus    `json:"status"`
	Priority    JiraPriority  `json:"priority"`
	Updated     string        `json:"updated"`
	IssueType   JiraIssueType `json:"issuetype"`
	Description any           `json:"description"` // ADF format
}

// JiraStatus representa el estado del issue
type JiraStatus struct {
	Name           string `json:"name"`
	StatusCategory struct {
		Key string `json:"key"` // "new", "indeterminate", "done"
	} `json:"statusCategory"`
}

// JiraPriority representa la prioridad del issue
type JiraPriority struct {
	Name string `json:"name"`
}

// JiraIssueType representa el tipo de issue
type JiraIssueType struct {
	Name string `json:"name"`
}

// Transition representa una transición de estado disponible para un issue
type Transition struct {
	ID   string // ID que usa la API para ejecutar la transición
	Name string // Nombre de la transición ("In Progress", "Done", etc.)
	To   string // Status destino normalizado
}

// NormalizedComment representa un comentario normalizado
type NormalizedComment struct {
	Author    string
	Body      string // Texto plano
	CreatedAt time.Time
}

// NormalizedIssueDetail extiende NormalizedIssue con información adicional
type NormalizedIssueDetail struct {
	Key         string
	Title       string
	Status      string
	Priority    string
	UpdatedAt   time.Time
	Source      string
	URL         string
	Description string              // Texto plano del description (sin ADF)
	Comments    []NormalizedComment // Últimos 5 comentarios
	Subtasks    int                 // Count de subtareas
	Labels      []string
}

// JiraTransitionsResponse representa la respuesta de /rest/api/3/issue/{key}/transitions
type JiraTransitionsResponse struct {
	Transitions []JiraTransition `json:"transitions"`
}

// JiraTransition representa una transición individual
type JiraTransition struct {
	ID   string     `json:"id"`
	Name string     `json:"name"`
	To   JiraStatus `json:"to"`
}

// JiraComment representa un comentario en la respuesta de Jira
type JiraComment struct {
	Author struct {
		DisplayName string `json:"displayName"`
	} `json:"author"`
	Body    any    `json:"body"` // ADF format
	Created string `json:"created"`
}

// JiraIssueDetail representa la respuesta completa de /rest/api/3/issue/{key}
type JiraIssueDetail struct {
	Key    string                `json:"key"`
	Fields JiraIssueDetailFields `json:"fields"`
}

// JiraIssueDetailFields contiene los campos completos del issue
type JiraIssueDetailFields struct {
	Summary     string        `json:"summary"`
	Status      JiraStatus    `json:"status"`
	Priority    JiraPriority  `json:"priority"`
	Updated     string        `json:"updated"`
	IssueType   JiraIssueType `json:"issuetype"`
	Description any           `json:"description"` // ADF format
	Comment     struct {
		Comments []JiraComment `json:"comments"`
	} `json:"comment"`
	Subtasks []any    `json:"subtasks"` // Solo necesitamos el count
	Labels   []string `json:"labels"`
}

// JiraCommentRequest estructura para POST /rest/api/3/issue/{key}/comment
type JiraCommentRequest struct {
	Body map[string]any `json:"body"` // Atlassian Document Format (ADF)
}

// parseJiraTime parsea el formato de tiempo de Jira a time.Time
// Soporta múltiples formatos: RFC3339 y formato con zona horaria sin dos puntos
func parseJiraTime(s string) (time.Time, error) {
	// Intentar RFC3339 primero (formato estándar)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Intentar formato con milisegundos y zona horaria sin dos puntos
	// Ejemplo: "2026-04-04T08:41:47.075-0500"
	if t, err := time.Parse("2006-01-02T15:04:05.999-0700", s); err == nil {
		return t, nil
	}

	// Intentar sin milisegundos
	if t, err := time.Parse("2006-01-02T15:04:05-0700", s); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("formato de tiempo no reconocido: %s", s)
}

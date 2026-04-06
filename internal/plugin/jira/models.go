package jira

import "time"

type JiraUser struct {
	AccountID   string `json:"accountId"`
	Email       string `json:"emailAddress"`
	DisplayName string `json:"displayName"`
}

type JiraError struct {
	ErrorMessages []string          `json:"errorMessages"`
	Errors        map[string]string `json:"errors"`
}

// JiraSearchResponse representa la respuesta de /rest/api/3/search
type JiraSearchResponse struct {
	Issues     []JiraIssue `json:"issues"`
	Total      int         `json:"total"`
	StartAt    int         `json:"startAt"`
	MaxResults int         `json:"maxResults"`
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
	Description string        `json:"description"`
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

// parseJiraTime parsea el formato de tiempo de Jira a time.Time
func parseJiraTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

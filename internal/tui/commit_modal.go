package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// commitTypes are the conventional commit types offered in the selector.
var commitTypes = []string{"feat", "fix", "chore", "refactor", "docs", "test", "ci", "style", "perf"}

// CommitModal lets the user compose a conventional commit message.
// It has three fields: type (selector), scope (text), and message (text).
type CommitModal struct {
	// Field selection: 0=type, 1=scope, 2=message
	field    int
	typeIdx  int
	scope    []rune
	message  []rune
	styles   Styles
	issueKey string // pre-filled as default scope
}

// NewCommitModal creates the commit modal. issueKey is used as default scope.
func NewCommitModal(issueKey string, styles Styles) *CommitModal {
	scope := []rune(strings.ToLower(issueKey))
	return &CommitModal{
		issueKey: issueKey,
		scope:    scope,
		styles:   styles,
	}
}

func (m *CommitModal) Title() string { return "Commit + Push" }

func (m *CommitModal) Init() tea.Cmd { return nil }

func (m *CommitModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return ModalClosedMsg{} }
		case "tab":
			m.field = (m.field + 1) % 3
		case "shift+tab":
			m.field = (m.field + 2) % 3
		case "ctrl+s":
			msgText := strings.TrimSpace(string(m.message))
			if msgText == "" {
				return m, nil // message is required
			}
			return m, func() tea.Msg {
				return CommitSubmitMsg{
					Type:    commitTypes[m.typeIdx],
					Scope:   string(m.scope),
					Message: msgText,
				}
			}
		default:
			switch m.field {
			case 0: // type selector
				m.handleTypeKey(msg)
			case 1: // scope input
				m.scope = handleTextInput(m.scope, msg)
			case 2: // message input
				m.message = handleTextInput(m.message, msg)
			}
		}
	}
	return m, nil
}

func (m *CommitModal) handleTypeKey(msg tea.KeyMsg) {
	switch msg.String() {
	case "up", "k":
		if m.typeIdx > 0 {
			m.typeIdx--
		}
	case "down", "j":
		if m.typeIdx < len(commitTypes)-1 {
			m.typeIdx++
		}
	case "left":
		if m.typeIdx > 0 {
			m.typeIdx--
		}
	case "right":
		if m.typeIdx < len(commitTypes)-1 {
			m.typeIdx++
		}
	}
}

// handleTextInput handles typing in a text field and returns the updated runes.
func handleTextInput(buf []rune, msg tea.KeyMsg) []rune {
	switch msg.String() {
	case "backspace":
		if len(buf) > 0 {
			buf = buf[:len(buf)-1]
		}
	default:
		if len(msg.Runes) > 0 {
			buf = append(buf, msg.Runes...)
		}
	}
	return buf
}

func (m *CommitModal) View() string {
	active := lipgloss.NewStyle().Foreground(Primary400)
	label := m.styles.Muted

	var lines []string

	// Type selector
	typeLabel := "Type:    "
	if m.field == 0 {
		typeLabel = active.Render("> ") + typeLabel
	} else {
		typeLabel = "  " + typeLabel
	}
	var types []string
	for i, t := range commitTypes {
		if i == m.typeIdx {
			types = append(types, active.Render("["+t+"]"))
		} else {
			types = append(types, label.Render(" "+t+" "))
		}
	}
	lines = append(lines, typeLabel+strings.Join(types, " "))

	// Scope input
	scopeLabel := "Scope:   "
	scopeVal := string(m.scope)
	if m.field == 1 {
		scopeLabel = active.Render("> ") + scopeLabel
		scopeVal += "_"
	} else {
		scopeLabel = "  " + scopeLabel
	}
	if scopeVal == "" {
		scopeVal = label.Render("(optional)")
	}
	lines = append(lines, scopeLabel+m.styles.Paragraph.Render(scopeVal))

	// Message input
	msgLabel := "Message: "
	msgVal := string(m.message)
	if m.field == 2 {
		msgLabel = active.Render("> ") + msgLabel
		msgVal += "_"
	} else {
		msgLabel = "  " + msgLabel
	}
	if msgVal == "" || msgVal == "_" {
		if m.field == 2 {
			msgVal = "_"
		} else {
			msgVal = label.Render("(required)")
		}
	}
	lines = append(lines, msgLabel+m.styles.Paragraph.Render(msgVal))

	// Preview
	lines = append(lines, "")
	preview := commitTypes[m.typeIdx]
	if scope := string(m.scope); scope != "" {
		preview += "(" + scope + ")"
	}
	preview += ": " + string(m.message)
	lines = append(lines, label.Render("Preview: ")+m.styles.Paragraph.Render(preview))

	lines = append(lines, "")
	lines = append(lines, m.styles.Muted.Render("tab next field · ctrl+s commit+push · esc cancel"))
	return strings.Join(lines, "\n")
}

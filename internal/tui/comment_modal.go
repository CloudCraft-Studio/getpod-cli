package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// CommentModal provides a text input for writing a comment on an issue.
// Supports multi-line input: Enter adds a newline, Ctrl+S submits.
type CommentModal struct {
	body   []rune
	cursor int
	styles Styles
}

// NewCommentModal creates the comment modal.
func NewCommentModal(styles Styles) *CommentModal {
	return &CommentModal{styles: styles}
}

func (m *CommentModal) Title() string { return "Add Comment" }

func (m *CommentModal) Init() tea.Cmd { return nil }

func (m *CommentModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return ModalClosedMsg{} }
		case "ctrl+s":
			body := strings.TrimSpace(string(m.body))
			if body == "" {
				return m, nil // don't submit empty comments
			}
			return m, func() tea.Msg { return CommentSubmitMsg{Body: body} }
		case "enter":
			m.body = append(m.body[:m.cursor], append([]rune{'\n'}, m.body[m.cursor:]...)...)
			m.cursor++
		case "backspace":
			if m.cursor > 0 {
				m.body = append(m.body[:m.cursor-1], m.body[m.cursor:]...)
				m.cursor--
			}
		case "left":
			if m.cursor > 0 {
				m.cursor--
			}
		case "right":
			if m.cursor < len(m.body) {
				m.cursor++
			}
		default:
			if len(msg.Runes) > 0 {
				runes := msg.Runes
				m.body = append(m.body[:m.cursor], append(runes, m.body[m.cursor:]...)...)
				m.cursor += len(runes)
			}
		}
	}
	return m, nil
}

func (m *CommentModal) View() string {
	var lines []string

	text := string(m.body)
	if text == "" {
		lines = append(lines, m.styles.Muted.Render("Type your comment..."))
	} else {
		lines = append(lines, m.styles.Paragraph.Render(text+"_"))
	}

	lines = append(lines, "")
	lines = append(lines, m.styles.Muted.Render("ctrl+s submit · esc cancel"))
	return strings.Join(lines, "\n")
}

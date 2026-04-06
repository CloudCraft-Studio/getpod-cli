package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CloudCraft-Studio/getpod-cli/internal/git"
)

// PRModal lets the user configure and submit a pull request.
// Fields: title (pre-filled), body (pre-filled), base branch (selector).
type PRModal struct {
	// Field selection: 0=title, 1=body, 2=baseBranch
	field     int
	title     []rune
	body      []rune
	branches  []string
	branchIdx int
	styles    Styles
}

// baseBranchOptions returns the common target branches.
var baseBranchOptions = []string{"develop", "release", "main"}

// NewPRModal creates the PR modal with pre-filled values from the issue.
func NewPRModal(issueKey, issueTitle, environment, workspace string, styles Styles) *PRModal {
	title := fmt.Sprintf("%s: %s", issueKey, issueTitle)
	body := fmt.Sprintf("## %s\n\n**Workspace:** %s\n**Environment:** %s\n",
		issueTitle, workspace, environment)

	// Select the suggested base branch
	suggested := git.SuggestBaseBranch(environment)
	branchIdx := 0
	for i, b := range baseBranchOptions {
		if b == suggested {
			branchIdx = i
			break
		}
	}

	return &PRModal{
		title:     []rune(title),
		body:      []rune(body),
		branches:  baseBranchOptions,
		branchIdx: branchIdx,
		styles:    styles,
	}
}

func (m *PRModal) Title() string { return "Create Pull Request" }

func (m *PRModal) Init() tea.Cmd { return nil }

func (m *PRModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			titleStr := strings.TrimSpace(string(m.title))
			if titleStr == "" {
				return m, nil // title required
			}
			return m, func() tea.Msg {
				return PRSubmitMsg{
					Title:      titleStr,
					Body:       string(m.body),
					BaseBranch: baseBranchOptions[m.branchIdx],
				}
			}
		default:
			switch m.field {
			case 0: // title
				m.title = handleTextInput(m.title, msg)
			case 1: // body
				m.handleBodyKey(msg)
			case 2: // base branch selector
				m.handleBranchKey(msg)
			}
		}
	}
	return m, nil
}

func (m *PRModal) handleBodyKey(msg tea.KeyMsg) {
	switch msg.String() {
	case "enter":
		m.body = append(m.body, '\n')
	case "backspace":
		if len(m.body) > 0 {
			m.body = m.body[:len(m.body)-1]
		}
	default:
		if len(msg.Runes) > 0 {
			m.body = append(m.body, msg.Runes...)
		}
	}
}

func (m *PRModal) handleBranchKey(msg tea.KeyMsg) {
	switch msg.String() {
	case "up", "k", "left":
		if m.branchIdx > 0 {
			m.branchIdx--
		}
	case "down", "j", "right":
		if m.branchIdx < len(m.branches)-1 {
			m.branchIdx++
		}
	}
}

func (m *PRModal) View() string {
	active := lipgloss.NewStyle().Foreground(Primary400)
	label := m.styles.Muted

	var lines []string

	// Title
	titleLabel := "Title:  "
	titleVal := string(m.title)
	if m.field == 0 {
		titleLabel = active.Render("> ") + titleLabel
		titleVal += "_"
	} else {
		titleLabel = "  " + titleLabel
	}
	lines = append(lines, titleLabel+m.styles.Paragraph.Render(titleVal))

	// Body
	bodyLabel := "Body:   "
	bodyVal := string(m.body)
	if m.field == 1 {
		bodyLabel = active.Render("> ") + bodyLabel
		bodyVal += "_"
	} else {
		bodyLabel = "  " + bodyLabel
	}
	// Show first 3 lines of body
	bodyLines := strings.Split(bodyVal, "\n")
	if len(bodyLines) > 3 {
		bodyVal = strings.Join(bodyLines[:3], "\n") + fmt.Sprintf("\n... (+%d lines)", len(bodyLines)-3)
	}
	lines = append(lines, bodyLabel+m.styles.Paragraph.Render(bodyVal))

	// Base branch
	branchLabel := "Base:   "
	if m.field == 2 {
		branchLabel = active.Render("> ") + branchLabel
	} else {
		branchLabel = "  " + branchLabel
	}
	var branchItems []string
	for i, b := range m.branches {
		if i == m.branchIdx {
			branchItems = append(branchItems, active.Render("["+b+"]"))
		} else {
			branchItems = append(branchItems, label.Render(" "+b+" "))
		}
	}
	lines = append(lines, branchLabel+strings.Join(branchItems, " "))

	lines = append(lines, "")
	lines = append(lines, m.styles.Muted.Render("tab next field · ctrl+s create PR · esc cancel"))
	return strings.Join(lines, "\n")
}

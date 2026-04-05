package tui

import "github.com/charmbracelet/lipgloss"

// Tokyo Night palette as lipgloss.Color
var (
	// Base colors
	Black = lipgloss.Color("#15161e")
	Grey0 = lipgloss.Color("#1a1b26")
	Grey1 = lipgloss.Color("#24283b")
	Grey2 = lipgloss.Color("#2e313f")
	Grey3 = lipgloss.Color("#3b4261")
	Grey4 = lipgloss.Color("#545c7e")
	Grey5 = lipgloss.Color("#7aa2f7")
	Grey6 = lipgloss.Color("#9d7cd8")
	Grey7 = lipgloss.Color("#c0caf5")

	// Accent colors
	Blue      = lipgloss.Color("#7aa2f7")
	LightBlue = lipgloss.Color("#89ddff")
	Cyan      = lipgloss.Color("#7dcfff")
	Teal      = lipgloss.Color("#73daca")
	Green     = lipgloss.Color("#9ece6a")
	Yellow    = lipgloss.Color("#e0af68")
	Orange    = lipgloss.Color("#ff9e64")
	Red       = lipgloss.Color("#f7768e")
	Magenta   = lipgloss.Color("#bb9af7")
	Violet    = lipgloss.Color("#9d7cd8")

	// Text colors
	Text      = lipgloss.Color("#a9b1d6")
	TextMuted = lipgloss.Color("#565f89")
)

// Styles container
type Styles struct {
	// Layout
	AppContainer lipgloss.Style
	TopBar       lipgloss.Style
	ContentArea  lipgloss.Style
	Footer       lipgloss.Style

	// Navigation
	ClientTab       lipgloss.Style
	ClientTabActive lipgloss.Style
	NavTab          lipgloss.Style
	NavTabActive    lipgloss.Style
	Badge           lipgloss.Style
	BadgeSuccess    lipgloss.Style
	BadgeWarning    lipgloss.Style
	BadgeDanger     lipgloss.Style
	BadgeInfo       lipgloss.Style

	// Text
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	Paragraph   lipgloss.Style
	HelpKey     lipgloss.Style
	HelpDesc    lipgloss.Style
	Placeholder lipgloss.Style

	// Borders
	BorderActive   lipgloss.Border
	BorderInactive lipgloss.Border
}

func DefaultStyles() Styles {
	// Border definitions
	activeBorder := lipgloss.Border{
		Top:         "▄",
		Bottom:      "▀",
		Left:        "▌",
		Right:       "▐",
		TopLeft:     "▜",
		TopRight:    "▛",
		BottomLeft:  "▟",
		BottomRight: "▙",
	}

	inactiveBorder := lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "┌",
		TopRight:    "┐",
		BottomLeft:  "└",
		BottomRight: "┘",
	}

	return Styles{
		// Layout
		AppContainer: lipgloss.NewStyle().
			Background(Grey0).
			Foreground(Text).
			Padding(0, 1),

		TopBar: lipgloss.NewStyle().
			Background(Grey1).
			Foreground(Text).
			Padding(0, 1),

		ContentArea: lipgloss.NewStyle().
			Background(Grey0).
			Foreground(Text),

		Footer: lipgloss.NewStyle().
			Background(Grey2).
			Foreground(TextMuted).
			Padding(0, 1),

		// Navigation
		ClientTab: lipgloss.NewStyle().
			Foreground(TextMuted).
			Padding(0, 2),

		ClientTabActive: lipgloss.NewStyle().
			Foreground(Green).
			Bold(true).
			Padding(0, 2),

		NavTab: lipgloss.NewStyle().
			Foreground(TextMuted).
			Padding(0, 2).
			Margin(0, 1),

		NavTabActive: lipgloss.NewStyle().
			Foreground(Blue).
			Bold(true).
			Padding(0, 2).
			Margin(0, 1).
			BorderBottom(true).
			BorderForeground(Blue),

		Badge: lipgloss.NewStyle().
			Background(Grey3).
			Foreground(Text).
			Padding(0, 1).
			Bold(true),

		BadgeSuccess: lipgloss.NewStyle().
			Background(Green).
			Foreground(Black).
			Padding(0, 1).
			Bold(true),

		BadgeWarning: lipgloss.NewStyle().
			Background(Yellow).
			Foreground(Black).
			Padding(0, 1).
			Bold(true),

		BadgeDanger: lipgloss.NewStyle().
			Background(Red).
			Foreground(Black).
			Padding(0, 1).
			Bold(true),

		BadgeInfo: lipgloss.NewStyle().
			Background(Blue).
			Foreground(Black).
			Padding(0, 1).
			Bold(true),

		// Text
		Title: lipgloss.NewStyle().
			Foreground(Blue).
			Bold(true),

		Subtitle: lipgloss.NewStyle().
			Foreground(TextMuted).
			Bold(true),

		Paragraph: lipgloss.NewStyle().
			Foreground(Text),

		HelpKey: lipgloss.NewStyle().
			Foreground(Yellow).
			Bold(true),

		HelpDesc: lipgloss.NewStyle().
			Foreground(TextMuted),

		Placeholder: lipgloss.NewStyle().
			Foreground(TextMuted).
			Italic(true),

		// Borders
		BorderActive:   activeBorder,
		BorderInactive: inactiveBorder,
	}
}

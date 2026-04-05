package tui

import "github.com/charmbracelet/lipgloss"

// GetPod design system colors (from web app)
var (
	// Surface colors (backgrounds & borders)
	Surface950 = lipgloss.Color("#0a0e1a") // Primary background
	Surface900 = lipgloss.Color("#0f172a") // Cards, panels
	Surface800 = lipgloss.Color("#1e293b") // Inputs, secondary
	Surface700 = lipgloss.Color("#334155") // Normal borders
	Surface600 = lipgloss.Color("#475569") // Subtle borders

	// Primary colors (Cyan - main accent)
	Primary300 = lipgloss.Color("#67e8f9") // Hover text
	Primary400 = lipgloss.Color("#22d3ee") // Brand text, accent, icons, links
	Primary500 = lipgloss.Color("#06b6d4") // Focus borders, tinted backgrounds
	Primary600 = lipgloss.Color("#0891b2") // Gradient start

	// Secondary colors (Violet - secondary accent)
	Secondary400 = lipgloss.Color("#a78bfa") // Secondary icons
	Secondary500 = lipgloss.Color("#8b5cf6") // Tinted backgrounds
	Secondary600 = lipgloss.Color("#7c3aed") // Gradient end

	// Content/text colors
	Content100 = lipgloss.Color("#f1f5f9") // Headings, primary text
	Content300 = lipgloss.Color("#cbd5e1") // Body text, labels
	Content400 = lipgloss.Color("#94a3b8") // Muted text, subtitles
	Content500 = lipgloss.Color("#64748b") // Placeholders, metadata
	Content600 = lipgloss.Color("#475569") // Disabled text

	// Semantic colors
	Success400 = lipgloss.Color("#34d399")
	Success500 = lipgloss.Color("#10b981")
	Warning400 = lipgloss.Color("#fbbf24")
	Warning500 = lipgloss.Color("#f59e0b")
	Danger400  = lipgloss.Color("#fb7185")
	Danger500  = lipgloss.Color("#f43f5e")
	Info400    = lipgloss.Color("#93c5fd")
	Info500    = lipgloss.Color("#3b82f6")
)

// Styles container
type Styles struct {
	// Navigation
	ClientButton       lipgloss.Style
	ClientButtonActive lipgloss.Style
	NavTab             lipgloss.Style
	NavTabActive       lipgloss.Style

	// Badges
	Badge        lipgloss.Style
	BadgeSuccess lipgloss.Style
	BadgeWarning lipgloss.Style
	BadgeDanger  lipgloss.Style
	BadgeInfo    lipgloss.Style

	// Text
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	Paragraph   lipgloss.Style
	Muted       lipgloss.Style
	BrandText   lipgloss.Style
	HelpKey     lipgloss.Style
	HelpDesc    lipgloss.Style
	Placeholder lipgloss.Style

	// Borders
	BorderRounded lipgloss.Border
}

func DefaultStyles() Styles {
	borderRounded := lipgloss.RoundedBorder()

	return Styles{
		// Navigation - Client buttons (complete boxes)
		ClientButton: lipgloss.NewStyle().
			Foreground(Content400).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Surface700).
			Padding(0, 1).
			MarginRight(1),

		ClientButtonActive: lipgloss.NewStyle().
			Foreground(Primary400).
			Bold(true).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary400).
			Padding(0, 1).
			MarginRight(1),

		NavTab: lipgloss.NewStyle().
			Foreground(Content400).
			Padding(0, 2),

		NavTabActive: lipgloss.NewStyle().
			Foreground(Primary400).
			Bold(true).
			Padding(0, 2),

		// Badges
		Badge: lipgloss.NewStyle().
			Background(Primary500).
			Foreground(Surface950).
			Bold(true),

		BadgeSuccess: lipgloss.NewStyle().
			Background(Success500).
			Foreground(Surface950).
			Bold(true),

		BadgeWarning: lipgloss.NewStyle().
			Background(Warning500).
			Foreground(Surface950).
			Bold(true),

		BadgeDanger: lipgloss.NewStyle().
			Background(Danger500).
			Foreground(Surface950).
			Bold(true),

		BadgeInfo: lipgloss.NewStyle().
			Background(Info500).
			Foreground(Surface950).
			Bold(true),

		// Text
		Title: lipgloss.NewStyle().
			Foreground(Content100).
			Bold(true),

		BrandText: lipgloss.NewStyle().
			Foreground(Primary400).
			Bold(true),

		Subtitle: lipgloss.NewStyle().
			Foreground(Content400),

		Paragraph: lipgloss.NewStyle().
			Foreground(Content300),

		Muted: lipgloss.NewStyle().
			Foreground(Content500),

		HelpKey: lipgloss.NewStyle().
			Foreground(Primary300).
			Bold(true),

		HelpDesc: lipgloss.NewStyle().
			Foreground(Content400),

		Placeholder: lipgloss.NewStyle().
			Foreground(Content500).
			Italic(true),

		// Borders
		BorderRounded: borderRounded,
	}
}

package tui

import tea "github.com/charmbracelet/bubbletea"

// Modal is the interface for all overlay components.
// App uses hasModal bool alongside this to avoid the nil-interface trap.
type Modal interface {
	tea.Model
	Title() string
}

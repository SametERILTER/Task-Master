package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"tm/internal/theme"
)

type InputMode int

const (
	ModeNormal InputMode = iota
	ModeCommand          // user typed ':'
	ModeSearch           // user typed '/'
)

// CommandBar handles the interactive bottom input.
type CommandBar struct {
	Input  textinput.Model
	Mode   InputMode
	Status string // temporary feedback message e.g. "Killed PID 1234"
	IsTree bool
}

// NewCommandBar creates a configured CommandBar.
func NewCommandBar() CommandBar {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 120
	return CommandBar{
		Input: ti,
		Mode:  ModeNormal,
	}
}

// OpenCommand opens input in ':' command mode.
func (cb *CommandBar) OpenCommand() tea.Cmd {
	cb.Mode = ModeCommand
	cb.Input.Prompt = ":"
	cb.Input.SetValue("")
	cb.Input.Focus()
	return textinput.Blink
}

// OpenSearch opens input in '/' search mode.
func (cb *CommandBar) OpenSearch() tea.Cmd {
	cb.Mode = ModeSearch
	cb.Input.Prompt = "/"
	cb.Input.SetValue("")
	cb.Input.Focus()
	return textinput.Blink
}

// Close cancels input and resets to normal mode.
func (cb *CommandBar) Close() {
	cb.Mode = ModeNormal
	cb.Input.Blur()
	cb.Input.SetValue("")
}

// View renders the bottom bar with the given styles.
func (cb CommandBar) View(styles theme.Styles, width int) string {
	if cb.Mode != ModeNormal {
		promptStyle := styles.CommandPrompt
		return promptStyle.Render(cb.Input.Prompt) + styles.CommandInput.Render(cb.Input.Value())
	}

	// Always display the initial command bar
	helpText := " [p] Tree  [Tab] Ports  [:] Command  [/] Search  [Enter] Details  [?] Help  [x] Kill  [s] Sort  [q] Quit"
	if cb.IsTree {
		helpText = " [p] Flat List  [Tab] Ports  [:] Command  [/] Search  [Enter] Details  [?] Help  [x] Kill  [s] Sort  [q] Quit"
	}
	base := styles.StatusBar.Render(helpText)

	if cb.Status != "" {
		return base + styles.Warning.Render("  » "+cb.Status)
	}

	return base
}

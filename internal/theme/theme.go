package theme

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Theme represents a complete color scheme for tm.
type Theme struct {
	Name        string
	Primary     string // Accent color (headers, highlights)
	Secondary   string // Sub-accents
	Background  string // Background tint
	Text        string // Primary text
	DimText     string // Subdued/muted text
	Border      string // Border color
	Warning     string // Warning metric color
	Danger      string // Critical / Danger color
	Success     string // Good metric color
	SelectedBg  string // Selected row background
	SelectedFg  string // Selected row text
}

// Styles holds Lipgloss styles pre-built from a Theme.
type Styles struct {
	Title         lipgloss.Style
	Subtitle      lipgloss.Style
	HeaderCard    lipgloss.Style
	CardLabel     lipgloss.Style
	CardValue     lipgloss.Style
	BorderBox     lipgloss.Style
	TableHeader   lipgloss.Style
	TableRow      lipgloss.Style
	TableRowSel   lipgloss.Style
	CommandPrompt lipgloss.Style
	CommandInput  lipgloss.Style
	StatusBar     lipgloss.Style
	Danger        lipgloss.Style
	Success       lipgloss.Style
	Warning       lipgloss.Style
	Dim           lipgloss.Style
}

// MakeStyles creates a cached Styles bundle from a Theme.
func (t Theme) MakeStyles() Styles {
	return Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(t.Primary)),

		Subtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.DimText)),

		HeaderCard: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(t.Border)).
			Padding(0, 1),

		CardLabel: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.DimText)).
			Bold(true),

		CardValue: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Primary)).
			Bold(true),

		BorderBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(t.Border)),

		TableHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(t.Primary)).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(t.Border)),

		TableRow: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Text)),

		TableRowSel: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(t.SelectedFg)).
			Background(lipgloss.Color(t.SelectedBg)),

		CommandPrompt: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(t.Secondary)),

		CommandInput: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Text)),

		StatusBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.DimText)),

		Danger:  lipgloss.NewStyle().Foreground(lipgloss.Color(t.Danger)),
		Success: lipgloss.NewStyle().Foreground(lipgloss.Color(t.Success)),
		Warning: lipgloss.NewStyle().Foreground(lipgloss.Color(t.Warning)),
		Dim:     lipgloss.NewStyle().Foreground(lipgloss.Color(t.DimText)),
	}
}

// AvailableThemes registry
var Themes = map[string]Theme{
	"default": {
		Name:       "default",
		Primary:    "#58A6FF", // Clean standard terminal blue
		Secondary:  "#79C0FF", // Soft sky blue
		Background: "",        // Transparent/native terminal background
		Text:       "#E6EDF3", // Crisp readable text
		DimText:    "#8B949E", // Subtle muted gray
		Border:     "#30363D", // Modern subtle border
		Warning:    "#D29922", // Warm yellow
		Danger:     "#F85149", // Clean red
		Success:    "#3FB950", // Clean emerald green
		SelectedBg: "#1F6FEB", // Standard selection highlight
		SelectedFg: "#FFFFFF",
	},
	"matrix": {
		Name:       "matrix",
		Primary:    "#00FF66",
		Secondary:  "#00CC44",
		Background: "#0A0F0D",
		Text:       "#E0FFE0",
		DimText:    "#4D7A57",
		Border:     "#008833",
		Warning:    "#FFEE55",
		Danger:     "#FF3344",
		Success:    "#00FF66",
		SelectedBg: "#00441B",
		SelectedFg: "#FFFFFF",
	},
	"cyberpunk": {
		Name:       "cyberpunk",
		Primary:    "#00F0FF",
		Secondary:  "#FFE600",
		Background: "#0D0221",
		Text:       "#FFFFFF",
		DimText:    "#7B61FF",
		Border:     "#FF007F",
		Warning:    "#FFE600",
		Danger:     "#FF0055",
		Success:    "#00F0FF",
		SelectedBg: "#FF007F",
		SelectedFg: "#000000",
	},
	"dracula": {
		Name:       "dracula",
		Primary:    "#BD93F9",
		Secondary:  "#FF79C6",
		Background: "#282A36",
		Text:       "#F8F8F2",
		DimText:    "#6272A4",
		Border:     "#6272A4",
		Warning:    "#F1FA8C",
		Danger:     "#FF5555",
		Success:    "#50FA7B",
		SelectedBg: "#44475A",
		SelectedFg: "#F8F8F2",
	},
	"nord": {
		Name:       "nord",
		Primary:    "#88C0D0",
		Secondary:  "#81A1C1",
		Background: "#2E3440",
		Text:       "#ECEFF4",
		DimText:    "#4C566A",
		Border:     "#434C5E",
		Warning:    "#EBCB8B",
		Danger:     "#BF616A",
		Success:    "#A3BE8C",
		SelectedBg: "#3B4252",
		SelectedFg: "#88C0D0",
	},
}

// GetTheme returns the theme by name, defaults to "default" (normal colors)
func GetTheme(name string) Theme {
	switch strings.ToLower(name) {
	case "classic", "standard", "normal":
		return Themes["default"]
	}
	if t, ok := Themes[strings.ToLower(name)]; ok {
		return t
	}
	return Themes["default"]
}

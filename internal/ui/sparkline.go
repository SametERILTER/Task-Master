package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var blocks = []rune{' ', ' ', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// RenderSparkline renders a 1-row sparkline representing historical float values (0 to 100).
func RenderSparkline(values []float64, maxLen int, style lipgloss.Style) string {
	if len(values) == 0 || maxLen <= 0 {
		return ""
	}

	start := 0
	if len(values) > maxLen {
		start = len(values) - maxLen
	}
	slice := values[start:]

	var sb strings.Builder
	for _, v := range slice {
		if v < 0 {
			v = 0
		}
		if v > 100 {
			v = 100
		}
		idx := int((v / 100.0) * float64(len(blocks)-1))
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		sb.WriteRune(blocks[idx])
	}

	return style.Render(sb.String())
}

// RenderProgressBar renders an ASCII progress bar [██████░░░░░] 65.4%
func RenderProgressBar(pct float64, width int, filledStyle, emptyStyle lipgloss.Style) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	if width < 5 {
		width = 10
	}

	filledChars := int((pct / 100.0) * float64(width))
	if filledChars > width {
		filledChars = width
	}
	emptyChars := width - filledChars

	filled := strings.Repeat("█", filledChars)
	empty := strings.Repeat("░", emptyChars)

	return filledStyle.Render(filled) + emptyStyle.Render(empty)
}

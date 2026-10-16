package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"tm/internal/metrics"
	"tm/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TickMsg time.Time

type ViewMode int

const (
	ViewModeMain ViewMode = iota
	ViewModeHelp
	ViewModeDetails
)

type MainModel struct {
	CurrentTheme theme.Theme
	styles       theme.Styles

	Stats      metrics.SystemStats
	CPUHistory *metrics.HistoryRingBuffer
	MemHistory *metrics.HistoryRingBuffer
	Processes  []metrics.ProcessInfo

	SelectedIdx  int
	ScrollOffset int
	SortBy       string
	Filter       string

	Width       int
	Height      int
	CommandBar  CommandBar
	Paused      bool
	Mode        ViewMode
	CurrDetails *metrics.ProcessDetails
}

// NewMainModel creates an initialized Bubbletea model.
func NewMainModel(initialSort string, initialFilter string, initialTheme string) MainModel {
	th := theme.GetTheme(initialTheme)
	if initialSort == "" {
		initialSort = "cpu"
	}

	return MainModel{
		CurrentTheme: th,
		styles:       th.MakeStyles(),
		CPUHistory:   metrics.NewHistoryRingBuffer(60),
		MemHistory:   metrics.NewHistoryRingBuffer(60),
		SortBy:       initialSort,
		Filter:       initialFilter,
		CommandBar:   NewCommandBar(),
		Mode:         ViewModeMain,
	}
}

func (m MainModel) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.refreshMetricsCmd(),
		m.tickCmd(),
	)
}

func (m MainModel) tickCmd() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m MainModel) refreshMetricsCmd() tea.Cmd {
	return func() tea.Msg {
		stats, _ := metrics.GetSystemStats()
		procs, _ := metrics.GetProcesses(m.SortBy, m.Filter)
		return MetricsUpdateMsg{
			Stats:     stats,
			Processes: procs,
		}
	}
}

type MetricsUpdateMsg struct {
	Stats     metrics.SystemStats
	Processes []metrics.ProcessInfo
}

type ClearStatusMsg struct{}

func clearStatusCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return ClearStatusMsg{}
	})
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

	case TickMsg:
		if !m.Paused {
			cmds = append(cmds, m.refreshMetricsCmd())
		}
		cmds = append(cmds, m.tickCmd())

	case ClearStatusMsg:
		m.CommandBar.Status = ""

	case MetricsUpdateMsg:
		m.Stats = msg.Stats
		m.Stats.ProcessCount = len(msg.Processes)
		m.CPUHistory.Push(msg.Stats.CPUPercent)
		m.MemHistory.Push(msg.Stats.MemPercent)
		m.Processes = msg.Processes

		// Clamp selection
		if m.SelectedIdx >= len(m.Processes) {
			m.SelectedIdx = len(m.Processes) - 1
		}
		if m.SelectedIdx < 0 {
			m.SelectedIdx = 0
		}

	case tea.KeyMsg:
		// When command bar is active, pass all keys to it
		if m.CommandBar.Mode != ModeNormal {
			switch msg.Type {
			case tea.KeyEsc:
				m.CommandBar.Close()
				return m, nil
			case tea.KeyEnter:
				cmdStr := strings.TrimSpace(m.CommandBar.Input.Value())
				mode := m.CommandBar.Mode
				m.CommandBar.Close()
				return m, m.handleCommand(mode, cmdStr)
			default:
				var cmd tea.Cmd
				m.CommandBar.Input, cmd = m.CommandBar.Input.Update(msg)
				return m, cmd
			}
		}

		// If a modal (Help or Details) is open, handle modal keys first
		if m.Mode != ViewModeMain {
			switch msg.String() {
			case "q", "esc", "enter", "backspace":
				m.Mode = ViewModeMain
				m.CommandBar.Status = "Returned to Main Menu"
				return m, nil
			case "x", "K":
				if m.Mode == ViewModeDetails && m.CurrDetails != nil {
					err := metrics.KillProcess(m.CurrDetails.PID)
					if err != nil {
						m.CommandBar.Status = fmt.Sprintf("ERROR: %v", err)
					} else {
						m.CommandBar.Status = fmt.Sprintf("Terminated: %s (PID: %d)", m.CurrDetails.Name, m.CurrDetails.PID)
						m.Mode = ViewModeMain
						return m, m.refreshMetricsCmd()
					}
				}
				return m, nil
			case "ctrl+c":
				return m, tea.Quit
			default:
				return m, nil
			}
		}

		// Normal Mode Keybindings
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "q":
			// If a status message or search filter is active, pressing 'q' clears it and returns to initial command bar!
			if m.CommandBar.Status != "" || m.Filter != "" {
				m.CommandBar.Status = ""
				m.Filter = ""
				return m, m.refreshMetricsCmd()
			}
			return m, tea.Quit

		case "esc":
			if m.CommandBar.Status != "" || m.Filter != "" {
				m.CommandBar.Status = ""
				m.Filter = ""
				return m, m.refreshMetricsCmd()
			}

		case "?", "h":
			m.Mode = ViewModeHelp
			return m, nil

		case "enter":
			if len(m.Processes) > 0 && m.SelectedIdx < len(m.Processes) {
				target := m.Processes[m.SelectedIdx]
				det, err := metrics.GetProcessDetails(target.PID)
				if err == nil {
					m.CurrDetails = det
					m.Mode = ViewModeDetails
					return m, nil
				}
				m.CommandBar.Status = fmt.Sprintf("Error fetching process details: %v", err)
			}

		case ":":
			return m, m.CommandBar.OpenCommand()

		case "/":
			return m, m.CommandBar.OpenSearch()

		case "up", "k":
			if m.SelectedIdx > 0 {
				m.SelectedIdx--
				if m.SelectedIdx < m.ScrollOffset {
					m.ScrollOffset = m.SelectedIdx
				}
			}

		case "down", "j":
			if m.SelectedIdx < len(m.Processes)-1 {
				m.SelectedIdx++
				maxVisible := m.tableRowsCount()
				if m.SelectedIdx >= m.ScrollOffset+maxVisible {
					m.ScrollOffset = m.SelectedIdx - maxVisible + 1
				}
			}

		case "pgup":
			m.SelectedIdx -= 10
			if m.SelectedIdx < 0 {
				m.SelectedIdx = 0
			}
			m.ScrollOffset = m.SelectedIdx

		case "pgdown":
			m.SelectedIdx += 10
			if m.SelectedIdx >= len(m.Processes) {
				m.SelectedIdx = len(m.Processes) - 1
			}
			maxVisible := m.tableRowsCount()
			if m.SelectedIdx >= m.ScrollOffset+maxVisible {
				m.ScrollOffset = m.SelectedIdx - maxVisible + 1
			}

		case "home", "g":
			m.SelectedIdx = 0
			m.ScrollOffset = 0

		case "end", "G":
			if len(m.Processes) > 0 {
				m.SelectedIdx = len(m.Processes) - 1
				maxVisible := m.tableRowsCount()
				if len(m.Processes) > maxVisible {
					m.ScrollOffset = len(m.Processes) - maxVisible
				}
			}

		case "s": // Cycle sort
			switch m.SortBy {
			case "cpu":
				m.SortBy = "mem"
			case "mem":
				m.SortBy = "pid"
			case "pid":
				m.SortBy = "name"
			default:
				m.SortBy = "cpu"
			}
			m.CommandBar.Status = fmt.Sprintf("Sort: %s", strings.ToUpper(m.SortBy))
			cmds = append(cmds, m.refreshMetricsCmd())

		case "t": // Cycle themes
			themeNames := []string{"default", "nord", "dracula", "matrix", "cyberpunk"}
			currIdx := 0
			for i, name := range themeNames {
				if name == m.CurrentTheme.Name {
					currIdx = i
					break
				}
			}
			nextTheme := themeNames[(currIdx+1)%len(themeNames)]
			m.applyTheme(nextTheme)
			m.CommandBar.Status = fmt.Sprintf("Theme: %s", strings.ToUpper(nextTheme))

		case "x", "K": // Kill selected
			if len(m.Processes) > 0 && m.SelectedIdx < len(m.Processes) {
				target := m.Processes[m.SelectedIdx]
				err := metrics.KillProcess(target.PID)
				if err != nil {
					m.CommandBar.Status = fmt.Sprintf("ERROR: %v", err)
				} else {
					m.CommandBar.Status = fmt.Sprintf("Terminated: %s (PID: %d)", target.Name, target.PID)
					cmds = append(cmds, m.refreshMetricsCmd())
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *MainModel) applyTheme(name string) {
	m.CurrentTheme = theme.GetTheme(name)
	m.styles = m.CurrentTheme.MakeStyles()
}

func (m *MainModel) handleCommand(mode InputMode, input string) tea.Cmd {
	if mode == ModeSearch {
		m.Filter = input
		m.CommandBar.Status = fmt.Sprintf("Filter: '%s'", input)
		return m.refreshMetricsCmd()
	}

	// Command Mode (:)
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	cmd := strings.ToLower(parts[0])
	switch cmd {
	case "q", "quit", "exit":
		return tea.Quit

	case "k", "kill":
		var pid int32
		if len(parts) > 1 {
			p, err := strconv.Atoi(parts[1])
			if err != nil {
				m.CommandBar.Status = "Error: Invalid PID"
				return nil
			}
			pid = int32(p)
		} else if len(m.Processes) > 0 && m.SelectedIdx < len(m.Processes) {
			pid = m.Processes[m.SelectedIdx].PID
		} else {
			m.CommandBar.Status = "Error: Specify PID or select a process"
			return nil
		}

		err := metrics.KillProcess(pid)
		if err != nil {
			m.CommandBar.Status = fmt.Sprintf("Error: %v", err)
		} else {
			m.CommandBar.Status = fmt.Sprintf("Process terminated: PID %d", pid)
		}
		return m.refreshMetricsCmd()

	case "s", "sort":
		if len(parts) > 1 {
			m.SortBy = strings.ToLower(parts[1])
			m.CommandBar.Status = fmt.Sprintf("Sort: %s", strings.ToUpper(m.SortBy))
			return m.refreshMetricsCmd()
		}
		m.CommandBar.Status = "Usage: :sort cpu|mem|name|pid"

	case "t", "theme":
		if len(parts) > 1 {
			m.applyTheme(parts[1])
			m.CommandBar.Status = fmt.Sprintf("Theme: %s", strings.ToUpper(parts[1]))
		} else {
			m.CommandBar.Status = "Usage: :theme default|nord|dracula|matrix|cyberpunk"
		}

	case "filter", "f":
		if len(parts) > 1 {
			m.Filter = parts[1]
			m.CommandBar.Status = fmt.Sprintf("Filter: '%s'", m.Filter)
		} else {
			m.Filter = ""
			m.CommandBar.Status = "Filter cleared"
		}
		return m.refreshMetricsCmd()

	case "clear":
		m.Filter = ""
		m.CommandBar.Status = "Filter reset"
		return m.refreshMetricsCmd()

	case "pause":
		m.Paused = true
		m.CommandBar.Status = "Monitoring paused"

	case "resume":
		m.Paused = false
		m.CommandBar.Status = "Monitoring resumed"
		return m.refreshMetricsCmd()

	case "help", "h", "?":
		m.Mode = ViewModeHelp
		return nil

	case "info", "details", "d":
		var pid int32
		if len(parts) > 1 {
			p, err := strconv.Atoi(parts[1])
			if err != nil {
				m.CommandBar.Status = "Error: Invalid PID"
				return nil
			}
			pid = int32(p)
		} else if len(m.Processes) > 0 && m.SelectedIdx < len(m.Processes) {
			pid = m.Processes[m.SelectedIdx].PID
		} else {
			m.CommandBar.Status = "Error: Specify PID or select a process"
			return nil
		}

		det, err := metrics.GetProcessDetails(pid)
		if err != nil {
			m.CommandBar.Status = fmt.Sprintf("Error: %v", err)
			return nil
		}
		m.CurrDetails = det
		m.Mode = ViewModeDetails
		return nil

	default:
		m.CommandBar.Status = fmt.Sprintf("Unknown command: %s (Type :help for commands)", cmd)
	}

	if m.CommandBar.Status != "" {
		return clearStatusCmd()
	}

	return nil
}

func (m MainModel) tableRowsCount() int {
	// Total height minus Header (~6 lines), Table header (2 lines), Footer (2 lines)
	rows := m.Height - 10
	if rows < 5 {
		return 5
	}
	return rows
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func (m MainModel) View() string {
	if m.Width == 0 || m.Height == 0 {
		return "Loading..."
	}

	if m.Mode == ViewModeHelp {
		return m.renderHelpModal()
	}

	if m.Mode == ViewModeDetails {
		return m.renderDetailsModal()
	}

	var sb strings.Builder

	// 1. App Title Bar
	title := m.styles.Title.Render("TaskMaster")
	sortInfo := m.styles.Dim.Render(fmt.Sprintf(" Sort: [%s]", strings.ToUpper(m.SortBy)))
	filterInfo := ""
	if m.Filter != "" {
		filterInfo = m.styles.Warning.Render(fmt.Sprintf(" Filter: '%s'", m.Filter))
	}
	pausedInfo := ""
	if m.Paused {
		pausedInfo = m.styles.Danger.Render(" [PAUSED]")
	}
	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, title, sortInfo, filterInfo, pausedInfo) + "\n")

	// 2. Metrics Header Cards
	halfWidth := (m.Width / 2) - 3
	if halfWidth < 20 {
		halfWidth = 20
	}

	barWidth := halfWidth - 4
	if barWidth < 10 {
		barWidth = 10
	}
	if barWidth > 35 {
		barWidth = 35
	}

	// CPU Card
	cpuBar := RenderProgressBar(m.Stats.CPUPercent, barWidth, m.styles.Success, m.styles.Dim)
	cpuContent := fmt.Sprintf("%s %5.1f%%\n%s",
		m.styles.CardLabel.Render("CPU USAGE:"),
		m.Stats.CPUPercent,
		cpuBar,
	)
	cpuCard := m.styles.HeaderCard.Width(halfWidth).Render(cpuContent)

	// Memory Card
	memBar := RenderProgressBar(m.Stats.MemPercent, barWidth, m.styles.CardValue, m.styles.Dim)
	memContent := fmt.Sprintf("%s %5.1f%% (%s / %s)\n%s",
		m.styles.CardLabel.Render("MEMORY (RAM):"),
		m.Stats.MemPercent,
		formatBytes(m.Stats.MemUsed),
		formatBytes(m.Stats.MemTotal),
		memBar,
	)
	memCard := m.styles.HeaderCard.Width(halfWidth).Render(memContent)

	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cpuCard, memCard) + "\n\n")

	// 3. Process Table Header
	pidWidth := 8
	cpuWidth := 9
	memWidth := 13
	nameWidth := m.Width - pidWidth - cpuWidth - memWidth - 8
	if nameWidth < 15 {
		nameWidth = 15
	}

	headerText := fmt.Sprintf(" %-*s %-*s %*s %*s ",
		pidWidth, "PID",
		nameWidth, "PROCESS NAME",
		cpuWidth, "CPU %",
		memWidth, "RAM (MB)",
	)
	sb.WriteString(m.styles.TableHeader.Render(headerText) + "\n")

	// 4. Process Table Rows
	maxRows := m.tableRowsCount()
	start := m.ScrollOffset
	end := start + maxRows
	if end > len(m.Processes) {
		end = len(m.Processes)
	}

	if len(m.Processes) == 0 {
		sb.WriteString(m.styles.Dim.Render("  No matching processes found...\n"))
	} else {
		for i := start; i < end; i++ {
			p := m.Processes[i]
			name := p.Name
			if len(name) > nameWidth {
				name = name[:nameWidth-1] + "…"
			}

			rowStr := fmt.Sprintf(" %-*d %-*s %*.1f%% %*s ",
				pidWidth, p.PID,
				nameWidth, name,
				cpuWidth-1, p.CPUPercent,
				memWidth, formatBytes(p.MemoryBytes),
			)

			if i == m.SelectedIdx {
				sb.WriteString(m.styles.TableRowSel.Render(rowStr) + "\n")
			} else {
				sb.WriteString(m.styles.TableRow.Render(rowStr) + "\n")
			}
		}
	}

	// 5. Fill remaining vertical space
	renderedRows := end - start
	for i := renderedRows; i < maxRows; i++ {
		sb.WriteString("\n")
	}

	// 6. Bottom Command Bar
	sb.WriteString(m.CommandBar.View(m.styles, m.Width))

	return sb.String()
}

func (m MainModel) renderHelpModal() string {
	boxWidth := 66
	if m.Width < boxWidth+4 {
		boxWidth = m.Width - 4
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		m.styles.Title.Render(" TaskMaster - Command & Shortcut Guide "),
		"",
		m.styles.CardLabel.Render("NAVIGATION:"),
		"  ↑ / k, ↓ / j     Move cursor up / down",
		"  PgUp / PgDn      Scroll 10 rows",
		"  Home / End       Jump to top / bottom",
		"  Enter            Inspect selected process details",
		"",
		m.styles.CardLabel.Render("KEY SHORTCUTS:"),
		"  /                Live search / filter processes",
		"  :                Open command bar",
		"  s                Cycle sort (CPU -> RAM -> PID -> Name)",
		"  t                Cycle color theme",
		"  x                Kill highlighted process immediately",
		"  ? / h            Open this help guide",
		"",
		m.styles.CardLabel.Render("IN-APP COMMANDS (:):"),
		"  :kill <pid>      Terminate process by PID",
		"  :info [pid]      View comprehensive process details",
		"  :sort <field>    Sort by cpu, mem, name, or pid",
		"  :theme <name>    Switch theme (default, nord, matrix, etc.)",
		"  :filter <query>  Filter processes by name or PID",
		"  :pause / :resume Pause or resume live metrics refresh",
		"  :q, :quit        Exit application",
		"",
		m.styles.Success.Render("  [q / ESC] Return to Main Menu"),
	)

	modal := m.styles.BorderBox.
		Width(boxWidth).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, modal)
}

func (m MainModel) renderDetailsModal() string {
	if m.CurrDetails == nil {
		return ""
	}
	d := m.CurrDetails
	boxWidth := 68
	if m.Width < boxWidth+4 {
		boxWidth = m.Width - 4
	}

	cmdline := d.Cmdline
	if cmdline == "" {
		cmdline = "(None / Access Restricted)"
	}
	if len(cmdline) > 120 {
		cmdline = cmdline[:117] + "..."
	}

	exe := d.Exe
	if exe == "" {
		exe = "(Access Restricted)"
	}
	if len(exe) > 120 {
		exe = exe[:117] + "..."
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		m.styles.Title.Render(fmt.Sprintf(" Process Details: %s (PID: %d) ", d.Name, d.PID)),
		"",
		fmt.Sprintf("  %-16s %d", m.styles.CardLabel.Render("Process ID:"), d.PID),
		fmt.Sprintf("  %-16s %d", m.styles.CardLabel.Render("Parent PID:"), d.PPID),
		fmt.Sprintf("  %-16s %.1f%%", m.styles.CardLabel.Render("CPU Usage:"), d.CPUPercent),
		fmt.Sprintf("  %-16s %s", m.styles.CardLabel.Render("Memory (RAM):"), formatBytes(d.MemoryBytes)),
		fmt.Sprintf("  %-16s %d", m.styles.CardLabel.Render("Threads:"), d.NumThreads),
		"",
		m.styles.CardLabel.Render("Executable Path:"),
		"  "+m.styles.Dim.Render(exe),
		"",
		m.styles.CardLabel.Render("Command Line:"),
		"  "+m.styles.Dim.Render(cmdline),
		"",
		lipgloss.JoinHorizontal(lipgloss.Center,
			m.styles.Success.Render("[q / ESC] Return to Main Menu"),
			"    ",
			m.styles.Danger.Render("[x] Kill Process"),
		),
	)

	modal := m.styles.BorderBox.
		Width(boxWidth).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, modal)
}

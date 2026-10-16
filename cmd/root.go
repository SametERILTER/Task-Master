package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"tm/internal/ui"
)

var (
	sortFlag   string
	filterFlag string
	themeFlag  string
)

var rootCmd = &cobra.Command{
	Use:   "tm",
	Short: "tm - Lightweight and live terminal task manager",
	Long: `TaskMaster (tm) - A lightweight terminal task manager built with Go,
featuring rich graphics, interactive command line, and customizable color themes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		p := tea.NewProgram(
			ui.NewMainModel(sortFlag, filterFlag, themeFlag),
			tea.WithAltScreen(),
		)
		_, err := p.Run()
		return err
	},
}

// Execute runs the root CLI command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&sortFlag, "sort", "s", "cpu", "Sort processes by field (cpu, mem, pid, name)")
	rootCmd.PersistentFlags().StringVarP(&filterFlag, "filter", "f", "", "Filter processes by name or PID")
	rootCmd.PersistentFlags().StringVarP(&themeFlag, "theme", "t", "default", "Color theme (default, nord, dracula, matrix, cyberpunk)")
}

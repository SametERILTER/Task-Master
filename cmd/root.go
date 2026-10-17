package cmd

import (
	"fmt"
	"os"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"tm/internal/ui"
)

var (
	sortFlag   string
	filterFlag string
	themeFlag  string
	listFlag   int
	portFlag   string
	treeFlag   bool
)

var rootCmd = &cobra.Command{
	Use:   "tm",
	Short: "tm - Lightweight and live terminal task manager",
	Long: `TaskMaster (tm) - A lightweight terminal task manager built with Go,
featuring rich graphics, interactive command line, and customizable color themes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("port") {
			var p uint32
			if portFlag != "" && portFlag != "all" {
				if n, err := strconv.Atoi(portFlag); err == nil && n > 0 {
					p = uint32(n)
				}
			}
			return printPortList(p)
		}

		if cmd.Flags().Changed("list") || listFlag > 0 {
			if listFlag <= 0 {
				listFlag = 20
			}
			return printProcessList(listFlag)
		}

		p := tea.NewProgram(
			ui.NewMainModel(sortFlag, filterFlag, themeFlag, treeFlag),
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
	rootCmd.Flags().IntVarP(&listFlag, "list", "l", 0, "Print top N processes snapshot and exit (e.g. tm --list or tm -l 10)")
	rootCmd.Flags().StringVarP(&portFlag, "port", "p", "", "Show open listening ports (e.g. tm --port or tm -p 3000)")
	rootCmd.Flags().BoolVarP(&treeFlag, "tree", "T", false, "Start in hierarchical process tree view mode")

	rootCmd.Flags().Lookup("list").NoOptDefVal = "20"
	rootCmd.Flags().Lookup("port").NoOptDefVal = "all"
}

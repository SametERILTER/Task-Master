package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"tm/internal/metrics"
)

var topCount int

func printProcessList(limit int) error {
	var procs []metrics.ProcessInfo
	var err error
	if treeFlag {
		procs, err = metrics.GetProcessTree(sortFlag, filterFlag)
	} else {
		procs, err = metrics.GetProcesses(sortFlag, filterFlag)
	}
	if err != nil {
		return err
	}

	if limit <= 0 || limit > len(procs) {
		limit = len(procs)
	}

	fmt.Printf("%-8s %-38s %10s %14s\n", "PID", "PROCESS NAME", "CPU %", "RAM (MB)")
	fmt.Println("----------------------------------------------------------------------")
	for i := 0; i < limit; i++ {
		p := procs[i]
		name := p.Name
		if treeFlag && p.TreePrefix != "" {
			name = p.TreePrefix + p.Name
		}
		runes := []rune(name)
		if len(runes) > 38 {
			name = string(runes[:35]) + "..."
		}
		ramMB := float64(p.MemoryBytes) / (1024 * 1024)
		fmt.Printf("%-8d %-38s %9.1f%% %12.1f MB\n", p.PID, name, p.CPUPercent, ramMB)
	}
	return nil
}

var listCmd = &cobra.Command{
	Use:   "list [count]",
	Short: "Print a snapshot process table to terminal and exit",
	RunE: func(cmd *cobra.Command, args []string) error {
		limit := topCount
		if len(args) > 0 {
			if n, err := strconv.Atoi(args[0]); err == nil && n > 0 {
				limit = n
			}
		}
		return printProcessList(limit)
	},
}

func init() {
	listCmd.Flags().IntVarP(&topCount, "top", "n", 20, "Number of top resource-consuming processes to display")
	listCmd.Flags().BoolVarP(&treeFlag, "tree", "T", false, "Display process snapshot in hierarchical tree view")
	rootCmd.AddCommand(listCmd)
}


package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"tm/internal/metrics"
)

var topCount int

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Print a snapshot process table to terminal and exit",
	RunE: func(cmd *cobra.Command, args []string) error {
		procs, err := metrics.GetProcesses(sortFlag, filterFlag)
		if err != nil {
			return err
		}

		limit := topCount
		if limit <= 0 || limit > len(procs) {
			limit = len(procs)
		}

		fmt.Printf("%-8s %-30s %10s %14s\n", "PID", "PROCESS NAME", "CPU %", "RAM (MB)")
		fmt.Println("----------------------------------------------------------------")
		for i := 0; i < limit; i++ {
			p := procs[i]
			name := p.Name
			if len(name) > 30 {
				name = name[:27] + "..."
			}
			ramMB := float64(p.MemoryBytes) / (1024 * 1024)
			fmt.Printf("%-8d %-30s %9.1f%% %12.1f MB\n", p.PID, name, p.CPUPercent, ramMB)
		}
		return nil
	},
}

func init() {
	listCmd.Flags().IntVarP(&topCount, "top", "n", 20, "Number of top resource-consuming processes to display")
	rootCmd.AddCommand(listCmd)
}
